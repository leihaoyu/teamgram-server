# Sticker 搜索/推荐 API 实现文档

> 涉及 3 个 API：`messages.getStickers`、`messages.getFeaturedStickers`、`messages.searchStickerSets`

---

## 1. 整体架构

```
客户端请求
    ↓
RPCStickers (session.yaml 路由)
    ↓
stickers_service_impl.go → core.New() → handler
    ↓                                      ↓
    ↓                            DAO 查询本地 MySQL 缓存
    ↓                                      ↓
    ↓                     (缓存未命中时) fetchAndCacheStickerSet
    ↓                                      ↓
    ↓                           Bot API getStickerSet
    ↓                                      ↓
    ↓                           下载文件 → DFS(MinIO)
    ↓                                      ↓
    ↓                           写入 MySQL 缓存
    ↓                                      ↓
    ←←←←←←←← 返回结果（内部 DFS document_id）
```

**关键点：Bot API file_id 不会暴露给客户端。**
所有贴纸文件经过 `DownloadAndUploadStickerFiles` 下载到 DFS 后，
`document_data` 存储的是序列化的 DFS Document protobuf（包含内部 document_id、access_hash）。
查询时直接反序列化返回，文件 ID 始终一致。

---

## 2. messages.getStickers — 按 emoji 查找贴纸

### 功能
用户在输入框输入 emoji 时，客户端调用此接口获取匹配的贴纸建议。

### 文件
- Handler: `app/bff/stickers/internal/core/messages.getStickers_handler.go`
- DAO: `StickerSetDocumentsDAO.SelectBySetIdsAndEmoji`

### 请求/响应
```
请求: TLMessagesGetStickers { Emoticon: "😂", Hash: int64 }
响应: Messages_Stickers { Hash: int64, Stickers: []*Document }
      或 MessagesStickersNotModified (hash 匹配时)
```

### 流程
```
1. 获取用户已安装的贴纸集 set_ids
   → UserInstalledStickerSetsDAO.SelectByUserAndType(userId, 0)

2. 查询匹配 emoji 的贴纸文档
   → StickerSetDocumentsDAO.SelectBySetIdsAndEmoji(setIds, emoticon)
   SQL: WHERE set_id IN (?,?,...) AND emoji = ?

3. 反序列化每个 document_data → Document protobuf

4. 计算 Telegram hash (combineInt64Hash)
   → 若与请求 hash 匹配，返回 NotModified

5. 返回匹配的 Document 列表
```

### SQL 索引（新增）
```sql
ALTER TABLE sticker_set_documents ADD KEY idx_set_emoji (set_id, emoji);
```
确保 `IN + emoji` 查询走索引，不做全表扫描。

### 数据来源
- **纯本地查询**，不调用 Bot API
- 数据来源于用户之前浏览/安装贴纸集时 `fetchAndCacheStickerSet` 下载并缓存的数据

---

## 3. messages.getFeaturedStickers — 热门贴纸推荐

### 功能
客户端"热门贴纸"/"Trending" tab 显示的推荐贴纸集列表。

### 文件
- Handler: `app/bff/stickers/internal/core/messages.getFeaturedStickers_handler.go`
- DAO: `UserInstalledStickerSetsDAO.SelectPopularSetIds`
- DAO: `StickerSetsDAO.SelectBySetIds`
- DAO: `StickerSetDocumentsDAO.SelectFirstBySetId`
- Config: `config.Config.FeaturedStickerSets`

### 请求/响应
```
请求: TLMessagesGetFeaturedStickers { Hash: int64 }
响应: Messages_FeaturedStickers {
        Count: int32,
        Hash:  int64,
        Sets:  []*StickerSetCovered,  // 每个集包含元数据 + 封面贴纸
        Unread: []int64,              // 暂为空
      }
      或 MessagesFeaturedStickersNotModified
```

### 流程
```
1. 查询安装量最高的贴纸集（最多 20 个）
   → SelectPopularSetIds(limit=20)
   SQL: SELECT set_id FROM user_installed_sticker_sets
        WHERE deleted=0 AND archived=0 AND set_type=0
        GROUP BY set_id ORDER BY COUNT(DISTINCT user_id) DESC LIMIT 20

2. 冷启动补充（安装数据不足时）
   → 读取配置 FeaturedStickerSets（贴纸集短名列表）
   → 正常情况下已由启动时的后台 warm-up goroutine 预先缓存（见 `docs/sticker-startup-optimization.md`）
   → 若 warm-up 尚未完成，对每个短名：
     a. 查本地缓存 SelectByShortName
     b. 未缓存 → fetchAndCacheStickerSet 从 Bot API 拉取并下载到 DFS（singleflight 保证并发安全）
   → 补充到列表直到 20 个

3. 构建 StickerSetCovered
   → SelectBySetIds 批量加载贴纸集元数据
   → SelectFirstBySetId 获取每个集的第一张贴纸作为封面
   → MakeTLStickerSetCovered({ Set, Cover })

4. 计算 hash → NotModified 支持

5. 返回 (Unread 暂为空，后续可加已读追踪表)
```

### 配置（bff.yaml）
```yaml
FeaturedStickerSets:
  - "UtyaDuck"
  - "Animals"
  - "FunnyAnimals"
```
- `json:",optional"` — 不配置也不报错
- 仅在安装量数据不足时触发
- 首次访问会自动通过 Bot API 拉取并缓存

### Bot API 调用时机
- **正常情况下不调用 Bot API**：服务启动时后台 warm-up goroutine 已完成 featured sets 的预缓存（见 `docs/sticker-startup-optimization.md`）
- 仅当 warm-up 尚未完成且集未缓存时才触发按需拉取
- 后续请求完全走本地 MySQL

---

## 4. messages.searchStickerSets — 贴纸集搜索

### 功能
用户在贴纸面板搜索栏输入关键词，搜索贴纸集。

### 文件
- Handler: `app/bff/stickers/internal/core/messages.searchStickerSets_handler.go`
- DAO: `StickerSetsDAO.SearchByQuery`
- 复用: `buildStickerSetsCovered`（来自 getFeaturedStickers handler）

### 请求/响应
```
请求: TLMessagesSearchStickerSets { Q: "duck", Hash: int64, ExcludeFeatured: bool }
响应: Messages_FoundStickerSets {
        Hash: int64,
        Sets: []*StickerSetCovered,
      }
      或 MessagesFoundStickerSetsNotModified
```

### 流程
```
1. 本地 MySQL 模糊搜索
   → SearchByQuery(q, limit=20)
   SQL: WHERE (title LIKE '%duck%' OR short_name LIKE '%duck%')
        ORDER BY sticker_count DESC LIMIT 20

2. 若本地无结果 → Bot API 精确名称回退
   → fetchAndCacheStickerSet(q)
   → 成功则重新查询本地
   （例：搜索 "UtyaDuck" 但本地没这个集 → Bot API 拉取 → 缓存 → 返回）

3. 构建 StickerSetCovered（复用 getFeatured 的 buildStickerSetsCovered）

4. 计算 hash → NotModified 支持
```

### Bot API 调用时机
- **仅在本地搜索无结果时**，尝试将搜索词当作贴纸集短名精确查询
- 本地有结果则不调用 Bot API
- 搜索词不是有效短名时 Bot API 返回 404，静默忽略

---

## 5. 文件 ID 一致性分析

### 为什么不会出现 file_id 不一致？

| 阶段 | ID 类型 | 说明 |
|------|---------|------|
| Bot API 返回 | Bot file_id | 仅用于下载，存到 `bot_file_id` 列备查 |
| DFS 上传后 | 内部 document_id | `MediaUploadedDocumentMedia` 生成，写入 documents 表 |
| DB 存储 | document_data (protobuf) | 序列化的 DFS Document，包含内部 ID |
| 客户端获取 | 反序列化 document_data | **始终返回内部 DFS ID**，与其他接口一致 |

```
Bot API file_id ──→ 下载文件 ──→ DFS 上传 ──→ 内部 document_id
                                                     ↓
                                            proto.Marshal → base64
                                                     ↓
                                            sticker_set_documents.document_data
                                                     ↓
                              getStickers / getFeatured / search 读取
                                                     ↓
                                            base64 → proto.Unmarshal
                                                     ↓
                                            返回 Document（内部 ID）
```

**所有三个新 API 的数据全部来自 MySQL `document_data` 字段，
与 `getStickerSet`、`getRecentStickers` 等已有接口使用完全相同的 Document。**

### 潜在的边缘情况

| 情况 | 风险 | 处理 |
|------|------|------|
| 同一贴纸集被多个请求并发获取 | 无 | `INSERT IGNORE` + `rowsAffected==0` 回退到已缓存数据 |
| Bot API file_id 过期 | 无 | 文件已下载到 DFS，不再依赖 Bot API |
| 贴纸集更新（官方新增贴纸） | 当前不自动刷新 | 后续可加 TTL 重新拉取机制 |
| DFS 上传中途失败 | 整个 set 获取失败返回错误 | 下次请求会重新拉取 |

---

## 6. 新增文件清单

| 操作 | 文件 |
|------|------|
| 新建 | `core/messages.getStickers_handler.go` |
| 新建 | `core/messages.getFeaturedStickers_handler.go` |
| 新建 | `core/messages.searchStickerSets_handler.go` |
| 修改 | `dal/dao/mysql_dao/sticker_set_documents_dao.go` — +`SelectBySetIdsAndEmoji`, +`SelectFirstBySetId` |
| 修改 | `dal/dao/mysql_dao/sticker_sets_dao.go` — +`SearchByQuery`, +`SelectBySetIds` |
| 修改 | `dal/dao/mysql_dao/user_installed_sticker_sets_dao.go` — +`SelectPopularSetIds` |
| 修改 | `config/config.go` — +`FeaturedStickerSets` |
| 修改 | `server/grpc/service/stickers_service_impl.go` — 3 个桩替换为 handler |
| 修改 | `teamgramd/sql/stickers.sql` — +`idx_set_emoji` 索引 |

---

## 7. 部署检查清单

- [ ] 执行 SQL 索引迁移：
  ```sql
  ALTER TABLE sticker_set_documents ADD KEY idx_set_emoji (set_id, emoji);
  ```
- [ ] 在 `bff.yaml` 中配置 `FeaturedStickerSets` 列表（可选，冷启动用）
- [ ] 确认 `TelegramBotToken` 已配置（search 回退和 featured 冷启动需要）
- [ ] 确认 `sticker_set_documents` 表已有数据（用户需要先安装过至少一个贴纸集，getStickers 才有结果）

---

## 8. 后续可优化项

| 优化 | 说明 |
|------|------|
| 贴纸集 TTL 刷新 | 定期重新拉取 Bot API 更新贴纸集内容（新增/删除贴纸） |
| Featured 已读追踪 | 新建 `user_featured_read` 表，填充 `Unread` 字段 |
| 搜索改进 | 支持按 emoji 文字搜索（如搜 "dog" 匹配 🐕 对应贴纸） |
| ExcludeFeatured | searchStickerSets 的 `ExcludeFeatured` flag 目前未处理 |
| 全文索引 | 贴纸集数量大时，LIKE 查询可替换为 FULLTEXT INDEX |

> 已实现的优化：后台预热（启动时平滑下载 FeaturedStickerSets）、singleflight 并发去重、下载并发数调低，详见 `docs/sticker-startup-optimization.md`。
