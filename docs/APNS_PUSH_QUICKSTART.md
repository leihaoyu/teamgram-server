# APNs 推送系统 - 快速启动

## 部署（新机器，零配置）

p8 文件已打包进 Docker 镜像，直接启动即可，APNs 全自动配置。

### 开发环境

```bash
docker-compose -f docker-compose-env.yaml up -d
docker-compose build && docker-compose up -d
```

### 生产环境

```bash
docker-compose -f docker-compose-env.prod.yaml up -d
docker-compose -f docker-compose.prod.yaml build && docker-compose -f docker-compose.prod.yaml up -d
```

启动后 APNs 推送自动工作，不需要运行任何配置脚本，不需要手动编辑任何文件。

### 可选验证

```bash
docker-compose logs teamgram | grep -i "apns"
# 预期: APNs: client initialized, bundleID=org.delta.pchat, production=false/true
```

## 工作原理

```
docker-compose up -d
  └→ entrypoint.sh
       ├→ 读取 APNS_* 环境变量（docker-compose.yaml 已预设）
       ├→ 自动注入 APNs 配置到 sync.yaml
       └→ runall-docker.sh 启动所有服务（包括 sync）
            └→ Sync 服务初始化 APNs 客户端，开始接收推送请求
```

## 故障排查

| 问题 | 解决 |
|------|------|
| 没有 "APNs: client initialized" 日志 | 检查 p8 文件: `ls teamgramd/etc/AuthKey_*.p8` |
| BadDeviceToken | 设备需要重新注册 token |
| 401 Unauthorized | 检查 KeyID/TeamID 是否匹配 p8 文件 |

## 相关文档

- 完整文档: `docs/APNS_PUSH.md`
- 配置清单: `APNS_CONFIGURATION.md`
