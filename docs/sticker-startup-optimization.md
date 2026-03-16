# 贴纸服务启动优化文档

## 背景与问题

新部署后，第一批用户请求 `messages.getFeaturedStickers` 时会触发大量贴纸文件的同步下载，导致 CPU 和内存在启动后短时间内急剧飙升。具体原因如下：

1. **冷启动同步阻塞**：配置的 `FeaturedStickerSets`（如 `UtyaDuck`、`Animals`）未预热，首次请求在 gRPC 处理线程中同步触发完整的 Bot API 拉取 + DFS 上传流程。
2. **并发请求重复下载**：多个用户同时请求同一个未缓存的贴纸集，每次请求各自独立发起完整下载，资源消耗成倍叠加。
3. **单次下载并发度过高**：每个贴纸集最多启动 10 个并发 worker 同时从 Telegram Bot API 拉取文件，下载期间 CPU/内存峰值集中。

---

## 解决方案概览

本次优化包含三项相互配合的改动：

| 改动 | 目的 |
|------|------|
| 后台预热（warm-up goroutine） | 服务启动后在后台静默缓存 featured sticker sets，消除冷启动时的用户请求阻塞 |
| singleflight 去重 | 并发请求同一未缓存贴纸集时，只发起一次下载，所有请求共享结果 |
| 下载并发数 10 → 3 | 降低单次贴纸集下载对 CPU 和内存的瞬时压力 |

---

## 详细说明

### 1. 后台预热（Background Warm-up）

**文件**：`app/bff/stickers/internal/server/grpc/service/service.go`

服务启动时，`New()` 会立即启动一个后台 goroutine `warmupFeaturedSetsInBackground()`：

```
service.New() 返回
    ↓
    └─ go warmupFeaturedSetsInBackground()
           ↓
           sleep 5s（等待 idgen / media / dfs 上游服务就绪）
           ↓
           for each name in FeaturedStickerSets:
               if already cached → skip
               else → FetchAndCacheStickerSet(name)
               sleep 10s（拉取下一个集前等待）
           ↓
           warmup complete
```

**关键参数**（常量，定义于 `service.go`）：

| 常量 | 默认值 | 含义 |
|------|--------|------|
| `warmupStartDelay` | 5s | 等待上游服务就绪的延迟 |
| `warmupInterSetDelay` | 10s | 相邻贴纸集下载之间的间隔 |

**效果**：
- 首个用户请求到达时 featured sticker sets 已在 DB 中，不再触发任何 Bot API 调用。
- 每两个集之间间隔 10s，避免集中下载造成 CPU/内存脉冲。
- 重启时已缓存的集直接跳过，重启预热时间接近零。

---

### 2. singleflight 并发去重

**文件**：
- `app/bff/stickers/internal/dao/dao.go`（新增 `FetchGroup singleflight.Group`）
- `app/bff/stickers/internal/core/messages.getStickerSet_handler.go`（包装逻辑）

```go
// dao.go
type Dao struct {
    ...
    FetchGroup singleflight.Group // 对同一 shortName 的并发下载请求进行去重
}
```

下载入口函数调用链：

```
FetchAndCacheStickerSet(shortName)          ← 对外导出，供 warm-up 调用
    └─ fetchAndCacheStickerSet(shortName)   ← 内部，带 singleflight 包装
           └─ FetchGroup.Do(shortName, fn)
                  └─ doFetchAndCacheStickerSet(shortName)  ← 实际下载逻辑
```

`singleflight.Group.Do` 的语义：同一时刻对同一 key（贴纸集短名）只执行一次 `fn`，其余并发调用者等待并共享同一结果。下载完成后 key 自动释放，后续请求直接命中 MySQL 缓存。

**效果**：即使在预热完成前有大量并发请求同一个集，也只触发一次 Bot API + DFS 流程。

---

### 3. 下载并发数降低（10 → 3）

**文件**：`app/bff/stickers/internal/dao/download.go`

```go
const (
    filePartSize    = 512 * 1024 // 512KB per part
    downloadWorkers = 3          // 保守并发数，平滑 CPU/内存使用
)
```

原来每个贴纸集同时启动 10 个 goroutine 并发下载文件，改为 3 个。对于一个包含 50 张贴纸的集合：

| 并发数 | 并发 goroutine 数 | 近似下载时间 | 峰值内存占用 |
|--------|-------------------|-------------|-------------|
| 10（旧）| 10 | 较短 | 高 |
| 3（新） | 3 | 适中 | 低 |

下载时间略有增加，但由于预热在后台进行（不阻塞用户请求），这一延迟对用户完全透明。

---

## 整体效果对比

| 场景 | 优化前 | 优化后 |
|------|--------|--------|
| 首次部署，第一个用户请求 featured stickers | 同步阻塞下载所有 featured sets，CPU/内存飙升 | warm-up 已完成，直接读 MySQL，无阻塞 |
| 多用户同时请求同一未缓存集 | 每个请求独立触发完整下载（N 倍开销） | singleflight 保证只下载一次 |
| 单个集的文件下载阶段 | 10 个并发 worker，内存峰值高 | 3 个并发 worker，内存平稳 |
| 服务重启（数据已缓存） | 与首次部署相同（内存依然飙升） | warm-up 跳过已缓存集，几乎零开销 |

---

## 配置说明

`teamgramd/etc/bff.yaml` 中的 `FeaturedStickerSets` 配置保持不变：

```yaml
FeaturedStickerSets:
  - "UtyaDuck"
  - "Animals"
```

- 列表为空时，warm-up goroutine 立即退出并记录日志，不影响服务启动。
- 列表中的集未在 Telegram 上存在时，Bot API 返回错误，warm-up 跳过该集并继续处理下一个。

---

## 关键文件清单

| 文件 | 修改内容 |
|------|---------|
| `app/bff/stickers/internal/server/grpc/service/service.go` | 新增 `warmupFeaturedSetsInBackground()` 及相关常量 |
| `app/bff/stickers/internal/dao/dao.go` | 新增 `FetchGroup singleflight.Group` 字段 |
| `app/bff/stickers/internal/core/messages.getStickerSet_handler.go` | 引入 `fetchAndCacheStickerSet` singleflight 包装层；导出 `FetchAndCacheStickerSet` |
| `app/bff/stickers/internal/dao/download.go` | `downloadWorkers` 从 10 降至 3 |

---

## 日志示例

正常启动预热的日志输出（级别 INFO）：

```
warmupFeaturedSets - scheduled for 2 sets: [UtyaDuck Animals]
warmupFeaturedSets - fetching UtyaDuck (1/2)
doFetchAndCacheStickerSet(UtyaDuck) - got 47 stickers from Bot API in 1.2s
DownloadAndUploadStickerFiles - start: 47 stickers, workers=3
DownloadAndUploadStickerFiles - SUCCESS: 47 stickers in 38.4s
doFetchAndCacheStickerSet(UtyaDuck) - DONE: 47 docs, total=39.6s
warmupFeaturedSets - UtyaDuck cached successfully
warmupFeaturedSets - fetching Animals (2/2)
...
warmupFeaturedSets - warmup complete
```

重启时（已缓存）：

```
warmupFeaturedSets - scheduled for 2 sets: [UtyaDuck Animals]
warmupFeaturedSets - UtyaDuck already cached, skipping
warmupFeaturedSets - Animals already cached, skipping
warmupFeaturedSets - warmup complete
```

singleflight 命中时（极少见于预热完成后）：

```
doFetchAndCacheStickerSet - set UtyaDuck already cached by another request, falling back
```
