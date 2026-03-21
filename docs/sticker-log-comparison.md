# Sticker 模块日志对比分析

> 对比 Telegram iOS 客户端 vs OwnPod iOS 客户端启动后点击 Sticker 菜单的完整流程

## 1. OwnPod 错误汇总

### 1.1 STICKER_ID_INVALID 错误 (7 个) — ✅ 已修复

客户端启动时请求 5 种系统内置 sticker set，服务端不认识这些 predicate 类型：

| 请求 | shortName | 状态 |
|------|-----------|------|
| `getStickerSet(inputStickerSetAnimatedEmoji)` | `AnimatedEmojies` | ✅ 已支持 |
| `getStickerSet(inputStickerSetAnimatedEmojiAnimations)` | `EmojiAnimations` | ✅ 已支持 |
| `getStickerSet(inputStickerSetEmojiGenericAnimations)` | `EmojiGenericAnimations` | ✅ 已支持 |
| `getStickerSet(inputStickerSetEmojiDefaultStatuses)` | `StatusPack` | ✅ 已支持 |
| `getStickerSet(inputStickerSetEmojiDefaultTopicIcons)` | `Topics` | ✅ 已支持 |

**实现方式**: 在 `getStickerSet` handler 的 switch 中新增 5 个 case，将每个 predicate 映射到对应的 shortName，复用已有的 `fetchAndCacheStickerSet` 流程（Bot API 获取 → DFS 下载 → MySQL 缓存）。如果 Bot API 获取失败（某些 custom_emoji 类型），返回空但合法的 `Messages_StickerSet` 而非报错。

**文件**: `app/bff/stickers/internal/core/messages.getStickerSet_handler.go`

### 1.2 METHOD_NOT_IMPL 错误 (20+ 个) — ✅ 已修复

客户端调用但服务端之前完全未实现的 API，现已全部在 `fake_rpc_result.go` 中返回空但合法的响应：

| 方法 | 说明 | 返回类型 |
|------|------|----------|
| `messages.getAvailableReactions` | 可用的表情回复列表 | `messagesAvailableReactions(hash:0, reactions:[])` |
| `messages.getRecentReactions` | 最近使用的 reaction | `messagesReactions(hash:0, reactions:[])` |
| `messages.getTopReactions` | 热门 reaction | `messagesReactions(hash:0, reactions:[])` |
| `messages.getEmojiGroups` | emoji 分组 | `messagesEmojiGroups(hash:0, groups:[])` |
| `messages.getEmojiStickers` | emoji sticker 集合 | `messagesAllStickers(hash:0, sets:[])` |
| `messages.getFeaturedEmojiStickers` | 热门 emoji sticker | `messagesFeaturedStickers(count:0, sets:[])` |
| `messages.getEmojiStatusGroups` | emoji 状态分组 | `messagesEmojiGroups(hash:0, groups:[])` |
| `messages.getEmojiProfilePhotoGroups` | 头像 emoji 分组 | `messagesEmojiGroups(hash:0, groups:[])` |
| `messages.getAttachMenuBots` | 附件菜单 bot | `attachMenuBots(hash:0, bots:[], users:[])` |
| `messages.getSuggestedDialogFilters` | 推荐对话文件夹 | `Vector_DialogFilterSuggested([])` |
| `help.getPremiumPromo` | Premium 推广 | `helpPremiumPromo(empty)` |
| `messages.getDefaultHistoryTTL` | 默认消息自毁时间 | `defaultHistoryTTL(period:0)` |
| `account.getSavedRingtones` | 保存的铃声 | `accountSavedRingtones(hash:0, ringtones:[])` |
| `account.getRecentEmojiStatuses` | 最近使用的 emoji 状态 | `accountEmojiStatuses(hash:0, statuses:[])` |
| `account.getDefaultEmojiStatuses` | 默认 emoji 状态 | `accountEmojiStatuses(hash:0, statuses:[])` |
| `account.getDefaultProfilePhotoEmojis` | 默认头像 emoji | `emojiList(hash:0, documentId:[])` |
| `account.getDefaultGroupPhotoEmojis` | 默认群头像 emoji | `emojiList(hash:0, documentId:[])` |
| `messages.getRecentLocations` | 最近位置 | `messagesMessages(empty)` |
| `messages.getScheduledHistory` | 定时消息 | `messagesMessages(empty)` |
| `help.test` | 测试端点 | `boolTrue` |

**实现方式**: 在 `fake_rpc_result.go` 的 switch 中新增 case，返回空但类型正确的响应。无需注册新 RPC 服务或修改 session.yaml 路由。

**文件**: `app/bff/bff/client/fake_rpc_result.go`

### 1.3 正常返回 (空数据)

这些 API 已实现，返回了空结果（符合预期 — 新用户没有数据）：

| 方法 | 返回 |
|------|------|
| `messages.getStickers(emoticon: 👋⭐️)` | ✅ 已修复 — 返回 DB 中所有带 👋 emoji 的 sticker |
| `messages.getRecentStickers(hash: 0)` | `recentStickers(hash: 0, packs: [], stickers: [], dates: [])` ✓ |
| `messages.getSavedGifs(hash: 0)` | `savedGifs(hash: 0, gifs: [])` ✓ |
| `messages.getFavedStickers(hash: 0)` | `favedStickers(hash: 0, packs: [], stickers: [])` ✓ |
| `messages.getAllStickers(hash: 0)` | `allStickers(hash: 0, sets: [])` ✓ |
| `messages.getMaskStickers(hash: 0)` | `allStickers(hash: 0, sets: [])` ✓ |
| `messages.getArchivedStickers(...)` | `archivedStickers(count: 0, sets: [])` ✓ |
| `messages.getFeaturedStickers(hash: 0)` | `featuredStickers(...)` ✓ (有数据) |
| `messages.getDialogFilters` | 正常返回 ✓ |

---

## 2. 数据格式差异对比

### 2.1 StickerSet 字段差异

| 字段 | Telegram | OwnPod (修复前) | OwnPod (修复后) | 说明 |
|------|----------|-----------------|-----------------|------|
| `flags` | 正确位图 (如 49) | 0 或仅 animated | ✅ 自动正确 | flags 由 proto 编码器根据字段自动计算 |
| `animated` | ✓ | ✓ 已正确 | ✓ | bit 5 |
| `videos` | ✓ | ✓ 已正确 | ✓ | bit 6 |
| `masks` | ✓ | ✓ 已正确 | ✓ | bit 3 → `isMasks` |
| `emojis` | ✓ | ✓ 已正确 | ✓ | bit 7 → `isEmojis` |
| `installedDate` | 真实时间戳 | ✅ 已修复 (之前 nil) | ✓ | 安装后 `getAllStickers`/`getStickerSet` 返回 |
| `thumbDocumentId` | 部分有值 | nil | ✅ 已修复 | 设为第一个 document 的 ID (bit 8) |
| `thumbs` + `thumbDcId` + `thumbVersion` | 有 (photoPathSize + photoSize, 100x100) | nil | ❌ 未修复 | 需要 set-level 缩略图上传到 MinIO `photos` bucket |
| `hash` | 非零 (如 887037557) | 0 | ✅ 已修复 | 基于 document IDs 的 `combineInt64Hash`，支持 NotModified |

**flags 分析** (proto 编码器自动计算):
- bit 0 (1): `InstalledDate != nil`
- bit 1 (2): `Archived`
- bit 2 (4): `Official`
- bit 3 (8): `Masks`
- bit 4 (16): `Thumbs != nil || ThumbDcId != nil || ThumbVersion != nil`
- bit 5 (32): `Animated`
- bit 6 (64): `Videos`
- bit 7 (128): `Emojis`
- bit 8 (256): `ThumbDocumentId != nil`

> **结论**: flags 不需要手动计算，proto 编码器自动做。OwnPod 之前 flags=0 的 webp set 实际就该是 0（非 animated, 非 video），animated set 的 flags=32 也是对的。主要缺失是 bit 4 (set thumbs) 和 bit 8 (thumbDocumentId)。bit 8 现已通过设置 ThumbDocId 修复。

### 2.2 Document 格式差异

| 字段 | Telegram | OwnPod (修复前) | OwnPod (修复后) | 说明 |
|------|----------|-----------------|-----------------|------|
| `fileReference` | 非空 bytes | `<null>` | ⚠️ 不改 | proto3 序列化/反序列化 `[]byte{}` → `nil`，服务端不校验，无实际影响 |
| `thumbs` | `[photoPathSize, photoSize(128x128)]` | nil | ✅ 已修复 | Bot API thumbnail → SSDB → DFS 解码 webp → `photoStrippedSize` + `photoSize(128x128)` |
| `flags` | 1 (有 thumbs) | 0 | ✅ 自动修复 | 有 thumbs 后 flags 自动包含 bit 0 |

**Telegram sticker Document thumbs 格式**:
```
thumbs: [
  photoPathSize(type: "j", bytes: ...),      ← SVG 路径压缩 (~100-270 bytes, 内联)
  photoSize(type: "m", w: 128, h: 128, size: ~3-5KB)  ← 需要从 MinIO photos/m/{docId}.dat 下载
]
```

**OwnPod 修复后 Document thumbs 格式**:
```
thumbs: [
  photoStrippedSize(type: "i", bytes: ...),  ← JPEG 缩略预览 (~100-200 bytes, 内联)
  photoSize(type: "m", w: 128, h: 128, size: ~3-5KB)  ← MinIO photos/m/{docId}.dat
]
```

> 差异：Telegram 用 `photoPathSize` (SVG path)，我们用 `photoStrippedSize` (stripped JPEG)。两者作用相同 — 提供内联的极小预览。`photoPathSize` 生成需要 SVG 路径提取算法，`photoStrippedSize` 用标准 JPEG 编码更易实现。客户端两者都支持。

---

## 3. 修复状态总结

### ✅ 已完成

| # | 工作项 | 文件 |
|---|--------|------|
| 1 | `getStickerSet` 支持 5 种系统内置 predicate | `messages.getStickerSet_handler.go` |
| 2 | `getStickerSet` 安装后返回 `InstalledDate` | `messages.getStickerSet_handler.go`, `user_installed_sticker_sets_dao.go` |
| 3 | Document 缩略图 (thumbs) — 通过 Bot API thumbnail 下载 | `download.go` (BFF 流式直传 MinIO) |
| 4 | StickerSet.ThumbDocumentId 设为第一个 document ID | `messages.getStickerSet_handler.go` |
| 5 | StickerSet.hash 非零 + getStickerSet NotModified 支持 | `messages.getStickerSet_handler.go` |
| 6 | METHOD_NOT_IMPL 20+ 个 API 返回空列表替代报错 | `fake_rpc_result.go` |
| 7 | `help.saveAppLog` 返回 boolTrue 替代 METHOD_NOT_IMPL | `fake_rpc_result.go` |
| 8 | Greeting sticker: `messages.getStickers(👋⭐️)` 返回真实 sticker | `messages.getStickers_handler.go`, `sticker_set_documents_dao.go` |

**缩略图实现流程（当前 — 流式直传）**:
```
Bot API sticker.thumbnail.file_id
  → BFF: downloadThumbBytes → []byte (小文件，内存缓冲)
  → BFF: MinIO PutObject(photos/m/{docId}.dat, thumbData)
  → BFF: imaging.DecodeWebp(thumbData) → Resize(40px) → EncodeStripped
  → Document.Thumbs = [photoStrippedSize("i"), photoSize("m")]
  → serialize to DB (document_data)
```

### ⚠️ 暂不修复

| # | 工作项 | 原因 |
|---|--------|------|
| 1 | `Document.fileReference` null → `[]byte{}` | 服务端不校验 fileReference，`nil` vs `[]byte{}` 无实际影响 |

### ❌ 待修复

#### P1 — 改进体验

| # | 工作项 | 说明 | 复杂度 |
|---|--------|------|--------|
| 1 | `StickerSet.thumbs/thumbDcId/thumbVersion` (bit 4) | Telegram 的 set-level 100x100 缩略图，需新增 DB 列或从 Bot API set.thumbnail 上传 | 中 |

---

## 4. 注意事项

- 已缓存的 sticker set 需要**清除 DB 数据后重新抓取**才能获得新的缩略图和 ThumbDocumentId
- 清除方式：`DELETE FROM sticker_sets; DELETE FROM sticker_set_documents;`
- 系统内置 set 中部分是 `custom_emoji` 类型（如 StatusPack, Topics），Bot API 可能无法获取，此时返回空 set
