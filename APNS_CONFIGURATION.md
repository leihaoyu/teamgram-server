# APNs 推送系统配置清单

**项目:** TeamGram Server
**最后更新:** 2026-03-23
**状态:** ✅ 已配置

---

## 配置概览

| 项目 | 值 |
|-----|-----|
| **App Bundle ID** | `org.delta.pchat` |
| **Apple Team ID** | `3WA4Q9D2GD` |
| **Apple Key ID** | `JH5C27A29G` |
| **认证方式** | Token-based (p8 文件) |
| **开发环境** | `docker-compose.yaml` → `Production: false` |
| **生产环境** | `docker-compose.prod.yaml` → `Production: true` |

---

## 自动化配置机制

APNs 配置通过 Docker 环境变量自动注入，**新机器部署无需手动配置**。

### 工作原理

1. `docker-compose.yaml` / `docker-compose.prod.yaml` 定义 APNs 环境变量
2. `entrypoint.sh` 读取环境变量，自动注入到 `sync.yaml`
3. Sync 服务启动时加载配置，初始化 APNs 客户端

### 开发环境 (`docker-compose.yaml`)

```yaml
environment:
  APNS_KEY_FILE: "../etc/AuthKey_JH5C27A29G.p8"
  APNS_KEY_ID: "JH5C27A29G"
  APNS_TEAM_ID: "3WA4Q9D2GD"
  APNS_BUNDLE_ID: "org.delta.pchat"
  APNS_PRODUCTION: "false"
```

### 生产环境 (`docker-compose.prod.yaml`)

```yaml
environment:
  APNS_KEY_FILE: "../etc/AuthKey_JH5C27A29G.p8"
  APNS_KEY_ID: "JH5C27A29G"
  APNS_TEAM_ID: "3WA4Q9D2GD"
  APNS_BUNDLE_ID: "org.delta.pchat"
  APNS_PRODUCTION: "true"
```

---

## 部署（新机器，零配置）

p8 文件已随代码打包进 Docker 镜像（Dockerfile COPY），直接启动即可：

```bash
# 开发环境
docker-compose build && docker-compose up -d

# 生产环境
docker-compose -f docker-compose.prod.yaml build && docker-compose -f docker-compose.prod.yaml up -d
```

**不需要**：
- 不需要运行 `configure-apns.sh`（那是给非 Docker 裸机部署用的）
- 不需要手动编辑 `sync.yaml`
- 不需要手动启动 sync 服务（`runall-docker.sh` 自动启动所有服务）

APNs 配置由 `entrypoint.sh` 从 docker-compose 环境变量自动注入。

### 可选验证

```bash
docker-compose logs teamgram | grep -i "apns"
# 预期: APNs: client initialized, bundleID=org.delta.pchat
```

---

## 文件清单

| 文件 | 状态 | 说明 |
|-----|------|------|
| `teamgramd/etc/AuthKey_JH5C27A29G.p8` | ✅ | APNs 认证密钥 |
| `teamgramd/etc/sync.yaml` | ✅ | APNs 配置模板（Docker 自动注入） |
| `teamgramd/docker/entrypoint.sh` | ✅ | 自动注入 APNs 配置 |
| `docker-compose.yaml` | ✅ | 开发环境 APNs 环境变量 |
| `docker-compose.prod.yaml` | ✅ | 生产环境 APNs 环境变量 |
| `teamgramd/scripts/configure-apns.sh` | ✅ | 手动配置脚本（仅非 Docker 裸机部署时使用） |
| `teamgramd/scripts/test-apns-push.sh` | ✅ | 配置测试脚本 |
| `app/messenger/sync/internal/dao/push_test.go` | ✅ | Go 单元测试（含 p8 真实连接测试） |
| `docs/APNS_PUSH.md` | ✅ | 完整文档 |
| `docs/APNS_PUSH_QUICKSTART.md` | ✅ | 快速启动指南 |

---

## 安全

- p8 文件已加入 `.gitignore` (`teamgramd/etc/*.p8`)
- p8 文件权限 644
- Docker 镜像构建时自动包含 p8 文件（通过 Dockerfile COPY）

---

## 测试结果

### Go 单元测试

```
✓ TestP8FileExists         - p8 文件存在
✓ TestP8FileFormat          - PEM 格式有效
✓ TestAuthKeyFromP8File     - 密钥加载成功
✓ TestAPNsTokenCreation     - 客户端创建成功
✓ TestAPNsSendToInvalidToken - APNs 连接验证（收到 BadDeviceToken 400）
✓ TestPushPayloadStructure  - 负载结构正确
✓ TestDeviceInfoStructure   - 设备结构正确
```

### 配置测试脚本 (15/15 通过)

```
✓ 配置文件存在
✓ APNs 配置模板存在
✓ KeyFile/KeyID/TeamID/BundleID 已配置
✓ p8 文件存在、格式有效、大小正常
✓ DevicesMySQL 配置存在
✓ Go 单元测试通过
✓ docker-compose.yaml/prod.yaml APNs 配置正确
✓ 开发 Production=false / 生产 Production=true
```
