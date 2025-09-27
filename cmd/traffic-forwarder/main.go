package main

import (
	"context"
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

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

type ConnectionManager struct {
	mu sync.RWMutex

	conns  map[net.Conn]struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	timeout  time.Duration
	maxConns int
}

type ForwardingRule struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
}

func NewConnectionManager(timeout time.Duration, maxConns int) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConnectionManager{
		conns:    make(map[net.Conn]struct{}),
		ctx:      ctx,
		cancel:   cancel,
		timeout:  timeout,
		maxConns: maxConns,
	}
}

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

func (cm *ConnectionManager) RemoveConnection(conn net.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.conns[conn]; exists {
		delete(cm.conns, conn)
		conn.Close()
		cm.wg.Done()
	}
}

func (cm *ConnectionManager) CloseAll() {
	cm.cancel()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for conn := range cm.conns {
		conn.Close()
		cm.wg.Done()
	}
	cm.conns = nil
}

func (cm *ConnectionManager) Wait() {
	cm.wg.Wait()
}

func (cm *ConnectionManager) Run() bool {
	if err := godotenv.Load(); err != nil {
		logrus.WithError(err).Fatal("Error loading .env file.")
	}

	// 读取和解析 GOTF_RULES
	rulesStr := os.Getenv("GOTF_RULES")
	var rules []ForwardingRule
	if rulesStr != "" {
		_rules := strings.Split(rulesStr, ",")
		for _, rule := range _rules {
			parts := strings.Split(rule, "|")
			if len(parts) != 3 {
				logrus.Warnf("Skip invalid rule:%s.", rule)
				continue
			}
			localPort, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			remoteHost := strings.TrimSpace(parts[1])
			remotePort, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			rules = append(rules, ForwardingRule{
				LocalPort:  localPort,
				RemoteHost: remoteHost,
				RemotePort: remotePort,
			})
		}
	}

	// 启动所有转发规则
	var wg sync.WaitGroup
	started := make(chan struct{}, len(rules))
	for _, rule := range rules {
		wg.Add(1)
		go func(r ForwardingRule) {
			defer wg.Done()
			cm.startForwarding(r, started)
		}(rule)
	}

	// 等待所有转发服务启动完成
	wg.Wait()
	close(started)

	// 等待上下文取消，确保所有goroutine都能正确退出
	go func() {
		<-cm.ctx.Done()
	}()

	return true
}

func (cm *ConnectionManager) startForwarding(rule ForwardingRule, started chan struct{}) {
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
		<-cm.ctx.Done()
		ln.Close()
	}()

	for {
		upstream, err := ln.Accept()
		if err != nil {
			if cm.ctx.Err() != nil {
				// 服务正在关闭
				return
			}
			logrus.WithError(err).Errorf("Failed to accept new connection on [::]:%d.", rule.LocalPort)
			continue
		}

		// 检查连接数量限制
		if !cm.AddConnection(upstream) {
			logrus.Warnf("Connection limit reached for port %d, rejecting connection from %s",
				rule.LocalPort, upstream.RemoteAddr().String())
			upstream.Close()
			continue
		}

		remote := upstream.RemoteAddr().String()
		logrus.Infof("Client<ip:%s> connected on [::]:%d.", remote, rule.LocalPort)

		go cm.handleConnection(upstream, rule)
	}
}

func (cm *ConnectionManager) handleConnection(upstream net.Conn, rule ForwardingRule) {
	defer cm.RemoveConnection(upstream)

	remote := upstream.RemoteAddr().String()

	// 设置连接超时
	upstream.SetDeadline(time.Now().Add(cm.timeout))

	// 连接到远程服务器
	var remoteAddr string
	if strings.Contains(rule.RemoteHost, ":") {
		remoteAddr = fmt.Sprintf("[%s]:%d", rule.RemoteHost, rule.RemotePort)
	} else {
		remoteAddr = fmt.Sprintf("%s:%d", rule.RemoteHost, rule.RemotePort)
	}
	downstream, err := net.DialTimeout("tcp", remoteAddr, cm.timeout)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to connect to %s for client<ip:%s>.",
			remoteAddr, remote)
		return
	}
	defer downstream.Close()

	// 设置下游连接超时
	downstream.SetDeadline(time.Now().Add(cm.timeout))

	// 添加到连接管理器，只有在连接成功后才添加
	cm.AddConnection(downstream)
	defer cm.RemoveConnection(downstream)

	logrus.Infof("Forwarding traffic from 0.0.0.0:%d to %s:%d for client<ip:%s>.",
		rule.LocalPort, rule.RemoteHost, rule.RemotePort, remote)

	// 创建上下文用于控制传输
	ctx, cancel := context.WithCancel(cm.ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer cancel()
		cm.transfer(ctx, downstream, upstream)
	}()
	go func() {
		defer wg.Done()
		defer cancel()
		cm.transfer(ctx, upstream, downstream)
	}()

	// 等待传输完成或上下文取消
	select {
	case <-ctx.Done():
		// 上下文被取消，等待goroutine完成
		wg.Wait()
	case <-time.After(cm.timeout):
		// 超时保护，避免goroutine泄漏
		cancel()
		wg.Wait()
	}
}

func (cm *ConnectionManager) transfer(ctx context.Context, dst io.Writer, src io.Reader) {
	// 尝试使用零拷贝传输
	pipe, err := NewPipe()
	if err == nil {
		defer pipe.Close()

		// 设置超时
		if rconn, ok := src.(net.Conn); ok {
			rconn.SetReadDeadline(time.Now().Add(cm.timeout))
		}
		if wconn, ok := dst.(net.Conn); ok {
			wconn.SetWriteDeadline(time.Now().Add(cm.timeout))
		}

		// 使用Transfer进行零拷贝传输
		done := make(chan struct{})
		var transferErr error
		go func() {
			defer close(done)
			_, transferErr = Transfer(dst, src)
		}()

		select {
		case <-ctx.Done():
			return
		case <-done:
			if transferErr != nil && transferErr != io.EOF {
				logrus.WithError(transferErr).Debug("Transfer error during zero-copy transfer")
			}
			return
		}
	}

	// 如果零拷贝不可用，回退到标准传输
	buffer := make([]byte, 32*1024) // 32KB buffer
	var rconn, wconn net.Conn
	var ok bool
	var n int
	var writeErr error
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if rconn, ok = src.(net.Conn); ok {
				rconn.SetReadDeadline(time.Now().Add(cm.timeout))
			}

			n, err = src.Read(buffer)
			if n > 0 {
				if wconn, ok = dst.(net.Conn); ok {
					wconn.SetWriteDeadline(time.Now().Add(cm.timeout))
				}

				_, writeErr = dst.Write(buffer[:n])
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
	app := &cli.App{
		Name:  "traffic-forwarder",
		Usage: "A high-performance TCP traffic forwarding tool implemented in Go, designed for efficient network traffic routing between local and remote endpoints.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output",
			},
			&cli.IntFlag{
				Name:  "timeout",
				Value: 30,
				Usage: "Connection timeout",
			},
			&cli.IntFlag{
				Name:  "maxconns",
				Value: 1000,
				Usage: "Maximum concurrent connections per port",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Int("timeout") <= 0 {
				return fmt.Errorf("timeout must be greater than 0")
			}
			if ctx.Int("timeout") > 600 {
				return fmt.Errorf("timeout must be less than 600")
			}
			timeout := time.Duration(ctx.Int("timeout")) * time.Second
			if ctx.Int("maxconns") <= 0 {
				return fmt.Errorf("maxconns must be greater than 0")
			}
			if ctx.Int("maxconns") > 10000 {
				return fmt.Errorf("maxconns must be less than 600")
			}
			maxconns := ctx.Int("maxconns")

			cm := NewConnectionManager(timeout, maxconns)
			defer cm.CloseAll()

			if cm.Run() {
				logrus.Info("Running traffic-forwarder...")
				signalCh := make(chan os.Signal, 1)
				signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
				<-signalCh
				logrus.Warning("To stop traffic-forwarder...")

				done := make(chan struct{})
				go func() {
					cm.Wait()
					close(done)
				}()

				select {
				case <-done:
					logrus.Info("All connections closed gracefully.")
				case <-time.After(10 * time.Second):
					logrus.Warning("Timeout waiting for connections to close, forcing shutdown.")
				}
				logrus.Warning("Stopped traffic-forwarder.")
			} else {
				logrus.Error("Failed to traffic-forwarder.")
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("Failed to run traffic-forwarder.")
	}
}
