# Traffic Forwarder

A high-performance TCP traffic forwarding service implemented in Go, designed for efficient network traffic routing between local and remote endpoints.

## Overview

Traffic Forwarder is a robust, memory-optimized TCP proxy that forwards traffic from local sockets to remote sockets with minimal latency and resource consumption. Built with Go 1.21+, it leverages zero-copy techniques and advanced connection management for optimal performance in production environments.

## Key Features

- **High-Performance Forwarding**: Zero-copy data transmission for maximum throughput
- **Memory Optimization**: Connection pooling, buffer reuse, and timeout management
- **Resource Management**: Automatic connection cleanup and graceful shutdown
- **Configurable**: Customizable connection timeouts, connection limits, and routing rules
- **Observability**: Comprehensive logging and performance metrics

## Architecture

### Connection Management
- Connection pool management with configurable limits
- Automatic cleanup of disconnected sessions
- Thread-safe connection handling with `sync.WaitGroup`

### Buffer Optimization
- Fixed-size buffer allocation (32KB) to minimize memory overhead
- Buffer reuse mechanisms to reduce garbage collection pressure
- Efficient memory utilization patterns

### Timeout Control
- Configurable read/write timeouts for all connections
- Automatic cleanup of stale connections
- Resource protection against long-running operations

### Context Management
- `context.Context` integration for lifecycle control
- Graceful shutdown capabilities
- Goroutine leak prevention

## Requirements

- Go 1.21 or later
- Linux, macOS, or Windows
- Network access to target endpoints

## Installation

### From Source

```bash
git clone https://github.com/amazingchow/traffic-forwarder.git
cd traffic-forwarder
make build
```

## Quick Start

### Basic Usage

1. **Start the service:**
   ```bash
   make start
   ```

2. **Stop the service:**
   ```bash
   make stop
   ```

3. **View logs:**
   ```bash
   tail -f -n50 nohup.out
   ```

### Optimized Build

For production deployments with enhanced memory management:

```bash
make build memory_optimized=1
```

## Configuration

### Command Line Options

```bash
./traffic-forwarder [options]

Options:
  -conf string
        Configuration file path (default: "./etc/traffic-forwarder.conf")
  -timeout duration
        Connection timeout (default: 30s)
  -max-conns int
        Maximum concurrent connections per port (default: 1000)
```

### Configuration File Format

The configuration file uses a simple pipe-delimited format:

```
# Format: local_port | remote_host | remote_port
18080 | 127.0.0.1 | 8080
18081 | 192.168.1.100 | 3306
```

## Performance Monitoring

### Memory Optimization Guidelines

1. **Connection Limits**: Adjust `-max-conns` based on available system resources
2. **Timeout Configuration**: Set appropriate timeouts for your network environment
3. **Memory Profiling**: Use `go tool pprof` for memory usage analysis
4. **Service Rotation**: Implement periodic service restarts in production

### Monitoring Metrics

- Connection count and status
- Memory usage patterns
- Throughput and latency metrics
- Error rates and failure modes

### Production Considerations

- Use reverse proxy (nginx, haproxy) for load balancing
- Implement health checks and monitoring
- Configure appropriate resource limits
- Set up log aggregation and alerting
