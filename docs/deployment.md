# Teamgram 部署文档

## 架构说明

```
客户端 ──TCP──> Gateway (10443/5222/8801) ──gRPC──> Session ──gRPC──> BFF ──gRPC──> 各服务
                                                                           │
                                                               MySQL / Redis / Kafka / etcd / MinIO
```

- **环境服务**（MySQL、Redis 等）：使用 bridge 网络，端口只绑定 `127.0.0.1`，外部不可访问
- **配置自动生成**：`entrypoint.sh` 从 `teamgramd/etc/` 模板 + 环境变量自动生成 `etc2/`，无需手动维护

### 开发 vs 生产

| | 开发环境 | 生产环境 |
|---|------|------|
| 系统 | Windows/macOS (Docker Desktop) | 原生 Linux 服务器 |
| 网络模式 | bridge（默认） | host |
| 客户端 IP | Docker 内网 IP（城市群不可用） | 真实 IP（城市群可用） |
| 启动命令 | `docker-compose up -d` | `docker-compose -f docker-compose.yaml -f docker-compose.prod.yaml up -d` |

> **注意**：`network_mode: host` 仅在原生 Linux 上有效。Docker Desktop (Windows/macOS) 的 host 模式指向虚拟机，无法访问宿主机服务。

## 端口说明

### 客户端端口（需要对外开放）

| 端口 | 协议 | 说明 |
|------|------|------|
| 10443 | TCP | MTProto 主端口 |
| 5222 | TCP | MTProto 备用端口 |
| 8801 | TCP | MTProto HTTP 端口 |

### 内部端口（生产环境禁止对外开放）

| 端口 | 服务 | 说明 |
|------|------|------|
| 20010 | BFF | gRPC 服务 |
| 20110 | Gateway | gRPC 管理 |
| 20120 | Session | gRPC 服务 |
| 20450 | Sync | gRPC 服务 |
| 20640-20670 | Biz 服务 | gRPC 内部通信 |
| 20420 | Msg | gRPC 服务 |
| 20020-20030 | 其他服务 | gRPC 内部通信 |
| 6061-6063 | Debug | pprof 监控 |

## 开发环境部署

```bash
# 启动环境服务
docker-compose -f docker-compose-env.yaml up -d

# 等待 MySQL 初始化完成（首次约 30 秒）
docker logs mysql 2>&1 | tail -5

# 构建并启动 teamgram
docker-compose build && docker-compose up -d
```

## 生产环境部署（Linux 服务器）

### 1. 配置防火墙

```bash
# 安装 ufw
sudo apt install ufw -y

# 默认策略：拒绝所有入站，允许所有出站
sudo ufw default deny incoming
sudo ufw default allow outgoing

# 允许 SSH（重要！先加这条，否则会断开连接）
sudo ufw allow 22/tcp

# 允许客户端端口
sudo ufw allow 10443/tcp    # MTProto 主端口
sudo ufw allow 5222/tcp     # MTProto 备用端口
sudo ufw allow 8801/tcp     # MTProto HTTP 端口

# 启用防火墙
sudo ufw enable

# 确认规则
sudo ufw status verbose
```

预期输出：
```
Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW IN    Anywhere
10443/tcp                  ALLOW IN    Anywhere
5222/tcp                   ALLOW IN    Anywhere
8801/tcp                   ALLOW IN    Anywhere
22/tcp (v6)                ALLOW IN    Anywhere (v6)
10443/tcp (v6)             ALLOW IN    Anywhere (v6)
5222/tcp (v6)              ALLOW IN    Anywhere (v6)
8801/tcp (v6)              ALLOW IN    Anywhere (v6)
```

### 2. 启动环境服务

```bash
docker-compose -f docker-compose-env.yaml up -d
```

等待 MySQL 初始化完成（首次约 30 秒）：
```bash
docker logs mysql 2>&1 | tail -5
# 看到 "ready for connections" 即可
```

### 3. 构建并启动 teamgram（生产模式）

```bash
docker-compose -f docker-compose.yaml -f docker-compose.prod.yaml build
docker-compose -f docker-compose.yaml -f docker-compose.prod.yaml up -d
```

可以设置 alias 简化命令：
```bash
alias dc-prod='docker-compose -f docker-compose.yaml -f docker-compose.prod.yaml'
# 之后使用：
dc-prod build && dc-prod up -d
```

### 4. 验证部署

```bash
# 确认端口监听（host 模式下 docker ps 不显示端口，用 ss 查看）
ss -tlnp | grep 10443

# 检查 BFF 日志
docker exec teamgram-server-teamgram-1 tail -20 /app/logs/bff/$(date -u +%Y-%m-%d).log

# 验证客户端 IP 获取（注册新用户后）
docker exec teamgram-server-teamgram-1 grep "autoJoinGroups" /app/logs/bff/*.log
# 应该看到真实客户端 IP，不是 172.20.0.x

# 检查防火墙（从外部机器测试）
# nc -zv <服务器IP> 20010    应该超时/拒绝
# nc -zv <服务器IP> 10443    应该成功
```

## 更新部署

```bash
git pull

# 开发环境
docker-compose build && docker-compose up -d

# 生产环境
docker-compose -f docker-compose.yaml -f docker-compose.prod.yaml build
docker-compose -f docker-compose.yaml -f docker-compose.prod.yaml up -d
```

## 环境变量参考

生产模式的环境变量在 `docker-compose.prod.yaml` 中配置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_PASSWORD` | `teamgram` | MySQL 密码 |
| `ETCD_URL` | `127.0.0.1:2379` | etcd 地址 |
| `REDIS_HOST` | `127.0.0.1:6379` | Redis 地址 |
| `KAFKA_HOST` | `127.0.0.1:9092` | Kafka 地址 |
| `MINIO_URI` | `127.0.0.1:9000` | MinIO 地址 |
| `MINIO_KEY` | `minio` | MinIO Access Key |
| `MINIO_SECRET` | `miniostorage` | MinIO Secret Key |

开发模式使用 `entrypoint.sh` 中的默认值（Docker DNS 名称）。

## 重置数据

> **警告：会清除所有用户数据！**

```bash
docker-compose down
docker-compose -f docker-compose-env.yaml down

rm -rf data/

docker-compose -f docker-compose-env.yaml up -d
sleep 30
docker-compose build && docker-compose up -d
```

## 故障排查

### 查看各服务日志

```bash
# 进入容器
docker exec -it teamgram-server-teamgram-1 bash

# 各服务日志路径
ls /app/logs/

# BFF 业务日志（go-zero file logger，按日期分文件）
cat /app/logs/bff/*.log | tail -50

# Gateway 启动日志
cat /app/logs/gateway.log

# 搜索错误
grep -r "error\|panic" /app/logs/bff/*.log | tail -20
```

### 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| 城市群没有创建 | clientAddr 是内网/Docker IP | 生产环境使用 host 模式，客户端从公网连接 |
| 欢迎消息看不到 | 消息通过 Kafka 异步投递 | 代码已加 3 秒延迟，等待客户端就绪 |
| group_assistant 在线时间异常 | UserTypeService 未跳过状态加载 | 已修复，`user.go` 跳过 Service 类型 |
| 小助手名字乱码 | SQL 未设置 utf8mb4 | `z_init.sql` 已加 `SET NAMES utf8mb4` |
| auto_groups 表不存在 | CREATE TABLE 在错误的数据库 | `1_teamgram.sql` 已修复位置 |
| host 模式下服务连不上 | Docker Desktop 不支持 host 模式 | 开发环境用 bridge 模式，仅生产 Linux 用 host |
| 容器内部端口被外部访问 | 防火墙未配置 | 按本文档防火墙步骤配置 ufw |
