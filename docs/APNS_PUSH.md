# APNs 推送系统文档

## 概述

TeamGram 的 Apple Push Notification (APNs) 推送系统，用于向 iOS 设备发送实时推送通知。

## 配置参数

| 参数 | 值 | 说明 |
|-----|----------|------|
| KeyFile | `../etc/AuthKey_JH5C27A29G.p8` | APNs 认证密钥文件路径 |
| KeyID | `JH5C27A29G` | Apple Key ID |
| TeamID | `3WA4Q9D2GD` | Apple Team ID |
| BundleID | `org.delta.pchat` | iOS 应用 Bundle ID |
| Production | dev: `false` / prod: `true` | 由 docker-compose 自动控制 |

## 自动化配置

### Docker 自动注入

APNs 配置通过 Docker 环境变量自动注入，无需手动修改 YAML 文件。

**工作流:**
1. `docker-compose.yaml` (dev) 或 `docker-compose.prod.yaml` (prod) 设定 `APNS_*` 环境变量
2. `entrypoint.sh` 检测到 `APNS_KEY_FILE` 非空，自动追加 APNs 配置到 `sync.yaml`
3. Sync 服务启动时加载配置并初始化 APNs 客户端

**开发环境:**
```bash
docker-compose build && docker-compose up -d
# APNS_PRODUCTION=false → 使用 APNs Development 环境
```

**生产环境:**
```bash
docker-compose -f docker-compose.prod.yaml build && docker-compose -f docker-compose.prod.yaml up -d
# APNS_PRODUCTION=true → 使用 APNs Production 环境
```

### 手动配置（仅非 Docker 裸机部署）

如需手动配置（不使用 Docker 部署时），运行：
```bash
./teamgramd/scripts/configure-apns.sh [--production]
```

### 验证

```bash
./teamgramd/scripts/test-apns-push.sh
```

## 推送流程架构

```
iOS 客户端
  │ account.registerDevice (token_type=1)
  ▼
BFF Notification Service → INSERT devices 表
  │
  │ 消息到达
  ▼
Sync Service: SyncPushUpdatesIfNot
  │
  ├─ GetUserAPNsDevices() → 查询 devices 表
  ├─ extractPushPayload() → 从 MTProto Updates 提取消息
  ├─ excludeMap → 排除在线 session
  └─ SendAPNsPush() → apns2.Client.PushWithContext()
       │
       ▼
  Apple APNs 服务器 → iOS 设备
```

### 代码文件

| 层级 | 文件 | 功能 |
|------|------|------|
| Config | `app/messenger/sync/internal/config/config.go` | APNsConfig 结构定义 |
| DAO | `app/messenger/sync/internal/dao/dao.go` | APNs 客户端初始化 |
| Push | `app/messenger/sync/internal/dao/push.go` | GetUserAPNsDevices / SendAPNsPush |
| Handler | `app/messenger/sync/internal/core/sync.pushUpdatesIfNot_handler.go` | 推送触发 & extractPushPayload |
| Register | `app/bff/notification/internal/core/account.registerDevice_handler.go` | 设备注册 |
| Unregister | `app/bff/notification/internal/core/account.unregisterDevice_handler.go` | 设备注销 |
| Devices DB | `app/bff/notification/internal/dao/devices.go` | RegisterDevice / UnregisterDevice |

### extractPushPayload 支持的 Update 类型

| PredicateName | 含义 | 提取内容 |
|---|---|---|
| `updateShortMessage` | 用户私聊 | UserId, Message, Id |
| `updateShortChatMessage` | 群组消息 | FromId, Message, ChatId |
| `updates` (container) | 含 `updateNewMessage` | Message, FromId, PeerId, Silent |
| `updateShort` | 单条 update | 同 updateNewMessage |

## 数据库

### devices 表

```sql
CREATE TABLE `devices` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `auth_key_id` bigint(20) NOT NULL,
  `user_id` bigint(20) NOT NULL,
  `token_type` int(11) NOT NULL,        -- 1: iOS APNs
  `token` varchar(512) NOT NULL,
  `no_muted` tinyint(1) NOT NULL DEFAULT '0',
  `app_sandbox` tinyint(1) NOT NULL DEFAULT '0',
  `secret` varchar(1024) NOT NULL DEFAULT '',
  `other_uids` varchar(1024) NOT NULL DEFAULT '',
  `state` tinyint(1) NOT NULL DEFAULT '0',  -- 0: 有效, 1: 无效
  PRIMARY KEY (`id`),
  UNIQUE KEY (`auth_key_id`, `user_id`, `token_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

注意: 该表已包含在 `teamgramd/sql/1_teamgram.sql` 中，随数据库初始化自动创建。

## 推送负载格式

### 标准推送

```json
{
  "aps": {
    "alert": { "title": "Alice", "body": "Hello!" },
    "sound": "default",
    "badge": 1,
    "mutable-content": 1
  },
  "custom": {
    "from_id": 123456,
    "msg_id": 1,
    "peer_type": "user",
    "peer_id": 789012
  }
}
```

### 群组消息

```json
{
  "aps": {
    "alert": { "title": "New Message", "body": "Let's meet" },
    "sound": "default", "badge": 1, "mutable-content": 1
  },
  "custom": {
    "from_id": 123, "msg_id": 2,
    "peer_type": "chat", "peer_id": 999, "chat_id": 999
  }
}
```

### 静音推送

```json
{
  "aps": { "content-available": 1, "mutable-content": 1 },
  "custom": { "from_id": 123, "msg_id": 1, "peer_type": "user", "peer_id": 456 }
}
```

## 错误处理

### 无效令牌自动清理

当 APNs 返回以下错误时，系统自动将设备标记为无效 (`state=1`)：

| Reason | 说明 |
|--------|------|
| `BadDeviceToken` | token 格式无效 |
| `Unregistered` | 设备已注销 |
| `ExpiredToken` | token 已过期 |

### 故障排查

```bash
# 检查 APNs 初始化
docker-compose logs teamgram | grep -i "apns"

# 检查推送日志
docker-compose logs teamgram | grep "SendAPNsPush"

# 检查设备注册
mysql -u root teamgram -e "SELECT user_id, token, state FROM devices;"
```

## 测试

### Go 单元测试

```bash
go test -vet=off ./app/messenger/sync/internal/dao/ -run "TestP8|TestAPNs|TestPushPayload|TestDeviceInfo" -v
```

测试覆盖：
- p8 文件存在性和格式验证
- AuthKey 加载
- APNs 客户端创建（dev/prod）
- APNs 真实连接测试（向无效 token 发送，验证收到 BadDeviceToken）
- PushPayload 和 DeviceInfo 结构体

### 配置测试脚本

```bash
./teamgramd/scripts/test-apns-push.sh
```

验证 15 项配置：文件、参数、p8、数据库、Docker 配置、Go 测试。

## 文件清单

```
teamgram-server/
├── docker-compose.yaml          # 开发环境 APNs 环境变量
├── docker-compose.prod.yaml     # 生产环境 APNs 环境变量
├── teamgramd/
│   ├── etc/
│   │   ├── sync.yaml            # APNs 配置模板
│   │   └── AuthKey_JH5C27A29G.p8  # 认证密钥（.gitignore）
│   ├── docker/
│   │   └── entrypoint.sh        # 自动注入 APNs 配置
│   └── scripts/
│       ├── configure-apns.sh    # 仅非 Docker 裸机部署时使用
│       └── test-apns-push.sh    # 配置测试脚本
├── app/messenger/sync/internal/
│   ├── config/config.go         # APNsConfig 定义
│   ├── dao/
│   │   ├── dao.go               # APNs 客户端初始化
│   │   ├── push.go              # 推送实现
│   │   └── push_test.go         # 推送测试
│   └── core/
│       └── sync.pushUpdatesIfNot_handler.go  # 推送触发
├── app/bff/notification/internal/
│   ├── core/
│   │   ├── account.registerDevice_handler.go
│   │   └── account.unregisterDevice_handler.go
│   └── dao/devices.go           # 设备 DB 操作
└── docs/
    ├── APNS_PUSH.md             # 本文档
    └── APNS_PUSH_QUICKSTART.md  # 快速指南
```

---

**更新日志:**

| 日期 | 说明 |
|------|------|
| 2026-03-23 | 初始版本: 推送系统实现 |
| 2026-03-23 | Docker 自动化配置: dev/prod 环境变量注入, p8 真实连接测试 |
