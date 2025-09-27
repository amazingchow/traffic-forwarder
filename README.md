A high-performance TCP traffic forwarding tool implemented in Golang, designed for efficient network traffic routing between local and remote endpoints. It's a robust, memory-optimized TCP proxy that forwards traffic from local sockets to remote sockets with minimal latency and resource consumption. Built with Go 1.21+, it leverages zero-copy techniques and advanced connection management for optimal performance in production environments.

## Requirements

- Go 1.21 or later
- Linux, macOS, or Windows
- Network access to target endpoints

## Installation

### From Source

```bash
git clone https://github.com/amazingchow/traffic-forwarder.git
cd traffic-forwarder
mv .env.example .env
make build release=1
```

## Quick Start

### Basic Usage

1. **Update the .env file:**

   change the .env file according to your needs

2. **Start the service:**

   ```bash
   make run
   ```

3. **Stop the service:**

   ```bash
   make stop
   ```

### Optimized Build

For production deployments with enhanced memory management:

```bash
make build release=1 memory_optimized=1
```
