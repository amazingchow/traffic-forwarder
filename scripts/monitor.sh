#!/bin/bash

# 内存监控脚本
# 用于监控 traffic-forwarder 的内存使用情况

set -e

# 配置
PROCESS_NAME="traffic-forwarder"
LOG_FILE="/tmp/traffic-forwarder-monitor.log"
ALERT_THRESHOLD_MB=500  # 内存使用超过500MB时告警
CHECK_INTERVAL=30       # 检查间隔（秒）

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

# 获取进程ID
get_pid() {
    pgrep -f "$PROCESS_NAME" | head -1
}

# 获取内存使用（MB）
get_memory_usage() {
    local pid=$1
    if [ -n "$pid" ]; then
        ps -p "$pid" -o rss= | awk '{print int($1/1024)}'
    else
        echo "0"
    fi
}

# 获取CPU使用率
get_cpu_usage() {
    local pid=$1
    if [ -n "$pid" ]; then
        ps -p "$pid" -o %cpu= | awk '{print $1}'
    else
        echo "0"
    fi
}

# 获取连接数
get_connection_count() {
    local pid=$1
    if [ -n "$pid" ]; then
        lsof -p "$pid" | grep -c ESTABLISHED || echo "0"
    else
        echo "0"
    fi
}

# 检查进程是否存在
check_process() {
    local pid
    pid=$(get_pid)
    if [ -n "$pid" ]; then
        echo "$pid"
    else
        echo ""
    fi
}

# 监控主循环
monitor() {
    log "开始监控 $PROCESS_NAME"
    
    while true; do
        local pid
        pid=$(check_process)
        
        if [ -n "$pid" ]; then
            local memory_mb
            local cpu_percent
            local connections
            
            memory_mb=$(get_memory_usage "$pid")
            cpu_percent=$(get_cpu_usage "$pid")
            connections=$(get_connection_count "$pid")
            
            # 输出状态
            echo -e "${GREEN}✓${NC} 进程运行中 (PID: $pid)"
            echo -e "  内存使用: ${YELLOW}${memory_mb}MB${NC}"
            echo -e "  CPU使用: ${YELLOW}${cpu_percent}%${NC}"
            echo -e "  连接数: ${YELLOW}${connections}${NC}"
            
            # 检查内存使用是否超过阈值
            if [ "$memory_mb" -gt "$ALERT_THRESHOLD_MB" ]; then
                log "⚠️  警告: 内存使用超过 ${ALERT_THRESHOLD_MB}MB (当前: ${memory_mb}MB)"
                echo -e "${RED}⚠️  内存使用过高!${NC}"
            fi
            
            # 记录到日志
            log "PID: $pid, 内存: ${memory_mb}MB, CPU: ${cpu_percent}%, 连接: ${connections}"
            
        else
            log "❌ 进程未运行"
            echo -e "${RED}❌ $PROCESS_NAME 进程未运行${NC}"
        fi
        
        echo "----------------------------------------"
        sleep "$CHECK_INTERVAL"
    done
}

# 性能分析
profile() {
    local pid
    pid=$(get_pid)
    if [ -z "$pid" ]; then
        echo "进程未运行"
        exit 1
    fi
    
    echo "开始性能分析..."
    
    # 获取内存映射
    echo "内存映射:"
    head -20 < /proc/"$pid"/maps
    
    echo ""
    echo "打开的文件描述符:"
    lsof -p "$pid" | head -10
    
    echo ""
    echo "进程状态:"
    grep -E "(VmRSS|VmSize|Threads)" < /proc/"$pid"/status
}

# 清理日志
cleanup() {
    echo "清理监控日志..."
    rm -f "$LOG_FILE"
    echo "日志已清理"
}

# 显示帮助
show_help() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  monitor    启动监控 (默认)"
    echo "  profile    显示性能分析信息"
    echo "  cleanup    清理监控日志"
    echo "  help       显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 monitor    # 启动监控"
    echo "  $0 profile    # 显示性能信息"
    echo "  $0 cleanup    # 清理日志"
}

# 主函数
main() {
    case "${1:-monitor}" in
        "monitor")
            monitor
            ;;
        "profile")
            profile
            ;;
        "cleanup")
            cleanup
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            echo "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
}

# 捕获中断信号
trap 'echo -e "\n${YELLOW}监控已停止${NC}"; exit 0' INT TERM

# 运行主函数
main "$@" 