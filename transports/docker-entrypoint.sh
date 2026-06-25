#!/bin/sh
set -e

# Function to fix permissions on mounted volumes
fix_permissions() {
    # Check if /app/data exists and fix ownership if needed
    if [ -d "/app/data" ]; then
        # Get current user info
        CURRENT_UID=$(id -u)
        CURRENT_GID=$(id -g)
        
        # Get directory ownership
        DATA_UID=$(stat -c '%u' /app/data 2>/dev/null || echo "0")
        DATA_GID=$(stat -c '%g' /app/data 2>/dev/null || echo "0")
        
        # If ownership doesn't match current user, try to fix it
        if [ "$DATA_UID" != "$CURRENT_UID" ] || [ "$DATA_GID" != "$CURRENT_GID" ]; then
            echo "Fixing permissions on /app/data (was $DATA_UID:$DATA_GID, setting to $CURRENT_UID:$CURRENT_GID)"
            
            # Try to change ownership (will work if running as root or if user has permission)
            if chown -R "$CURRENT_UID:$CURRENT_GID" /app/data 2>/dev/null; then
                echo "Successfully updated permissions on /app/data"
            else
                echo "Warning: Could not change ownership of /app/data. You may need to run:"
                echo "  docker run --user \$(id -u):\$(id -g) ..."
                echo "  or ensure the host directory is owned by UID:GID $CURRENT_UID:$CURRENT_GID"
            fi
        fi
        
        # Ensure logs subdirectory exists with correct permissions
        mkdir -p /app/data/logs
        chmod 755 /app/data/logs 2>/dev/null || true
    fi
}

# Fix permissions before starting the application
fix_permissions

if [ -f /app/default-config.json ] && [ ! -f "$APP_DIR/config.json" ]; then
    cp /app/default-config.json "$APP_DIR/config.json"
fi

# Parse command line arguments and set environment variables
parse_args() {
    while [ $# -gt 0 ]; do
        case $1 in
            --port|-port)
                if [ -n "$2" ]; then
                    export APP_PORT="$2"
                    shift 2
                else
                    echo "Error: --port requires a value"
                    exit 1
                fi
                ;;
            --host|-host)
                if [ -n "$2" ]; then
                    export APP_HOST="$2"
                    shift 2
                else
                    echo "Error: --host requires a value"
                    exit 1
                fi
                ;;
            *)
                # Keep other arguments for the main application
                set -- "$@" "$1"
                shift
                ;;
        esac
    done
}

# Parse arguments if any are provided
if [ $# -gt 1 ]; then
    parse_args "$@"
fi

# Auto-tune the Go runtime to the container's memory limit to reduce RAM usage.
# If GOMEMLIMIT is not already set, read the cgroup memory limit (v2 then v1) and
# set a soft heap cap at ~90% of it. This makes the GC reclaim memory aggressively
# as the heap approaches the container limit instead of growing toward ~2x live heap.
if [ -z "$GOMEMLIMIT" ]; then
    CGROUP_MEM=""
    if [ -r /sys/fs/cgroup/memory.max ]; then
        # cgroup v2
        CGROUP_MEM=$(cat /sys/fs/cgroup/memory.max 2>/dev/null)
    elif [ -r /sys/fs/cgroup/memory/memory.limit_in_bytes ]; then
        # cgroup v1
        CGROUP_MEM=$(cat /sys/fs/cgroup/memory/memory.limit_in_bytes 2>/dev/null)
    fi
    # "max" (v2) or an absurdly large v1 value means "unlimited" — skip in that case.
    if [ -n "$CGROUP_MEM" ] && [ "$CGROUP_MEM" != "max" ] && [ "$CGROUP_MEM" -gt 0 ] 2>/dev/null \
        && [ "$CGROUP_MEM" -lt 1000000000000000 ] 2>/dev/null; then
        # 90% of the limit, in bytes
        GOMEMLIMIT_BYTES=$(( CGROUP_MEM / 100 * 90 ))
        export GOMEMLIMIT="${GOMEMLIMIT_BYTES}B"
        echo "Auto-set GOMEMLIMIT=$GOMEMLIMIT (90% of cgroup limit ${CGROUP_MEM} bytes)"
    fi
fi

# Default GOGC to a lower target than Go's default (100) to trade a little CPU for
# a smaller resident heap. Override by setting GOGC in the environment.
if [ -z "$GOGC" ]; then
    export GOGC=75
fi

# Build the command with environment variables and standard arguments
exec /app/main -app-dir "$APP_DIR" -port "$APP_PORT" -host "$APP_HOST" -log-level "$LOG_LEVEL" -log-style "$LOG_STYLE"
