# Stickers 代理模块 (`app/bff/stickers`)

> 通过 Telegram Bot API 代理获取官方贴纸包数据，同步下载贴纸文件到 DFS 存储，缓存到本地数据库。

---

## 功能概述

当客户端调用 `messages.getStickerSet` 时：

1. **查本地缓存** — 在 `teamgram_stickers` 数据库中查找该贴纸集
2. **缓存命中** — 从 `sticker_sets` + `sticker_set_documents` 表反序列化 Document protobuf，直接返回
3. **缓存未命中** — 通过 Telegram Bot API 获取元数据，**同步下载所有贴纸文件到 DFS（MinIO）**，写入数据库，返回给客户端

```
客户端 → BFF(gRPC) → StickersCore
                         ├─ 命中 → MySQL(sticker_sets + sticker_set_documents) → 返回
                         └─ 未命中 → Bot API getStickerSet
                                        → 并发下载所有文件 (3 workers)
                                        → DFS 上传 (MinIO)
                                        → 写入 MySQL
                                        → 返回 (DFS 真实 docId)
```

---

## 文件结构

```
app/bff/stickers/
├── helper.go                                    # 对外工厂函数 New(Config)
├── plugin/plugin.go                             # 插件接口占位
└── internal/
    ├── config/config.go                         # 配置定义
    ├── svc/service_context.go                   # ServiceContext (Config + Dao)
    ├── core/
    │   ├── core.go                              # StickersCore 基础结构
    │   └── messages.getStickerSet_handler.go     # 核心业务逻辑
    ├── dao/
    │   ├── dao.go                               # Dao 聚合 (MySQL + IDGen + Media + DFS + BotAPI)
    │   ├── mysql.go                             # MySQL wrapper
    │   ├── botapi.go                            # Telegram Bot API HTTP 客户端
    │   └── download.go                          # 同步文件下载+DFS上传管线
    ├── dal/
    │   ├── dataobject/
    │   │   ├── sticker_sets_do.go               # sticker_sets 表数据对象
    │   │   └── sticker_set_documents_do.go      # sticker_set_documents 表数据对象
    │   └── dao/mysql_dao/
    │       ├── sticker_sets_dao.go              # sticker_sets CRUD
    │       └── sticker_set_documents_dao.go     # sticker_set_documents CRUD
    └── server/grpc/
        ├── grpc.go                              # gRPC 服务注册
        └── service/
            ├── service.go                       # Service 结构
            └── stickers_service_impl.go         # RPCStickersServer 30 个方法实现
```

---

## 数据库

使用独立数据库 `teamgram_stickers`，SQL 文件：
- **独立脚本**: `teamgramd/sql/stickers.sql`
- **自动部署**: 已合入 `teamgramd/sql/1_teamgram.sql` 尾部

### sticker_sets

| 字段 | 类型 | 说明 |
|------|------|------|
| set_id | BIGINT | 本地生成的贴纸集 ID（IDGen snowflake） |
| access_hash | BIGINT | 随机生成的 access hash |
| short_name | VARCHAR(128) | 贴纸集短名（唯一索引，如 `UtyaDuck`） |
| title | VARCHAR(256) | 贴纸集标题 |
| sticker_type | VARCHAR(32) | `regular` / `mask` / `custom_emoji` |
| is_animated | TINYINT | 是否 TGS 动画贴纸 |
| is_video | TINYINT | 是否 WebM 视频贴纸 |
| sticker_count | INT | 贴纸数量 |
| data_json | MEDIUMTEXT | Bot API 原始 JSON 响应（调试用） |
| fetched_at | BIGINT | 抓取时间戳 |

### sticker_set_documents

| 字段 | 类型 | 说明 |
|------|------|------|
| set_id | BIGINT | 所属贴纸集 ID |
| document_id | BIGINT | DFS 分配的文档 ID（唯一索引） |
| sticker_index | INT | 贴纸在集合中的顺序 |
| emoji | VARCHAR(64) | 对应的 emoji |
| bot_file_id | VARCHAR(512) | Bot API file_id（用于下载） |
| bot_file_unique_id | VARCHAR(256) | Bot API file_unique_id |
| bot_thumb_file_id | VARCHAR(512) | 缩略图的 Bot API file_id |
| document_data | MEDIUMTEXT | base64 编码的 protobuf 序列化 Document（缓存恢复用） |
| file_downloaded | TINYINT | 文件是否已下载到 DFS（同步模式下插入时始终为 1） |

---

## 配置

在 `teamgramd/etc/bff.yaml` 中添加：

```yaml
TelegramBotToken: "你的Bot Token"
StickersMysql:
  DSN: root:password@tcp(127.0.0.1:3306)/teamgram_stickers?charset=utf8mb4&parseTime=true
```

### BFF 注册逻辑

`app/bff/bff/internal/server/server.go` 中，当 `TelegramBotToken` 非空时，自动注册 `RPCStickersServer`：

```go
if c.TelegramBotToken != "" {
    mtproto.RegisterRPCStickersServer(grpcServer, stickers_helper.New(stickers_helper.Config{
        TelegramBotToken: c.TelegramBotToken,
        Mysql:            c.StickersMysql,
        IdgenClient:      c.IdgenClient,
        MediaClient:      c.MediaClient,
        DfsClient:        c.DfsClient,
    }))
}
```

不配置 `TelegramBotToken` 则不注册，不影响原有服务。

`app/bff/bff/internal/config/config.go` 中对应字段均标记为 `json:",optional"`。

---

## 依赖的内部服务

| 服务 | 用途 |
|------|------|
| IDGen | `NextId()` 生成 sticker set ID 和临时文件 ID |
| DFS | `DfsWriteFilePartData()` 写入临时分片 + `DfsUploadDocumentFileV2()` 最终写入 MinIO |
| Media | 预留（当前缓存方案不依赖 media 的 documents 表） |

---

## 核心流程详解

### 1. 首次获取 (fetchAndCacheStickerSet) — 同步下载

```
Bot API getStickerSet?name=xxx
        │
        ▼
为每个 sticker 构建 StickerDownloadInput (mime, attributes)
        │
        ▼
DownloadAndUploadStickerFiles (并发 3 workers):
  ├─ Bot API getFile → file_path
  ├─ Bot API /file/bot{token}/{path} → []byte
  ├─ DFS.WriteFilePartData (512KB 分片，写入 SSDB 临时)
  └─ DFS.UploadDocumentFileV2 → *Document (DFS 分配真实 docId，写入 MinIO)
        │
        ▼
用 DFS 返回的 Document 构建 stickerDocDOs
        │
        ▼
INSERT IGNORE sticker_sets + sticker_set_documents
        │
        ▼
返回 Messages_StickerSet 给客户端（docId 即 MinIO 中的真实文件）
```

### 2. 缓存命中 (buildStickerSetFromCache)

```
SELECT FROM sticker_sets WHERE short_name = ?
SELECT FROM sticker_set_documents WHERE set_id = ?
        │
        ▼
遍历 document_data:
  base64.Decode → proto.Unmarshal → *mtproto.Document
        │
        ▼
构建 StickerPack[] (emoji → document_id 映射)
        │
        ▼
返回 Messages_StickerSet 给客户端
```

---

## RPCStickersServer 方法实现状态

| 方法 | 状态 |
|------|------|
| `MessagesGetStickerSet` | **已实现** — 代理查询 + 缓存 |
| 其他查询方法（GetStickers, GetAllStickers, GetFeaturedStickers 等） | 返回空结果 |
| 写入方法（InstallStickerSet, CreateStickerSet 等） | 返回 `ErrMethodNotImpl` |

---

## MIME 类型映射

| 贴纸类型 | MIME | 文件扩展名 |
|----------|------|-----------|
| 普通贴纸 | `image/webp` | `.webp` |
| TGS 动画 | `application/x-tgsticker` | `.tgs` |
| WebM 视频 | `video/webm` | `.webm` |

---

## 已有数据库时的升级 SQL

如果之前已经建过表但没有 `document_data` 列：

```sql
ALTER TABLE teamgram_stickers.sticker_set_documents
  ADD COLUMN document_data MEDIUMTEXT NOT NULL AFTER bot_thumb_file_id;
```

---

## 注意事项

1. **Bot Token 安全**: Token 配置在 `bff.yaml` 中，不要提交到公开仓库
2. **ID 体系独立**: 本地 `set_id` 由 IDGen 生成；`document_id` 由 DFS 服务内部 IDGen 生成，与 Telegram 官方 ID 无关
3. **不写 media 的 documents 表**: Document 序列化后直接存在 `sticker_set_documents.document_data`，不依赖 media 服务持久化
4. **首次请求较慢**: 因为需要同步下载所有贴纸文件，首次请求耗时可能较长（取决于贴纸数量和网络），但贴纸图片可立即显示
5. **贴纸集不会自动刷新**: 一旦缓存了某个贴纸集，后续请求始终返回缓存数据。如需刷新，需手动删除 `sticker_sets` 中对应的行
6. **并发安全**: 多个客户端同时请求同一个未缓存的贴纸集时，使用 `INSERT IGNORE` + `rowsAffected==0` 回退机制，只有一个请求会完成下载，其余读取已缓存数据
