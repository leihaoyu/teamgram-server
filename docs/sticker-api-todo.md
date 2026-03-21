# Sticker 模块 API 实现状态

> 文件：`app/bff/stickers/internal/server/grpc/service/stickers_service_impl.go`
> 共 30 个 RPC 方法

---

## 已实现（10 个）

核心的贴纸消费流程已完成。

| # | 方法 | 功能 | 备注 |
|---|------|------|------|
| 1 | `messages.getStickerSet` | 获取贴纸集详情 | Bot API 代理 + 流式直传 MinIO + MySQL 缓存 |
| 2 | `messages.getAllStickers` | 获取用户已安装的所有贴纸集 | Telegram hash 支持 NotModified |
| 3 | `messages.installStickerSet` | 安装贴纸集 | IncrementOrderNum + InsertOrUpdate |
| 4 | `messages.uninstallStickerSet` | 卸载贴纸集 | 软删除 |
| 5 | `messages.reorderStickerSets` | 重新排序贴纸集 | 按客户端 Order 数组更新 |
| 6 | `messages.getRecentStickers` | 获取最近使用的贴纸 | hash 支持 NotModified |
| 7 | `messages.saveRecentSticker` | 保存最近贴纸（客户端调用） | 加密聊天用；普通聊天由服务端自动记录 |
| 8 | `messages.clearRecentStickers` | 清空最近贴纸 | 软删除所有 |
| 9 | `messages.getFavedStickers` | 获取收藏的贴纸 | reverseHashOrder |
| 10 | `messages.faveSticker` | 收藏/取消收藏贴纸 | toggle 模式 |

---

## 未实现 — 返回空数据（12 个）

这些方法不会报错，但返回空列表。客户端能正常运行，只是功能缺失。

### 优先级 P1 — 影响用户体验（建议优先做）

| # | 方法 | 功能 | 难度 | 说明 |
|---|------|------|------|------|
| 11 | `messages.getStickers` | 按 emoji 搜索贴纸 | ⭐⭐ | 用户在输入框输入 emoji 时，弹出该 emoji 对应的贴纸建议。需查询已安装贴纸集中匹配 emoji 的 Document |
| 12 | `messages.searchStickerSets` | 搜索贴纸集 | ⭐⭐⭐ | 需要对接 Bot API 搜索或本地全文索引。客户端"搜索贴纸"功能依赖此接口 |
| 13 | `messages.getFeaturedStickers` | 获取推荐/热门贴纸集 | ⭐⭐ | 客户端"热门贴纸"tab 依赖此接口。可从 Bot API 获取热门集或后台管理配置 |

### 优先级 P2 — 锦上添花

| # | 方法 | 功能 | 难度 | 说明 |
|---|------|------|------|------|
| 14 | `messages.readFeaturedStickers` | 标记推荐贴纸已读 | ⭐ | 依赖 getFeaturedStickers 先实现。记录用户已读的推荐集 ID |
| 15 | `messages.getOldFeaturedStickers` | 获取历史推荐贴纸 | ⭐ | 分页获取旧的推荐集，依赖 getFeaturedStickers |
| 16 | `messages.getArchivedStickers` | 获取已归档的贴纸集 | ⭐⭐ | 用户归档（隐藏）的贴纸集列表。需要数据库支持 archived 状态 |
| 17 | `messages.getMaskStickers` | 获取面具贴纸 | ⭐ | 同 getAllStickers 但 set_type=1。照片编辑器面具功能 |
| 18 | `messages.getAttachedStickers` | 获取照片/视频上附着的贴纸 | ⭐⭐ | 照片编辑器贴纸覆盖层，使用场景少 |
| 19 | `messages.toggleStickerSets` | 批量安装/卸载/归档贴纸集 | ⭐⭐ | 批量操作，需要遍历 set_id 列表执行 install/uninstall/archive |
| 20 | `messages.searchEmojiStickerSets` | 搜索 emoji 贴纸集 | ⭐⭐ | 自定义 emoji 功能的搜索接口 |

### 优先级 P3 — 暂不需要

| # | 方法 | 功能 | 难度 | 说明 |
|---|------|------|------|------|
| 21 | `stickers.checkShortName` | 检查贴纸集短名是否可用 | ⭐ | 创建贴纸集前检查。目前返回 BoolFalse |
| 22 | `stickers.deleteStickerSet` | 删除贴纸集 | ⭐ | 目前返回 BoolTrue（假装成功）|

---

## 未实现 — 返回 ErrMethodNotImpl（8 个）

这些方法调用时会报错。都是贴纸集**创建和管理**功能（`stickers.*` 命名空间），面向贴纸集作者/管理员。

### 优先级 P3 — 贴纸创作者功能

| # | 方法 | 功能 | 难度 | 说明 |
|---|------|------|------|------|
| 23 | `stickers.createStickerSet` | 创建新贴纸集 | ⭐⭐⭐⭐ | 完整的贴纸集创建流程：上传图片、设定 emoji、设置元数据。最复杂的接口 |
| 24 | `stickers.addStickerToSet` | 向贴纸集添加贴纸 | ⭐⭐⭐ | 上传新贴纸到已有集。需处理 InputStickerSetItem |
| 25 | `stickers.removeStickerFromSet` | 从贴纸集移除贴纸 | ⭐⭐ | 通过 InputDocument 定位并移除 |
| 26 | `stickers.changeStickerPosition` | 调整贴纸在集中的位置 | ⭐ | 更新 position 字段 |
| 27 | `stickers.changeSticker` | 修改贴纸属性 | ⭐⭐ | 修改贴纸的 emoji、mask 坐标等 |
| 28 | `stickers.setStickerSetThumb` | 设置贴纸集缩略图 | ⭐⭐ | 上传并设置贴纸集封面 |
| 29 | `stickers.renameStickerSet` | 重命名贴纸集 | ⭐ | 修改贴纸集标题 |
| 30 | `stickers.suggestShortName` | 推荐贴纸集短名 | ⭐ | 根据标题自动生成短名建议 |

---

## 建议实现顺序

### 第一批：提升用户体验（3 个方法）

```
1. messages.getStickers        — emoji 输入时的贴纸建议
2. messages.getFeaturedStickers — 热门/推荐贴纸 tab
3. messages.searchStickerSets  — 贴纸搜索功能
```

**预估工作量**：2-3 天

### 第二批：完善安装管理（4 个方法）

```
4. messages.getMaskStickers      — 面具贴纸（复用 getAllStickers 逻辑）
5. messages.getArchivedStickers  — 归档列表
6. messages.toggleStickerSets    — 批量操作
7. messages.readFeaturedStickers — 已读标记
```

**预估工作量**：1-2 天

### 第三批：贴纸创作者功能（8 个方法）

```
8.  stickers.createStickerSet
9.  stickers.addStickerToSet
10. stickers.removeStickerFromSet
11. stickers.changeStickerPosition
12. stickers.changeSticker
13. stickers.setStickerSetThumb
14. stickers.renameStickerSet
15. stickers.suggestShortName
```

**预估工作量**：5-7 天（createStickerSet 最复杂）

### 第四批：其他（3 个方法）

```
16. messages.getOldFeaturedStickers  — 历史推荐
17. messages.getAttachedStickers     — 照片附着贴纸
18. messages.searchEmojiStickerSets  — emoji 贴纸搜索
```

**预估工作量**：1-2 天

---

## 总计

| 状态 | 数量 |
|------|------|
| 已实现 | 10 |
| 空数据桩 | 12 |
| 返回错误 | 8 |
| **合计** | **30** |

> 完全实现所有 30 个接口预估需要 **10-15 天**。
> 优先做第一批（P1）即可显著改善客户端贴纸使用体验。
