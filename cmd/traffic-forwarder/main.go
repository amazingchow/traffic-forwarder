package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	_ConfigFile = flag.String("conf", "./etc/traffic-forwarder.conf", "The path of the configuration file.")
	_Timeout    = flag.Duration("timeout", 30*time.Second, "Connection timeout")
	_MaxConns   = flag.Int("max-conns", 1000, "Maximum concurrent connections per port")
)

// ConnectionManager 管理连接的生命周期
type ConnectionManager struct {
	mu       sync.RWMutex
	conns    map[net.Conn]struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	maxConns int
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager(maxConns int) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConnectionManager{
		conns:    make(map[net.Conn]struct{}),
		ctx:      ctx,
		cancel:   cancel,
		maxConns: maxConns,
	}
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn net.Conn) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.conns) >= cm.maxConns {
		return false
	}

	cm.conns[conn] = struct{}{}
	cm.wg.Add(1)
	return true
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(conn net.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.conns[conn]; exists {
		delete(cm.conns, conn)
		conn.Close()
		cm.wg.Done()
	}
}

// CloseAll 关闭所有连接
func (cm *ConnectionManager) CloseAll() {
	cm.cancel()
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for conn := range cm.conns {
		conn.Close()
		cm.wg.Done()
	}
	cm.conns = make(map[net.Conn]struct{})
}

// Wait 等待所有连接关闭
func (cm *ConnectionManager) Wait() {
	cm.wg.Wait()
}

// 全局连接管理器
var globalConnManager *ConnectionManager

// ForwardingRule 转发规则
type ForwardingRule struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
}

// RunTrafficForwarder 运行流量转发器
func RunTrafficForwarder(configFile string) bool {
	logrus.Infof("Loading setting file:%s.", configFile)
	fin, err := os.Open(configFile)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to load setting file:%s.", configFile)
		return false
	}
	defer fin.Close()

	var rules []ForwardingRule
	scanner := bufio.NewScanner(fin)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		if ok := strings.HasPrefix(line, "#"); ok {
			continue
		}
		setting := strings.Split(line, "|")
		if len(setting) != 3 {
			logrus.Warnf("Skip invalid setting:%s.", line)
			continue
		}

		localPort, _ := strconv.Atoi(strings.TrimSpace(setting[0]))
		remoteHost := strings.TrimSpace(setting[1])
		remotePort, _ := strconv.Atoi(strings.TrimSpace(setting[2]))
		if localPort <= 0 || remotePort <= 0 {
			logrus.Warnf("Skip invalid setting:%s.", line)
			continue
		}

		logrus.Infof("Use line:'%s' to setup forwarding tunnel.", line)
		rules = append(rules, ForwardingRule{
			LocalPort:  localPort,
			RemoteHost: remoteHost,
			RemotePort: remotePort,
		})
	}
	if err = scanner.Err(); err != nil {
		logrus.WithError(err).Errorf("Failed to continue to load setting file:%s.", *_ConfigFile)
		return false
	}

	// 启动所有转发规则
	var wg sync.WaitGroup
	started := make(chan struct{}, len(rules))

	for _, rule := range rules {
		wg.Add(1)
		go func(r ForwardingRule) {
			defer wg.Done()
			startForwarding(r, started)
		}(rule)
	}

	// 等待所有转发服务启动完成
	wg.Wait()
	close(started)

	// 等待上下文取消，确保所有goroutine都能正确退出
	go func() {
		<-globalConnManager.ctx.Done()
		logrus.Info("Shutdown signal received, stopping all forwarding services.")
	}()

	return true
}

// startForwarding 启动单个转发服务
func startForwarding(rule ForwardingRule, started chan struct{}) {
	ln, err := net.Listen("tcp", fmt.Sprintf("[::]:%d", rule.LocalPort))
	if err != nil {
		logrus.WithError(err).Errorf("Failed to listen on [::]:%d.", rule.LocalPort)
		return
	}
	defer ln.Close()
	logrus.Infof("Listening on [::]:%d.", rule.LocalPort)

	// 发送启动完成信号
	started <- struct{}{}

	// 创建一个goroutine来监听上下文取消，用于优雅关闭监听器
	go func() {
		<-globalConnManager.ctx.Done()
		ln.Close()
	}()

	for {
		upstream, err := ln.Accept()
		if err != nil {
			if globalConnManager.ctx.Err() != nil {
				// 服务正在关闭
				return
			}
			logrus.WithError(err).Errorf("Failed to accept new connection on [::]:%d.", rule.LocalPort)
			continue
		}

		// 检查连接数量限制
		if !globalConnManager.AddConnection(upstream) {
			logrus.Warnf("Connection limit reached for port %d, rejecting connection from %s",
				rule.LocalPort, upstream.RemoteAddr().String())
			upstream.Close()
			continue
		}

		remote := upstream.RemoteAddr().String()
		logrus.Infof("Client<ip:%s> connected on [::]:%d.", remote, rule.LocalPort)

		go handleConnection(upstream, rule)
	}
}

// handleConnection 处理单个连接
func handleConnection(upstream net.Conn, rule ForwardingRule) {
	defer globalConnManager.RemoveConnection(upstream)

	remote := upstream.RemoteAddr().String()

	// 设置连接超时
	upstream.SetDeadline(time.Now().Add(*_Timeout))

	// 连接到远程服务器
	downstream, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", rule.RemoteHost, rule.RemotePort), *_Timeout)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to connect to %s:%d for client<ip:%s>.",
			rule.RemoteHost, rule.RemotePort, remote)
		return
	}
	defer downstream.Close()

	// 设置下游连接超时
	downstream.SetDeadline(time.Now().Add(*_Timeout))

	// 添加到连接管理器 - 修复：只有在连接成功后才添加
	globalConnManager.AddConnection(downstream)
	defer globalConnManager.RemoveConnection(downstream)

	logrus.Infof("Forwarding traffic from 0.0.0.0:%d to %s:%d for client<ip:%s>.",
		rule.LocalPort, rule.RemoteHost, rule.RemotePort, remote)

	// 创建上下文用于控制传输
	ctx, cancel := context.WithCancel(globalConnManager.ctx)
	defer cancel()

	// 使用改进的传输函数 - 修复：使用sync.WaitGroup确保两个goroutine都完成
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer cancel()
		TransferWithContext(ctx, downstream, upstream)
	}()

	go func() {
		defer wg.Done()
		defer cancel()
		TransferWithContext(ctx, upstream, downstream)
	}()

	// 等待传输完成或上下文取消
	select {
	case <-ctx.Done():
		// 上下文被取消，等待goroutine完成
		wg.Wait()
	case <-time.After(*_Timeout):
		// 超时保护，避免goroutine泄漏
		cancel()
		wg.Wait()
	}
}

// TransferWithContext 带上下文的传输函数
func TransferWithContext(ctx context.Context, dst io.Writer, src io.Reader) {
	// 使用带缓冲的传输来减少内存分配
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 设置读取超时
			if conn, ok := src.(net.Conn); ok {
				conn.SetReadDeadline(time.Now().Add(*_Timeout))
			}

			n, err := src.Read(buffer)
			if n > 0 {
				if conn, ok := dst.(net.Conn); ok {
					conn.SetWriteDeadline(time.Now().Add(*_Timeout))
				}

				_, writeErr := dst.Write(buffer[:n])
				if writeErr != nil {
					logrus.WithError(writeErr).Debug("Write error during transfer")
					return
				}
			}

			if err != nil {
				if err != io.EOF {
					logrus.WithError(err).Debug("Read error during transfer")
				}
				return
			}
		}
	}
}

func main() {
	flag.Parse()

	if *_ConfigFile == "" {
		logrus.Error("No configuration file provided.")
		return
	}
	if *_MaxConns <= 0 {
		logrus.Error("Invalid maximum concurrent connections.")
		return
	}
	if *_MaxConns > 4096 {
		logrus.Error("Maximum concurrent connections is too large.")
		return
	}

	// 初始化全局连接管理器
	globalConnManager = NewConnectionManager(*_MaxConns)
	defer globalConnManager.CloseAll()

	if RunTrafficForwarder(*_ConfigFile) {
		logrus.Info("Service started.")
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-signalCh
		logrus.Warning("Service stopped.")

		// 等待所有连接优雅关闭，添加超时保护
		done := make(chan struct{})
		go func() {
			globalConnManager.Wait()
			close(done)
		}()

		select {
		case <-done:
			logrus.Info("All connections closed gracefully.")
		case <-time.After(10 * time.Second):
			logrus.Warning("Timeout waiting for connections to close, forcing shutdown.")
		}
	} else {
		logrus.Error("Failed to start service.")
	}
}
