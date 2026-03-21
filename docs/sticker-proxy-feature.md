# Sticker Proxy Feature - 技术文档

## 概述

通过 Telegram Bot API 代理获取贴纸包数据，流式下载贴纸文件直接写入自有 MinIO 存储（跳过 DFS gRPC 链路），使客户端可以在私有部署的 Teamgram 服务器上查看和使用公共贴纸包。

## 架构

```
客户端 → Gateway → Session → BFF(stickers) → Bot API(获取元数据/流式下载文件)
                                            → MinIO(直接流式写入，跳过 DFS gRPC)
                                            → MySQL(缓存元数据)
```

### 关键路由配置

`teamgramd/etc/session.yaml` 的 IDMap 中必须有：
```yaml
/mtproto.RPCStickers: bff.bff
```

## 贴纸类型

| 类型 | MIME | 文件扩展名 | is_animated | is_video | 状态 |
|------|------|-----------|-------------|----------|------|
| Lottie 动画 | application/x-tgsticker | .tgs | true | false | ✅ 正常 |
| WebP 静态 | image/webp | .webp | false | false | ✅ 正常 |
| WebM 视频 | video/webm | .webm | false | true | ⚠️ 依赖客户端 VP9 解码支持 |

## 核心流程

### 首次请求 `messages.getStickerSet`

```
1. BFF 收到 inputStickerSetShortName("AP_DI2")
2. 查 MySQL 缓存 → 没有
3. 调用 Bot API getStickerSet → 获取贴纸列表和元数据
4. 并发流式下载所有贴纸文件（5 workers, 10 per batch）：
   a. Bot API getFile → 获取 file_path
   b. Bot API DownloadFileStream → HTTP resp.Body (io.ReadCloser)
   c. MinIO PutObject(documents/{docId}.dat, resp.Body) — 流式零拷贝，跳过 DFS gRPC
   d. 缩略图: Bot API DownloadFile(thumb) → MinIO PutObject(photos/m/{docId}.dat)
   e. 缩略图处理: DecodeWebp → Resize(40px) → EncodeStripped → PhotoStrippedSize
   f. BFF 本地构建 Document protobuf (IDGen 生成 docId，本地计算 accessHash)
5. 存储到 MySQL: sticker_sets + sticker_set_documents
6. 返回 messages.StickerSet 响应（包含本地生成的 document ID）
```

### 后续请求（缓存命中）

```
1. 查 MySQL → 找到 sticker_sets 记录
2. 读取 sticker_set_documents → 反序列化 document_data (base64 protobuf)
3. 直接返回响应
```

### 客户端下载文件

```
1. 客户端发送 upload.getFile(inputDocumentFileLocation{id, accessHash})
2. DFS 先查 SSDB 缓存（cache_file_info_{docId} → 2h TTL）
3. 缓存过期后从 MinIO 读取（{docId}.dat）
4. 返回 upload.File{type, bytes}
```

## 最近使用 & 收藏贴纸

### 功能概述

实现了 5 个 MTProto 方法，支持客户端的「最近使用」和「收藏」贴纸面板：

| 方法 | 功能 |
|------|------|
| `messages.getRecentStickers` | 获取用户最近使用的贴纸列表（最多 200 条） |
| `messages.saveRecentSticker` | 保存/移除最近使用的贴纸 |
| `messages.clearRecentStickers` | 清空所有最近使用的贴纸 |
| `messages.getFavedStickers` | 获取用户收藏的贴纸列表（最多 200 条） |
| `messages.faveSticker` | 收藏/取消收藏贴纸 |

### 数据模型

两张 per-user 表存储在 `teamgram_stickers` 数据库中：

```sql
-- 最近使用
CREATE TABLE user_recent_stickers (
  user_id       BIGINT NOT NULL,
  document_id   BIGINT NOT NULL,
  emoji         VARCHAR(64),       -- 从 documentAttributeSticker.Alt 提取
  document_data MEDIUMTEXT,        -- base64 protobuf Document（与 sticker_set_documents 格式相同）
  date2         BIGINT,            -- unix timestamp，用于排序和返回给客户端
  deleted       TINYINT(1),        -- 软删除标志
  UNIQUE KEY (user_id, document_id)
);

-- 收藏
CREATE TABLE user_faved_stickers (
  -- 结构同 user_recent_stickers
);
```

核心设计：`document_data` 存储完整的 `Document` protobuf 序列化结果（base64），Save 时调用 `MediaGetDocument` 获取一次，后续 Get 直接反序列化，无需再调 media 服务。

### 请求/响应流程

**Save (saveRecentSticker / faveSticker)**:
```
1. 客户端发送 InputDocument{id, accessHash}
2. unsave/unfave=true → 软删除 (UPDATE SET deleted=1)
3. unsave/unfave=false:
   a. MediaClient.MediaGetDocument(docId) → 获取完整 Document
   b. 从 documentAttributeSticker.Alt 提取 emoji
   c. SerializeStickerDoc(doc) → base64 protobuf
   d. INSERT ... ON DUPLICATE KEY UPDATE (upsert)
4. 返回 BoolTrue
```

**Get (getRecentStickers / getFavedStickers)**:
```
1. SELECT ... WHERE user_id=? AND deleted=0 ORDER BY date2 DESC LIMIT 200
2. 反序列化每条 document_data → []*Document
3. 按 emoji 分组 → []*StickerPack
4. 计算 hash（fnv64a over document IDs）
5. 如果 request.hash == computed hash → 返回 NotModified
6. 否则返回完整 Messages_RecentStickers / Messages_FavedStickers
```

**Clear (clearRecentStickers)**:
```
UPDATE user_recent_stickers SET deleted=1 WHERE user_id=? AND deleted=0
```

## 安装 & 排序贴纸包

### 功能概述

实现了 4 个 MTProto 方法，支持客户端的「安装/卸载贴纸包」和「排序我的贴纸」功能：

| 方法 | 功能 |
|------|------|
| `messages.installStickerSet` | 安装贴纸包（新增到用户列表顶部，或归档） |
| `messages.uninstallStickerSet` | 卸载贴纸包（软删除） |
| `messages.reorderStickerSets` | 按客户端指定的顺序重新排列贴纸包 |
| `messages.getAllStickers` | 获取用户已安装的所有贴纸包（支持 NotModified） |

### 数据模型

```sql
CREATE TABLE user_installed_sticker_sets (
  user_id        BIGINT NOT NULL,
  set_id         BIGINT NOT NULL,
  set_type       TINYINT(1),        -- 0=regular, 1=masks, 2=emojis
  order_num      INT,               -- 排序序号，越小越靠前
  installed_date BIGINT,            -- unix timestamp
  archived       TINYINT(1),        -- 归档标志
  deleted        TINYINT(1),        -- 软删除标志
  UNIQUE KEY (user_id, set_id),
  KEY (user_id, set_type)
);
```

### 请求/响应流程

**installStickerSet**:
```
1. 解析 InputStickerSet → set_id + set_type
2. archived=true → 直接 upsert（archived=1）
3. archived=false:
   a. 所有同类 set 的 order_num +1（腾出 0 号位）
   b. Upsert 新 set（order_num=0, archived=0）
4. 返回 StickerSetInstallResultSuccess
```

**uninstallStickerSet**:
```
1. 解析 InputStickerSet → set_id
2. UPDATE SET deleted=1 (软删除)
3. 返回 BoolTrue
```

**reorderStickerSets**:
```
1. 根据 Masks/Emojis flag 确定 set_type
2. 按客户端发来的 Order 数组，逐个 UPDATE order_num = 数组下标
3. 返回 BoolTrue
```

**getAllStickers**:
```
1. 查询 user_installed_sticker_sets WHERE set_type=0 AND deleted=0 AND archived=0
2. 计算 hash（fnv64a over set IDs）
3. request.hash == hash → 返回 AllStickersNotModified
4. 否则 JOIN sticker_sets 获取完整元数据，设置 InstalledDate，返回 AllStickers
```

### 关键文件

| 文件 | 用途 |
|------|------|
| `app/bff/stickers/internal/core/messages.installedStickerSets_handler.go` | 4 个方法的核心逻辑 |
| `app/bff/stickers/internal/dal/dao/mysql_dao/user_installed_sticker_sets_dao.go` | 安装贴纸包 DAO |
| `app/bff/stickers/internal/dal/dataobject/user_installed_sticker_sets_do.go` | 安装贴纸包数据对象 |

### NotModified 支持

客户端发送上次收到的 `hash` 值，服务端计算当前 hash（对所有 documentId 做 fnv64a），如果相等返回 `messagesRecentStickersNotModified` / `messagesFavedStickersNotModified`，节省带宽。

### 关键文件

| 文件 | 用途 |
|------|------|
| `app/bff/stickers/internal/core/messages.recentAndFavedStickers_handler.go` | 5 个方法的核心逻辑 |
| `app/bff/stickers/internal/dal/dao/mysql_dao/user_recent_stickers_dao.go` | 最近贴纸 DAO |
| `app/bff/stickers/internal/dal/dao/mysql_dao/user_faved_stickers_dao.go` | 收藏贴纸 DAO |
| `app/bff/stickers/internal/dal/dataobject/user_recent_stickers_do.go` | 最近贴纸数据对象 |
| `app/bff/stickers/internal/dal/dataobject/user_faved_stickers_do.go` | 收藏贴纸数据对象 |

## 关键文件

| 文件 | 用途 |
|------|------|
| `app/bff/stickers/internal/core/messages.getStickerSet_handler.go` | 主处理器：获取/缓存/返回贴纸集 |
| `app/bff/stickers/internal/dao/download.go` | 流式下载 → MinIO 直接上传，Document protobuf 构建 |
| `app/bff/stickers/internal/dao/botapi.go` | Bot API HTTP 客户端（连接池，DownloadFileStream 流式下载） |
| `app/bff/stickers/internal/dao/dao.go` | Dao 聚合（MySQL + IDGen + MinIO Client + BotAPI） |
| `app/bff/stickers/internal/dal/dao/mysql_dao/` | MySQL DAO（sticker_sets, sticker_set_documents, user_recent/faved_stickers, user_installed_sticker_sets） |
| `pkg/imaging/` | 共享图片处理包（DecodeWebp, Resize, EncodeStripped） |
| `app/service/dfs/internal/core/dfs.downloadFile_handler.go` | DFS 文件下载处理器（客户端通过 DFS 读取 MinIO 文件） |

## 已修复的 Bug

### Bug 1: inputStickerSetID 未处理

**现象**: 客户端用 `inputStickerSetID(id, accessHash)` 请求贴纸集时返回错误。

**原因**: handler 只处理了 `inputStickerSetShortName`，没有处理 `inputStickerSetID`。

**修复**: 添加 `case mtproto.Predicate_inputStickerSetID` 分支，根据 `set_id` 查 MySQL 返回缓存数据。

### Bug 2: DFS 文件未注册到 documents 表 → documentEmpty

**现象**: 用户 A 发送贴纸给用户 B，B 收到空白消息（`Document.documentEmpty`）。

**原因**: 下载逻辑直接调用 `DfsClient.DfsUploadDocumentFileV2`，只写入 MinIO 文件，不注册到主 `documents` 表。当 B 的客户端请求 Document 元数据时，`GetDocumentById` 在表中找不到 → 返回 `documentEmpty`。

**修复**: 改为调用 `MediaClient.MediaUploadedDocumentMedia`，它内部先调 `DfsUploadDocumentFileV2` 再调 `SaveDocumentV2`，同时完成文件存储和元数据注册。

### Bug 3: FileReference 为 nil

**现象**: iOS 客户端日志显示 `fileReference: <null>`。

**原因**: `dfs.uploadDocumentFileV2_handler.go` 中 `FileReference: nil`。

**修复**: 改为 `FileReference: []byte{}`（空但非 nil），与 MP4 上传处理器保持一致。

### Bug 4: .webm 文件类型映射缺失 → storage_filePartial

**现象**: 客户端下载 WebM 贴纸时收到 `storage.FileType.filePartial`（文件不完整）。

**原因**: `image_util.go` 的 `GetStorageFileTypeConstructor` 没有 `.webm` case，走 default 返回 `storage_filePartial`。

**修复**: 添加 `.webm` → `CRC32_storage_fileMp4`（MTProto schema 中没有 `storage_fileWebm`，`fileMp4` 是最接近的视频类型）。

**影响**: accessHash 的高 32 位编码了文件类型，所以修复后需要清除旧的贴纸缓存数据让文件重新上传。

### Bug 5: 视频贴纸缺少 documentAttributeVideo

**现象**: 视频贴纸只有 `documentAttributeImageSize`，没有 `documentAttributeVideo`。

**修复**: `buildDocumentAttributes` 对 `IsVideo` 贴纸使用 `documentAttributeVideo`（含 `Nosound: true`, `SupportsStreaming: true`）。

**注意**: 经 iOS 客户端代码验证，iOS 识别视频贴纸不依赖此属性（只看 `mimeType == "video/webm"` + `.Sticker` 属性），但其他客户端可能需要。

### Bug 6: MinIO 写入 0 字节（SSDBReader size=-1）

**现象**: 贴纸文件下载几小时后变为空白（`bytes: <null>`）。MinIO `documents` 桶中文件大小为 0。

**原因**: `PutDocumentFile` 使用 `SSDBReader` 作为 `io.Reader`，MinIO SDK 的 `PutObject(reader, -1)` 对未知大小的流写入 0 字节。SSDB 缓存过期（2-3h）后回退到 MinIO，但 MinIO 文件为空。

**修复**: 先用 `ReadAll()` 将所有数据从 SSDB 读到内存，再用 `bytes.NewReader(cacheData)` 以已知大小上传。同时将 MinIO 写入改为同步（不再使用后台 goroutine），写入失败时返回错误。

### Bug 7: 贴纸缩略图黑底/格式错误

**现象**: 缩略图背景为黑色，或格式不对。

**原因**: WebP 缩略图（含透明通道）被转为 JPEG，alpha 通道丢失。

**修复**: 外部缩略图（Bot API 下载的）直接以原始 WebP 格式存储，保留透明度。内部图片缩略图（isThumb=true 的情况）使用 `FlattenToWhite()` 将透明区域填充为白色后再编码为 JPEG。

### Bug 8: 贴纸下载内存飙涨 + 速度慢

**现象**: Bot API 下载贴纸包时内存飙升，每个 sticker 文件经过多次内存拷贝。100 个贴纸的包需要 100 秒。

**原因**: 下载链路经过 `Bot API → BFF([]byte) → gRPC(Media) → gRPC(DFS) → SSDB → MinIO`，文件字节在进程间多次拷贝，且并发度只有 1。

**修复（v1 — SSDB 绕行）**: 新增 `FileData`/`ThumbData` 字段跳过 SSDB，减少到 2 次拷贝。

**修复（v2 — 流式直传，当前方案）**: 彻底跳过 DFS/Media gRPC 链路。BFF 直接持有 MinIO 客户端，HTTP 下载流（`resp.Body`）通过 `io.Reader` 直接传给 `MinIO.PutObject()`，实现零拷贝流式传输：
- HTTP 响应体流式写入 MinIO，文件数据不在内存中完整缓存
- 5 个并发 worker + 全局信号量限制总并发
- 每批 10 个处理完后触发 `runtime.GC()` + `debug.FreeOSMemory()` 回收内存
- Document protobuf 在 BFF 本地构建（IDGen 生成 docId，本地计算 accessHash）
- 缩略图（小文件）仍使用 `[]byte` 缓冲，下载后直接写入 MinIO `photos` 桶
- 缩略图处理使用 `pkg/imaging` 共享包：DecodeWebp → Resize(40px) → EncodeStripped → PhotoStrippedSize

### Bug 9: 缩略图下载返回 fileJpeg 而非 fileWebp

**现象**: 缩略图文件内容已经是 WebP（`RIFF` 头），但 `upload.File.type` 返回 `storage.FileType.fileJpeg`。

**原因**: `dfs.downloadFile_handler.go` 中 `inputDocumentFileLocation`（thumbSize 非空）、`inputPeerPhotoFileLocation`、`inputStickerSetThumb` 三处硬编码 `sType = CRC32_storage_fileJpeg`，不检测实际文件格式。

**修复**: 新增 `model.DetectStorageFileType(data)` 函数，根据文件头魔数（magic bytes）自动检测：
- `RIFF....WEBP` → `storage_fileWebp`
- `FF D8 FF` → `storage_fileJpeg`
- `89 PNG` → `storage_filePng`
- `GIF8` → `storage_fileGif`
- 其他 → fallback `storage_fileJpeg`

替换了三处硬编码 `fileJpeg`。

### Bug 10: 容器内存飙涨至 1200MB（GOMEMLIMIT 配置错误）

**现象**: 下载贴纸包时，即使只有 20 个贴纸（每个 ~100KB），Docker 容器内存也会从 ~200MB 飙涨到 1200MB（接近 `mem_limit: 1280m`）。

**原因**: `docker-compose.yaml` 设置了 `GOMEMLIMIT=1100MiB` 作为环境变量，被容器内 **11 个独立 Go 进程** 继承。每个进程的 GC scavenger 认为自己可以安全持有 1100MiB 的内存页，不急于归还给 OS。

- 11 个进程 × 1100MiB = 12.1 GiB 理论上限，但容器只有 1280MB
- Go GC 收集对象后，页面标记为空闲但 **不归还 OS**（因为 scavenger 认为 usage << 1100MiB）
- 贴纸下载时 BFF/Media/DFS 三个进程快速分配和释放 gRPC 缓冲区，Runtime 持有的页面累积
- 最终容器总 RSS 撑到 cgroup 限制

**修复**:
1. `docker-compose.yaml`: `GOMEMLIMIT=1100MiB` → `GOMEMLIMIT=100MiB`（全局兜底值）
2. `runall-docker.sh`: 每个进程独立设置 GOMEMLIMIT（BFF=150MiB, DFS=200MiB, Media=100MiB, 其他=50-80MiB）
3. 总预算 ~1000MiB，给 OS/page cache 留 280MiB

**GOMEMLIMIT 分配**:
| 进程 | GOMEMLIMIT | 理由 |
|------|-----------|------|
| dfs | 200MiB | 文件 I/O + 图片处理 + MinIO 上传 |
| bff | 150MiB | 贴纸下载 + gRPC 调用 |
| session | 100MiB | 会话状态管理 |
| gateway | 100MiB | 连接管理 |
| media | 100MiB | gRPC 中转 |
| authsession | 80MiB | 认证会话 |
| biz | 80MiB | 业务逻辑 |
| msg | 80MiB | 消息处理 |
| sync | 80MiB | 同步 |
| idgen | 50MiB | ID 生成（轻量） |
| status | 50MiB | 状态查询（轻量） |

## 已知限制

### 视频贴纸（WebM）依赖客户端 VP9 解码能力

**现象**: 贴纸集正常返回，文件正确下载（有效 EBML 头 `0x1A45DFA3`，大小匹配），但视频贴纸不显示。

**根因**: 客户端的 `SoftwareVideoSource`（VP9 解码器）无法解码文件。`VideoStickerDirectFrameSource.frameCount == 0`，缓存文件只有 20 字节 header，无帧数据。

**排查确认路径**:
1. `isVideoSticker` 返回 `true` ✓
2. 进入视频贴纸渲染分支 ✓
3. 文件下载完整（size 匹配）✓
4. `cacheVideoStickerFrames` 被调用 ✓
5. `SoftwareVideoSource` 解码失败 → `frameCount == 0` ✗

**结论**: 这是客户端解码器问题，不是服务端问题。需要客户端支持 VP9/WebM 解码。

### ~~DFS 后台写入 MinIO~~ (已修复)

~~`DfsUploadDocumentFileV2` 使用 `threading2.WrapperGoFunc` 在后台 goroutine 写入 MinIO~~

**修复**: MinIO 写入改为同步，在返回 Document 之前完成。直接模式（FileData 字段）更是完全跳过 SSDB，文件字节直接从 gRPC 请求写入 MinIO。

### ~~无缩略图~~ (已修复)

~~视频贴纸（video/webm）在 DFS 层不生成缩略图~~

**修复**: 支持外部缩略图（ThumbData 字段），Bot API 下载的缩略图以原始 WebP 格式存储到 MinIO `photos` 桶，保留透明度。Document 的 `thumbs` 包含 `photoStrippedSize` + `photoSize(m)` 两级缩略图。

## 数据库 Schema

### sticker_sets
```sql
CREATE TABLE sticker_sets (
  set_id BIGINT PRIMARY KEY,
  access_hash BIGINT,
  short_name VARCHAR(64) UNIQUE,
  title VARCHAR(128),
  sticker_type VARCHAR(32),
  is_animated TINYINT(1),
  is_video TINYINT(1),
  is_masks TINYINT(1),
  is_emojis TINYINT(1),
  is_official TINYINT(1),
  sticker_count INT,
  hash INT,
  thumb_doc_id BIGINT,
  data_json TEXT,
  fetched_at BIGINT
);
```

### sticker_set_documents
```sql
CREATE TABLE sticker_set_documents (
  set_id BIGINT,
  document_id BIGINT,
  sticker_index INT,
  emoji VARCHAR(32),
  bot_file_id VARCHAR(256),
  bot_file_unique_id VARCHAR(128),
  bot_thumb_file_id VARCHAR(256),
  document_data TEXT,          -- base64 encoded protobuf Document
  file_downloaded TINYINT(1),
  PRIMARY KEY (set_id, document_id)
);
```

### user_recent_stickers / user_faved_stickers
```sql
CREATE TABLE user_recent_stickers (   -- user_faved_stickers 结构相同
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  document_id BIGINT NOT NULL,
  emoji VARCHAR(64),
  document_data MEDIUMTEXT,           -- base64 encoded protobuf Document
  date2 BIGINT,                       -- unix timestamp
  deleted TINYINT(1) DEFAULT 0,       -- 软删除标志
  UNIQUE KEY (user_id, document_id),
  KEY (user_id)
);
```

### user_installed_sticker_sets
```sql
CREATE TABLE user_installed_sticker_sets (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  set_id BIGINT NOT NULL,
  set_type TINYINT(1) DEFAULT 0,      -- 0=regular, 1=masks, 2=emojis
  order_num INT DEFAULT 0,            -- 排序序号
  installed_date BIGINT DEFAULT 0,    -- unix timestamp
  archived TINYINT(1) DEFAULT 0,      -- 归档标志
  deleted TINYINT(1) DEFAULT 0,       -- 软删除标志
  UNIQUE KEY (user_id, set_id),
  KEY (user_id, set_type)
);
```

## 调试指南

### pprof 性能分析

go-zero 内置 DevServer 支持 pprof，已在 BFF/DFS/Media 的 YAML 配置中启用：

| 服务 | pprof 端口 | URL |
|------|-----------|-----|
| BFF | 6061 | `http://localhost:6061/debug/pprof/` |
| DFS | 6062 | `http://localhost:6062/debug/pprof/` |
| Media | 6063 | `http://localhost:6063/debug/pprof/` |

```bash
# 查看堆内存分配 top 20
go tool pprof http://localhost:6061/debug/pprof/heap

# 保存并分析 heap profile
curl -o /tmp/bff-heap.prof http://localhost:6061/debug/pprof/heap
go tool pprof -http=:8080 /tmp/bff-heap.prof

# Docker 容器内查看各进程内存
docker exec <container> ps aux --sort=-rss | head -20
```

### 服务端日志
```bash
docker exec -it <container> tail -f /app/logs/bff/error.log      # BFF 处理器
docker exec -it <container> tail -f /app/logs/dfs/error.log      # DFS 上传/下载
docker exec -it <container> tail -f /app/logs/media/error.log    # Media 服务
```

### 验证 MinIO 文件
```bash
# 提取文件（docId 从客户端日志获取）
docker exec <minio-container> sh -c "cat /data/documents/<docId>.dat" > /tmp/test.webm
file /tmp/test.webm           # 应显示 WebM
xxd -l 32 /tmp/test.webm     # 应以 1a45dfa3 开头
ffprobe /tmp/test.webm       # 查看编解码器
```

### 清除贴纸缓存（重新下载）
```bash
docker exec -i <mysql-container> mysql -u root -p teamgram_stickers -e "
  DELETE FROM sticker_set_documents WHERE set_id IN (SELECT set_id FROM sticker_sets WHERE short_name = 'AP_DI2');
  DELETE FROM sticker_sets WHERE short_name = 'AP_DI2';
"
```

### iOS 客户端调试断点

| 断点位置 | 检查内容 |
|----------|---------|
| `SyncCore_TelegramMediaFile.swift:629` `isVideoSticker` | 确认 `mimeType` 和 `.Sticker` 属性 |
| `StickerPackPreviewGridItem.swift:229` | 确认进入视频贴纸分支 |
| `AnimatedStickerNode.swift:475` `directData.6` | 确认 isVideo=true，directData 非 nil |
| `AnimatedStickerUtils.swift` `cacheVideoStickerFrames` | 确认 VP9 帧提取是否成功 |
| `VideoStickerFrameSource.swift` `frameCount` | 确认解码帧数 > 0 |

### SSDB 缓存 TTL

| Key | TTL | 用途 |
|-----|-----|------|
| `file_{creator}_{fileId}` | 3h | 文件分片数据 |
| `file_info_{creator}_{fileId}` | 3h | 文件元数据 |
| `cache_file_info_{documentId}` | 2h | docId → 原始文件映射 |

**注意**: 贴纸下载使用流式直传模式（BFF 直接写入 MinIO），完全跳过 SSDB 和 DFS gRPC 链路，上述缓存不再适用于贴纸文件。普通文件上传仍使用 SSDB 流程。

## Docker 配置

### GOMEMLIMIT

容器内 11 个 Go 进程共享 `mem_limit: 1280m`。`GOMEMLIMIT` 是**进程级**别的环境变量：

- `docker-compose.yaml` 的 `GOMEMLIMIT=100MiB` 是全局兜底值
- `runall-docker.sh` 中每个进程有独立的 `GOMEMLIMIT=XMiB`（覆盖全局值）
- 总预算 ~1000MiB，留 280MiB 给 OS、page cache、非 Go 内存

# 例如查看 BFF 的堆内存分配
go tool pprof http://localhost:6061/debug/pprof/heap

# 查看 DFS 的 goroutine 堆栈
curl http://localhost:6062/debug/pprof/goroutine?debug=1

**⚠️ 注意**: 不要把 GOMEMLIMIT 设置为接近容器 mem_limit 的值（如 1100MiB），否则每个 Go 进程的 scavenger 都不会积极归还内存页，导致容器 RSS 接近限制。

### entrypoint.sh

`sed` 替换 127.0.0.1 为 Docker 服务名。注意 `&` 在 sed 中是特殊字符，需要转义：
```bash
${STICKERS_MYSQL_URI//&/\\&}
```

### MySQL 初始化

`docker-entrypoint-initdb.d/2_stickers_grant.sh` 授权 `teamgram_stickers.*` 给 `$MYSQL_USER`，仅在首次初始化时执行。

## Proto 构造模式

```go
// 创建 Document
mtproto.MakeTLDocument(&mtproto.Document{
    Id: docId, AccessHash: accessHash, FileReference: []byte{},
    Date: date, MimeType: mimeType,
    Size2_INT32: int32(size), Size2_INT64: size,
    DcId: 1, Attributes: attrs,
}).To_Document()

// 创建 documentAttributeSticker
mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{
    Alt: emoji,
    Stickerset: mtproto.MakeTLInputStickerSetID(&mtproto.InputStickerSet{
        Id: setId, AccessHash: setAccessHash,
    }).To_InputStickerSet(),
}).To_DocumentAttribute()

// 创建 documentAttributeVideo（视频贴纸）
mtproto.MakeTLDocumentAttributeVideo(&mtproto.DocumentAttribute{
    RoundMessage: false, SupportsStreaming: true, Nosound: true,
    W: width, H: height, Duration: 0,
}).To_DocumentAttribute()
```
