# A TCP Traffic Forwarder

Golang implementation of 「A TCP Traffic Forwarder」, which does forward traffic from local socket to remote socket for client. The code requires Golang version 1.21 or newer.

## 特性 (Features)

- **高性能转发**: 使用零拷贝技术进行数据传输
- **内存优化**: 连接池管理、缓冲区复用、超时控制
- **资源管理**: 自动连接清理、优雅关闭
- **可配置**: 支持连接超时、最大连接数等参数
- **监控友好**: 详细的日志记录和性能指标

## 内存优化改进 (Memory Optimization Improvements)

### 1. 连接管理 (Connection Management)
- 实现连接池管理，限制最大并发连接数
- 自动清理断开的连接，避免内存泄漏
- 使用 `sync.WaitGroup` 确保所有连接正确关闭

### 2. 缓冲区优化 (Buffer Optimization)
- 使用固定大小的缓冲区 (32KB) 减少内存分配
- 避免频繁的内存分配和垃圾回收
- 实现缓冲区复用机制

### 3. 超时控制 (Timeout Control)
- 为所有连接设置读写超时
- 防止连接长时间占用资源
- 自动清理超时连接

### 4. 上下文管理 (Context Management)
- 使用 `context.Context` 控制数据传输生命周期
- 支持优雅关闭和取消操作
- 避免 goroutine 泄漏

## 安装 (Installation)

```shell
git clone https://github.com/amazingchow/traffic-forwarder.git
cd /path/to/traffic-forwarder
make build
```

## 使用方法 (Usage)

### 基本使用 (Basic Usage)

启动转发器:
```shell
cd /path/to/traffic-forwarder
make start
```

停止转发器:
```shell
cd /path/to/traffic-forwarder
make stop
```

查看日志:
```shell
cd /path/to/traffic-forwarder
tail -f -n50 nohup.out
```

### 优化版本 (Optimized Version)

使用内存优化版本:
```shell
make build-optimized
make local_run_optimized
```

### 命令行参数 (Command Line Options)

```shell
./traffic-forwarder [options]

Options:
  -conf string
        The path of the configuration file. (default "./etc/traffic-forwarder.conf")
  -timeout duration
        Connection timeout (default 30s)
  -max-conns int
        Maximum concurrent connections per port (default 1000)
```

### 配置文件格式 (Configuration Format)

配置文件格式: `local_port | remote_host | remote_port`

示例:
```
# local port | remote host | remote port
18080 | 127.0.0.1 | 8080
18081 | 192.168.1.100 | 3306
```

## 构建选项 (Build Options)

### 内存优化构建 (Memory Optimized Build)
```shell
make build-optimized
```

### 调试构建 (Debug Build)
```shell
make build-debug
```

### 性能分析构建 (Profiling Build)
```shell
make build profile=1
```

## 性能监控 (Performance Monitoring)

### 运行基准测试 (Run Benchmarks)
```shell
make bench
```

### 运行竞态检测 (Run Race Detection)
```shell
make test-race
```

### 代码质量检查 (Code Quality)
```shell
make lint
make vet
make fmt
```

## Docker 支持 (Docker Support)

构建 Docker 镜像:
```shell
make docker-build
```

运行 Docker 容器:
```shell
make docker-run
```

停止 Docker 容器:
```shell
make docker-stop
```

## 内存使用优化建议 (Memory Usage Optimization Tips)

1. **调整连接数限制**: 根据服务器资源调整 `-max-conns` 参数
2. **设置合适的超时**: 根据网络环境调整 `-timeout` 参数
3. **监控内存使用**: 使用 `go tool pprof` 分析内存使用情况
4. **定期重启**: 在生产环境中定期重启服务以释放内存

## 故障排除 (Troubleshooting)

### 常见问题 (Common Issues)

1. **连接数过多**: 调整 `-max-conns` 参数
2. **内存使用过高**: 检查是否有连接泄漏，调整超时设置
3. **性能问题**: 使用性能分析工具定位瓶颈

### 日志级别 (Log Levels)

- `Info`: 正常操作信息
- `Warn`: 警告信息（如无效配置）
- `Error`: 错误信息（如连接失败）
- `Debug`: 调试信息（如传输错误）

## 贡献 (Contributing)

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 许可证 (License)

本项目采用 MIT 许可证。
