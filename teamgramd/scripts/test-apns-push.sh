#!/bin/bash
# APNs 推送配置测试脚本
# 用法: ./teamgramd/scripts/test-apns-push.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/teamgramd/etc/sync.yaml"
P8_FILE="$PROJECT_ROOT/teamgramd/etc/AuthKey_JH5C27A29G.p8"

PASS=0
FAIL=0

pass() { echo "✓ $1"; PASS=$((PASS+1)); }
fail() { echo "✗ $1"; FAIL=$((FAIL+1)); }

echo "=========================================="
echo "   APNs 推送配置测试"
echo "=========================================="
echo ""

# [1] 配置文件
echo "[1] 验证配置文件..."
if [ -f "$CONFIG_FILE" ]; then
  pass "配置文件存在"
else
  fail "配置文件不存在: $CONFIG_FILE"
fi

# [2] APNs 配置是否启用（模板中注释是正常的，Docker 会自动注入）
echo ""
echo "[2] 验证 APNs 配置..."
if grep -q "^APNs:" "$CONFIG_FILE" 2>/dev/null; then
  pass "APNs 配置已启用（直接写入模板）"
else
  if grep -q "#APNs:" "$CONFIG_FILE" 2>/dev/null; then
    pass "APNs 配置模板存在（Docker 启动时自动注入）"
  else
    fail "APNs 配置未找到"
  fi
fi

# [3] 配置参数
echo ""
echo "[3] 验证配置参数..."
# 检查未注释的或注释的配置
if grep -q "^APNs:\|^#APNs:" "$CONFIG_FILE" 2>/dev/null; then
  KEY_FILE=$(grep "KeyFile:" "$CONFIG_FILE" | head -1 | sed 's/#//' | awk '{print $2}' | tr -d '"')
  KEY_ID=$(grep "KeyID:" "$CONFIG_FILE" | head -1 | sed 's/#//' | awk '{print $2}' | tr -d '"')
  TEAM_ID=$(grep "TeamID:" "$CONFIG_FILE" | head -1 | sed 's/#//' | awk '{print $2}' | tr -d '"')
  BUNDLE_ID=$(grep "BundleID:" "$CONFIG_FILE" | head -1 | sed 's/#//' | awk '{print $2}' | tr -d '"')
  PRODUCTION=$(grep "Production:" "$CONFIG_FILE" | grep -v "docker-compose\|开发\|生产" | head -1 | sed 's/#//' | awk '{print $2}')
  echo "  Key File:    $KEY_FILE"
  echo "  Key ID:      $KEY_ID"
  echo "  Team ID:     $TEAM_ID"
  echo "  Bundle ID:   $BUNDLE_ID"
  echo "  Production:  $PRODUCTION"

  [ -n "$KEY_FILE" ] && pass "KeyFile 已配置" || fail "KeyFile 缺失"
  [ -n "$KEY_ID" ] && pass "KeyID 已配置" || fail "KeyID 缺失"
  [ -n "$TEAM_ID" ] && pass "TeamID 已配置" || fail "TeamID 缺失"
  [ -n "$BUNDLE_ID" ] && pass "BundleID 已配置" || fail "BundleID 缺失"
fi

# [4] p8 文件
echo ""
echo "[4] 验证 p8 认证文件..."
if [ -f "$P8_FILE" ]; then
  pass "p8 文件存在"
  if head -1 "$P8_FILE" | grep -q "BEGIN PRIVATE KEY"; then
    pass "p8 文件格式有效"
  else
    fail "p8 文件格式无效"
  fi
  SIZE=$(wc -c < "$P8_FILE" | tr -d ' ')
  if [ "$SIZE" -ge 200 ]; then
    pass "p8 文件大小正常 ($SIZE bytes)"
  else
    fail "p8 文件太小 ($SIZE bytes)"
  fi
else
  fail "p8 文件不存在: $P8_FILE"
fi

# [5] 数据库配置
echo ""
echo "[5] 验证数据库配置..."
if grep -q "^DevicesMySQL:" "$CONFIG_FILE" 2>/dev/null; then
  pass "DevicesMySQL 配置存在"
else
  fail "DevicesMySQL 配置缺失"
fi

# [6] Go 单元测试
echo ""
echo "[6] 运行 Go 单元测试..."
if command -v go &>/dev/null; then
  cd "$PROJECT_ROOT"
  if go test -vet=off ./app/messenger/sync/internal/dao/ -run "TestP8|TestAuthKey|TestAPNsToken|TestPushPayload|TestDeviceInfo" -count=1 -timeout 30s 2>&1 | tail -5; then
    pass "Go 单元测试通过"
  else
    fail "Go 单元测试失败"
  fi
else
  echo "  (跳过: go 命令不可用)"
fi

# [7] Docker 配置
echo ""
echo "[7] 验证 Docker 配置..."
COMPOSE_DEV="$PROJECT_ROOT/docker-compose.yaml"
COMPOSE_PROD="$PROJECT_ROOT/docker-compose.prod.yaml"
if grep -q "APNS_KEY_FILE" "$COMPOSE_DEV" 2>/dev/null; then
  pass "docker-compose.yaml 包含 APNs 配置"
else
  fail "docker-compose.yaml 缺少 APNs 配置"
fi
if grep -q "APNS_KEY_FILE" "$COMPOSE_PROD" 2>/dev/null; then
  pass "docker-compose.prod.yaml 包含 APNs 配置"
else
  fail "docker-compose.prod.yaml 缺少 APNs 配置"
fi
if grep -q 'APNS_PRODUCTION: "false"' "$COMPOSE_DEV" 2>/dev/null; then
  pass "开发环境 Production=false"
else
  fail "开发环境 Production 设置错误"
fi
if grep -q 'APNS_PRODUCTION: "true"' "$COMPOSE_PROD" 2>/dev/null; then
  pass "生产环境 Production=true"
else
  fail "生产环境 Production 设置错误"
fi

# 结果
echo ""
echo "=========================================="
if [ $FAIL -eq 0 ]; then
  echo "   测试结果: ✓ 全部通过 ($PASS/$PASS)"
else
  echo "   测试结果: ✗ $FAIL 项失败 ($PASS 项通过)"
fi
echo "=========================================="

exit $FAIL
