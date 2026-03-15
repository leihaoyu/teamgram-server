# Sticker Proxy Feature - 技术文档

## 概述

通过 Telegram Bot API 代理获取贴纸包数据，下载贴纸文件存储到自有 DFS（MinIO），使客户端可以在私有部署的 Teamgram 服务器上查看和使用公共贴纸包。

## 架构

```
客户端 → Gateway → Session → BFF(stickers) → Bot API(获取元数据/下载文件)
                                            → Media Service → DFS(MinIO 存储)
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
4. 并发下载所有贴纸文件（10 worker）：
   a. Bot API getFile → 获取 file_path
   b. Bot API downloadFile → 下载原始文件字节
   c. DfsWriteFilePartData → 写入 SSDB 临时存储
   d. MediaUploadedDocumentMedia → 上传到 MinIO + 注册到 documents 表
5. 存储到 MySQL: sticker_sets + sticker_set_documents
6. 返回 messages.StickerSet 响应（包含 DFS 分配的 document ID）
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

## 关键文件

| 文件 | 用途 |
|------|------|
| `app/bff/stickers/internal/core/messages.getStickerSet_handler.go` | 主处理器：获取/缓存/返回贴纸集 |
| `app/bff/stickers/internal/dao/download.go` | 下载 & 上传逻辑，序列化/反序列化 |
| `app/bff/stickers/internal/dal/dao/mysql_dao/` | MySQL DAO（sticker_sets, sticker_set_documents） |
| `app/service/dfs/internal/core/dfs.uploadDocumentFileV2_handler.go` | DFS 通用文件上传处理器 |
| `app/service/dfs/internal/core/dfs.downloadFile_handler.go` | DFS 文件下载处理器 |
| `app/service/dfs/internal/model/image_util.go` | 文件类型映射（扩展名 → storage.FileType） |
| `app/service/dfs/internal/dao/cache_file.go` | SSDB 缓存 & MinIO 读取 |
| `app/service/dfs/internal/dao/ssdb_reader.go` | SSDB 分片读取器 |
| `app/service/media/internal/core/media.uploadedDocumentMedia_handler.go` | 媒体上传：DFS + documents 表注册 |
| `app/service/media/internal/dao/document.go` | Document 表 CRUD（SaveDocumentV2, GetDocumentById） |
| `app/bff/bff/client/bff_proxy_client.go` | BFF 代理客户端（60s 超时） |

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

### DFS 后台写入 MinIO

`DfsUploadDocumentFileV2` 使用 `threading2.WrapperGoFunc` 在后台 goroutine 写入 MinIO，函数立即返回 Document。在 MinIO 写入完成前，文件通过 SSDB 缓存提供服务（2h TTL）。

**风险**:
- 服务重启时后台 goroutine 可能未完成 → MinIO 无文件 → SSDB 过期后文件丢失
- MinIO 写入失败只记日志，不重试

### 无缩略图

视频贴纸（video/webm）在 DFS 层不生成缩略图（`IsMimeAcceptedForPhotoVideoAlbum("video/webm")` 返回 false）。Document 的 `thumbs` 和 `videoThumbs` 为 nil，`flags: 0`。经 iOS 代码验证，缺少缩略图不影响渲染。

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

## 调试指南

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

## Docker 配置

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
