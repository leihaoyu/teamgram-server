# Session Routes 待实现清单

> 来源: `teamgramd/etc/session.yaml` 第 38-90 行被注释的路由
> 生成日期: 2026-03-16
> 总计: 28 个路由，230 个方法

---

## 第一批：快速见效（返回空/默认值即可，~1 周）

### 1. RPCGifs (2 methods)
- [ ] `messages.getSavedGifs` — 获取保存的 GIF 列表
- [ ] `messages.saveGif` — 保存/移除 GIF
- **备注**: 和 RecentStickers 模式几乎一样，CRUD + hash NotModified

### 2. RPCEmoji (4 methods)
- [ ] `messages.getEmojiKeywords` — 获取 emoji 关键词
- [ ] `messages.getEmojiKeywordsDifference` — 增量更新
- [ ] `messages.getEmojiKeywordsLanguages` — 支持的语言列表
- [ ] `messages.getEmojiURL` — emoji 资源 URL
- **备注**: 可先返回空列表

### 3. RPCPromoData (2 methods)
- [ ] `help.getPromoData` — 获取推广数据
- [ ] `help.hidePromoData` — 隐藏推广
- **备注**: 返回 promoDataEmpty 即可

### 4. RPCTsf (2 methods)
- [ ] `help.getUserInfo` — 获取用户帮助信息
- [ ] `help.editUserInfo` — 编辑用户帮助信息（管理员工具）
- **备注**: 返回空 userInfoEmpty

### 5. RPCCreditCards (1 method)
- [ ] `payments.getBankCardData` — 银行卡信息查询
- **备注**: 返回错误即可，私有部署不需要

### 6. RPCDeepLinks (3 methods)
- [ ] `messages.startBot` — 启动 Bot（核心）
- [ ] `help.getRecentMeUrls` — 获取最近 me 链接
- [ ] `help.getDeepLinkInfo` — 获取深度链接信息
- **备注**: startBot 是 Bot 交互入口，其余可返回空

### 7. RPCSeamless (5 methods)
- [ ] `account.getWebAuthorizations` — 获取 Web 授权列表
- [ ] `account.resetWebAuthorization` — 重置单个 Web 授权
- [ ] `account.resetWebAuthorizations` — 重置所有 Web 授权
- [ ] `messages.requestUrlAuth` — 请求 URL 授权
- [ ] `messages.acceptUrlAuth` — 接受 URL 授权
- **备注**: 返回空列表 / 默认值

### 8. RPCInternalBot (3 methods)
- [ ] `help.setBotUpdatesStatus` — 设置 Bot 更新状态
- [ ] `bots.sendCustomRequest` — 发送自定义请求
- [ ] `bots.answerWebhookJSONQuery` — 回复 Webhook 查询
- **备注**: Bot 内部接口，返回默认值

---

## 第二批：核心功能（需要数据库 + 业务逻辑，~3 周）

### 9. RPCTwoFa (7 methods)
- [ ] `account.getPassword` — 获取密码状态
- [ ] `account.getPasswordSettings` — 获取密码设置
- [ ] `account.updatePasswordSettings` — 更新密码设置
- [ ] `account.confirmPasswordEmail` — 确认密码邮箱
- [ ] `account.resendPasswordEmail` — 重发密码邮箱
- [ ] `account.cancelPasswordEmail` — 取消密码邮箱
- [ ] `account.declinePasswordReset` — 拒绝密码重置
- **备注**: 两步验证，安全相关，较重要

### 10. RPCReports (6 methods)
- [ ] `account.reportPeer` — 举报用户
- [ ] `account.reportProfilePhoto` — 举报头像
- [ ] `messages.reportSpam` — 举报垃圾信息
- [ ] `messages.report` — 举报消息
- [ ] `messages.reportEncryptedSpam` — 举报加密聊天垃圾
- [ ] `channels.reportSpam` — 举报频道垃圾
- **备注**: 可以先只记录日志不做后续处理

### 11. RPCPolls (3 methods)
- [ ] `messages.sendVote` — 发送投票
- [ ] `messages.getPollResults` — 获取投票结果
- [ ] `messages.getPollVotes` — 查看投票者
- **备注**: 需要投票数据表

### 12. RPCWebPage (3 methods)
- [ ] `messages.getWebPagePreview` — 获取链接预览
- [ ] `messages.getWebPage` (两个版本) — 获取网页信息
- **备注**: 需要 OG 标签解析服务

### 13. ~~RPCLangpack (5 methods)~~ ✅ 已完成
- [x] `langpack.getLangPack` — 获取语言包
- [x] `langpack.getStrings` — 获取翻译字符串
- [x] `langpack.getDifference` — 增量更新
- [x] `langpack.getLanguages` — 语言列表
- [x] `langpack.getLanguage` — 单个语言信息
- **实现**: `app/bff/langpack/` — 从 translations.telegram.org 抓取并缓存到本地文件，支持 33 种语言（含简体/繁体中文）
- **文档**: `docs/langpack-feature.md`

### 14. RPCScheduledMessages (4 methods)
- [ ] `messages.getScheduledHistory` — 获取定时消息列表
- [ ] `messages.getScheduledMessages` — 获取指定定时消息
- [ ] `messages.sendScheduledMessages` — 发送定时消息
- [ ] `messages.deleteScheduledMessages` — 删除定时消息
- **备注**: 需要定时任务调度机制

---

## 第三批：重要功能（多表关联 + 复杂逻辑，~2 月）

### 15. RPCFolders (17 methods)
- [ ] `messages.getDialogFilters` — 获取对话文件夹
- [ ] `messages.getSuggestedDialogFilters` — 建议的文件夹
- [ ] `messages.updateDialogFilter` — 更新文件夹
- [ ] `messages.updateDialogFiltersOrder` — 文件夹排序
- [ ] `folders.editPeerFolders` — 编辑归档
- [ ] `folders.deleteFolder` — 删除文件夹
- [ ] `chatlist` 系列 (11 个) — 共享文件夹/邀请链接
- **备注**: 对话文件夹是高频功能，但方法多、逻辑复杂

### 16. RPCReactions (12 methods)
- [ ] `messages.sendReaction` — 发送表情回应
- [ ] `messages.getMessagesReactions` — 获取消息的回应
- [ ] `messages.getMessageReactionsList` — 回应者列表
- [ ] `messages.setChatAvailableReactions` — 设置可用回应
- [ ] `messages.getAvailableReactions` — 获取可用回应列表
- [ ] `messages.setDefaultReaction` — 设置默认回应
- [ ] `messages.getUnreadReactions` — 未读回应
- [ ] `messages.readReactions` — 标记回应已读
- [ ] `messages.reportReaction` — 举报回应
- [ ] `messages.getTopReactions` — 热门回应
- [ ] `messages.getRecentReactions` — 最近用过的回应
- [ ] `messages.clearRecentReactions` — 清除最近回应
- **备注**: 消息回应是现代 IM 核心功能

### 17. RPCBots (6 methods) + RPCInlineBot (7 methods)
- [ ] `bots.setBotCommands` / `resetBotCommands` / `getBotCommands` — Bot 命令管理
- [ ] `bots.setBotInfo` / `getBotInfo` — Bot 信息
- [ ] `messages.getInlineBotResults` — inline 查询
- [ ] `messages.setInlineBotResults` — 设置 inline 结果
- [ ] `messages.sendInlineBotResult` — 发送 inline 结果
- [ ] `messages.editInlineBotMessage` — 编辑 inline 消息
- [ ] `messages.getBotCallbackAnswer` / `setBotCallbackAnswer` — 回调按钮
- [ ] `messages.sendBotRequestedPeer` — Bot 请求选择联系人
- **备注**: Bot 全套功能，13 个方法

### 18. RPCThemes (9 methods)
- [ ] `account.uploadTheme` — 上传主题
- [ ] `account.createTheme` — 创建主题
- [ ] `account.updateTheme` — 更新主题
- [ ] `account.saveTheme` — 保存主题
- [ ] `account.installTheme` — 安装主题
- [ ] `account.getTheme` — 获取主题
- [ ] `account.getThemes` — 主题列表
- [ ] `account.getChatThemes` — 聊天主题
- [ ] `messages.setChatTheme` — 设置聊天主题
- **备注**: 个性化功能

### 19. RPCWallpapers (8 methods)
- [ ] `account.getWallPapers` — 壁纸列表
- [ ] `account.getWallPaper` — 单个壁纸
- [ ] `account.uploadWallPaper` — 上传壁纸
- [ ] `account.saveWallPaper` — 保存壁纸
- [ ] `account.installWallPaper` — 安装壁纸
- [ ] `account.resetWallPapers` — 重置壁纸
- [ ] `account.getMultiWallPapers` — 批量获取壁纸
- [ ] `messages.setChatWallPaper` — 设置聊天壁纸
- **备注**: 个性化功能

---

## 第四批：高级功能（~2-3 月）

### 20. RPCChannels (34 methods)
- [ ] 频道 CRUD（create/delete/get/join/leave/invite）
- [ ] 权限管理（editAdmin/editBanned/editCreator）
- [ ] 消息操作（readHistory/deleteMessages/getMessages）
- [ ] 频道设置（editTitle/editPhoto/toggleSignatures/toggleSlowMode 等）
- [ ] 管理日志（getAdminLog）
- [ ] 讨论组（getGroupsForDiscussion/setDiscussionGroup）
- **备注**: 最大的模块，34 个方法，需要频道基础数据模型

### 21. RPCMessageThreads (4 methods)
- [ ] `contacts.blockFromReplies` — 屏蔽回复者
- [ ] `messages.getReplies` — 获取回复列表
- [ ] `messages.getDiscussionMessage` — 获取讨论消息
- [ ] `messages.readDiscussion` — 标记讨论已读
- **备注**: 依赖频道支持

### 22. RPCStatistics (8 methods)
- [ ] `stats.getBroadcastStats` — 频道统计
- [ ] `stats.getMegagroupStats` — 超级群统计
- [ ] `stats.getMessageStats` — 消息统计
- [ ] `stats.getStoryStats` — 故事统计
- [ ] `stats.loadAsyncGraph` — 加载统计图表
- [ ] 其他 3 个方法
- **备注**: 依赖频道支持 + 数据统计系统

---

## 第五批：按需实现

### 23. RPCSecretChats (10 methods)
- [ ] DH 密钥交换 + 加密消息收发
- **备注**: 端到端加密，复杂度高，需独立方案

### 24. RPCPassport (11 methods)
- [ ] Telegram Passport 身份验证
- **备注**: 私有部署通常不需要

### 25. RPCPayments (12 methods)
- [ ] 支付功能全套
- **备注**: 需要支付网关集成

### 26. RPCGames (4 methods)
- [ ] 游戏分数/排行榜
- **备注**: 依赖 Bot 平台

### 27. RPCImportedChats (5 methods)
- [ ] 聊天记录导入（WhatsApp 等迁移）
- **备注**: 特殊场景

### 28. RPCVoipCalls (11 methods) + RPCGroupCalls (21 methods)
- [ ] 1v1 语音/视频通话 + 群组通话
- **备注**: 需要独立 VoIP 服务端（WebRTC/SRTP），工作量最大

---

## 统计

| 批次 | 路由数 | 方法数 | 预估工作量 |
|------|--------|--------|-----------|
| 第一批 | 8 | 22 | ~1 周 |
| 第二批 | 6 | 28 | ~3 周 | RPCLangpack ✅ |
| 第三批 | 5 | 55 | ~2 月 |
| 第四批 | 3 | 46 | ~2-3 月 |
| 第五批 | 5 | 79 | 按需 |
| **总计** | **28** | **230** | **~5-7 月** |
