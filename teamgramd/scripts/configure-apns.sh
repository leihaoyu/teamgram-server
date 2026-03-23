#!/bin/bash
# APNs 自动化配置脚本
# 用法: ./teamgramd/scripts/configure-apns.sh [--production]

set -e

# 默认配置
KEY_ID="JH5C27A29G"
TEAM_ID="3WA4Q9D2GD"
BUNDLE_ID="org.delta.pchat"
PRODUCTION="false"

# 解析参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --key-file) P8_FILE="$2"; shift 2;;
    --key-id) KEY_ID="$2"; shift 2;;
    --team-id) TEAM_ID="$2"; shift 2;;
    --bundle-id) BUNDLE_ID="$2"; shift 2;;
    --production) PRODUCTION="true"; shift;;
    *) echo "未知参数: $1"; exit 1;;
  esac
done

# 定位项目根目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/teamgramd/etc/sync.yaml"
P8_FILE="${P8_FILE:-$PROJECT_ROOT/teamgramd/etc/AuthKey_${KEY_ID}.p8}"

echo "=========================================="
echo "   APNs 自动化配置"
echo "=========================================="
echo "项目根目录: $PROJECT_ROOT"
echo "配置文件:   $CONFIG_FILE"
echo "p8文件:     $P8_FILE"
echo "Key ID:     $KEY_ID"
echo "Team ID:    $TEAM_ID"
echo "Bundle ID:  $BUNDLE_ID"
echo "Production: $PRODUCTION"
echo ""

# 验证 p8 文件
if [ ! -f "$P8_FILE" ]; then
  echo "✗ p8 文件不存在: $P8_FILE"
  exit 1
fi

if ! head -1 "$P8_FILE" | grep -q "BEGIN PRIVATE KEY"; then
  echo "✗ p8 文件格式无效"
  exit 1
fi
echo "✓ p8 文件有效"

# 验证配置文件
if [ ! -f "$CONFIG_FILE" ]; then
  echo "✗ 配置文件不存在: $CONFIG_FILE"
  exit 1
fi
echo "✓ 配置文件存在"

# 备份配置文件
BACKUP_FILE="${CONFIG_FILE}.bak.$(date +%Y%m%d%H%M%S)"
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✓ 配置备份: $BACKUP_FILE"

# 计算相对路径 (从 bin/ 目录运行，所以用 ../etc/)
RELATIVE_P8="../etc/$(basename "$P8_FILE")"

# 移除现有的 APNs 注释块和配置
sed -i.tmp '/^# APNs 推送配置/,/^#  Production:/d' "$CONFIG_FILE"
sed -i.tmp '/^APNs:/,/^  Production:/d' "$CONFIG_FILE"
rm -f "${CONFIG_FILE}.tmp"

# 在 DevicesMySQL 之前插入 APNs 配置
if grep -q "^DevicesMySQL:" "$CONFIG_FILE"; then
  sed -i.tmp "/^DevicesMySQL:/i\\
\\
APNs:\\
  KeyFile: \"$RELATIVE_P8\"\\
  KeyID: \"$KEY_ID\"\\
  TeamID: \"$TEAM_ID\"\\
  BundleID: \"$BUNDLE_ID\"\\
  Production: $PRODUCTION\\
" "$CONFIG_FILE"
  rm -f "${CONFIG_FILE}.tmp"
else
  # 追加到文件末尾
  cat >> "$CONFIG_FILE" <<EOF

APNs:
  KeyFile: "$RELATIVE_P8"
  KeyID: "$KEY_ID"
  TeamID: "$TEAM_ID"
  BundleID: "$BUNDLE_ID"
  Production: $PRODUCTION
EOF
fi
echo "✓ APNs 配置已写入 sync.yaml"

echo ""
echo "=========================================="
echo "   配置完成!"
echo "=========================================="
echo ""
echo "下一步:"
echo "  开发环境: docker-compose build && docker-compose up -d"
echo "  生产环境: docker-compose -f docker-compose.prod.yaml build && docker-compose -f docker-compose.prod.yaml up -d"
echo ""
echo "验证: grep 'APNs:' $CONFIG_FILE"
