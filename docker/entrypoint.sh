#!/bin/sh
set -e

CONFIG_DIR="/opt/workflow/ferry/config"
DEFAULT_DIR="/opt/workflow/ferry/default_config"

# 1. 首次启动时复制默认配置文件
if [ ! -f "$CONFIG_DIR/settings.yml" ]; then
    echo ">>> 未检测到 settings.yml，复制默认配置..."
    cp "$DEFAULT_DIR"/* "$CONFIG_DIR/"
fi

# 2. 始终更新 db.sql 和 ferry.sql（确保种子数据与当前版本一致）
cp -f "$DEFAULT_DIR/db.sql" "$CONFIG_DIR/db.sql" 2>/dev/null || true
cp -f "$DEFAULT_DIR/ferry.sql" "$CONFIG_DIR/ferry.sql" 2>/dev/null || true

# 3. 判断是否需要初始化数据库
NEED_INIT=false

# 3a. 强制初始化：检查 needinit 标记文件
if [ -f "$CONFIG_DIR/needinit" ]; then
    NEED_INIT=true
    echo ">>> 检测到 needinit 标记，将执行数据库初始化"
fi

# 3b. 自动初始化：检查 .initialized 标记文件是否存在
#     首次部署时 .initialized 不存在，自动触发初始化
#     初始化成功后创建该文件，后续重启不再初始化
if [ "$NEED_INIT" = "false" ] && [ ! -f "$CONFIG_DIR/.initialized" ]; then
    NEED_INIT=true
    echo ">>> 未检测到 .initialized 标记，首次部署，将执行数据库初始化"
fi

# 4. 执行数据库初始化
if [ "$NEED_INIT" = "true" ]; then
    /opt/workflow/ferry/ferry init -c="$CONFIG_DIR/settings.yml"
    rm -f "$CONFIG_DIR/needinit"
    touch "$CONFIG_DIR/.initialized"
    echo ">>> 数据库初始化完成"
fi

# 5. 启动服务
/opt/workflow/ferry/ferry server -c="$CONFIG_DIR/settings.yml"
#!/bin/sh
set -e

CONFIG_DIR="/opt/workflow/ferry/config"
DEFAULT_DIR="/opt/workflow/ferry/default_config"

# 1. 首次启动时复制默认配置文件
if [[ ! -f "$CONFIG_DIR/settings.yml" ]]; then
    echo ">>> 未检测到 settings.yml，复制默认配置..."
    cp "$DEFAULT_DIR"/* "$CONFIG_DIR/"
fi

# 2. 始终更新 db.sql 和 ferry.sql（确保种子数据与当前版本一致）
cp -f "$DEFAULT_DIR/db.sql" "$CONFIG_DIR/db.sql" 2>/dev/null || true
cp -f "$DEFAULT_DIR/ferry.sql" "$CONFIG_DIR/ferry.sql" 2>/dev/null || true

# 3. 判断是否需要初始化数据库
NEED_INIT=false

# 3a. 优先检查 needinit 标记文件（deploy.sh 首次部署时创建）
if [[ -f "$CONFIG_DIR/needinit" ]]; then
    NEED_INIT=true
    echo ">>> 检测到 needinit 标记，将执行数据库初始化"
fi

# 3b. 检查 sys_settings 表是否存在（兜底：防止 settings.yml 已存在但数据库为空）
if [ "$NEED_INIT" = "false" ] && command -v mysql >/dev/null 2>&1; then
    DB_HOST=$(grep -A5 'database' "$CONFIG_DIR/settings.yml" | grep 'host' | head -1 | awk -F': ' '{print $2}' | tr -d ' "'"'"'')
    DB_PORT=$(grep -A5 'database' "$CONFIG_DIR/settings.yml" | grep 'port' | head -1 | awk -F': ' '{print $2}' | tr -d ' "'"'"'')
    DB_NAME=$(grep -A5 'database' "$CONFIG_DIR/settings.yml" | grep 'name' | head -1 | awk -F': ' '{print $2}' | tr -d ' "'"'"'')
    DB_USER=$(grep -A5 'database' "$CONFIG_DIR/settings.yml" | grep 'username' | head -1 | awk -F': ' '{print $2}' | tr -d ' "'"'"'')
    DB_PASS=$(grep -A5 'database' "$CONFIG_DIR/settings.yml" | grep 'password' | head -1 | awk -F': ' '{print $2}' | tr -d ' "'"'"'')
    if [ -n "$DB_HOST" ] && [ -n "$DB_NAME" ]; then
        TABLE_EXISTS=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -N -e \
            "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='$DB_NAME' AND table_name='sys_settings'" 2>/dev/null || echo "1")
        if [ "$TABLE_EXISTS" = "0" ]; then
            NEED_INIT=true
            echo ">>> 检测到 sys_settings 表不存在，将执行数据库初始化"
        fi
    fi
fi

# 4. 执行数据库初始化
if [ "$NEED_INIT" = "true" ]; then
    /opt/workflow/ferry/ferry init -c="$CONFIG_DIR/settings.yml"
    rm -f "$CONFIG_DIR/needinit"
    echo ">>> 数据库初始化完成"
fi

# 5. 启动服务
/opt/workflow/ferry/ferry server -c="$CONFIG_DIR/settings.yml"
