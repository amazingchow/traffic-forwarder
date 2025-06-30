package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// BenchmarkConnectionManager 测试连接管理器的性能
func BenchmarkConnectionManager(b *testing.B) {
	cm := NewConnectionManager(1000)
	defer cm.CloseAll()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 模拟连接
			conn := &mockConn{}
			if cm.AddConnection(conn) {
				cm.RemoveConnection(conn)
			}
		}
	})
}

// BenchmarkTransfer 测试数据传输性能
func BenchmarkTransfer(b *testing.B) {
	// 创建测试连接
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

			// 启动双向传输
			go TransferWithContext(ctx, server, client)
			go TransferWithContext(ctx, client, server)

			// 发送一些测试数据
			testData := []byte("test data")
			client.Write(testData)

			// 读取数据以完成传输
			buffer := make([]byte, 1024)
			server.Read(buffer)

			// 等待传输完成
			<-ctx.Done()
			cancel()
		}
	})
}

// BenchmarkMemoryUsage 测试内存使用情况
func BenchmarkMemoryUsage(b *testing.B) {
	cm := NewConnectionManager(100)
	defer cm.CloseAll()

	var wg sync.WaitGroup
	conns := make([]net.Conn, 0, 100)

	b.ResetTimer()

	// 创建连接
	for i := 0; i < 100; i++ {
		conn := &mockConn{}
		if cm.AddConnection(conn) {
			conns = append(conns, conn)
		}
	}

	// 模拟数据传输
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
		}()
	}

	wg.Wait()

	// 清理连接
	for _, conn := range conns {
		cm.RemoveConnection(conn)
	}
}

// TestConnectionManager 测试连接管理器功能
func TestConnectionManager(t *testing.T) {
	cm := NewConnectionManager(3)
	defer cm.CloseAll()

	// 测试添加连接
	conn1 := &mockConn{}
	if !cm.AddConnection(conn1) {
		t.Error("Failed to add first connection")
	}

	conn2 := &mockConn{}
	if !cm.AddConnection(conn2) {
		t.Error("Failed to add second connection")
	}

	conn3 := &mockConn{}
	if !cm.AddConnection(conn3) {
		t.Error("Failed to add third connection")
	}

	// 测试连接数限制
	conn4 := &mockConn{}
	if cm.AddConnection(conn4) {
		t.Error("Should not add fourth connection")
	}

	// 测试移除连接
	cm.RemoveConnection(conn1)
	if !cm.AddConnection(conn4) {
		t.Error("Failed to add fourth connection")
	}
}

// TestTransferWithContext 测试带上下文的传输
func TestTransferWithContext(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 启动双向传输
	go TransferWithContext(ctx, server, client)
	go TransferWithContext(ctx, client, server)

	// 发送测试数据
	testData := []byte("hello world")
	_, err := client.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// 读取数据
	buffer := make([]byte, 1024)
	n, err := server.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	if string(buffer[:n]) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(buffer[:n]))
	}

	// 等待传输完成
	<-ctx.Done()
}

// mockConn 模拟连接用于测试
type mockConn struct {
	closed bool
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, fmt.Errorf("connection closed")
	}
	return 0, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, fmt.Errorf("connection closed")
	}
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &mockAddr{}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &mockAddr{}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type mockAddr struct{}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return "127.0.0.1:8080"
}

// BenchmarkMemoryAllocation 测试内存分配
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 测试缓冲区分配
		buffer := make([]byte, 32*1024)
		_ = buffer

		// 测试连接管理器创建
		cm := NewConnectionManager(100)
		cm.CloseAll()
	}
}

// TestMemoryLeak 测试内存泄漏
func TestMemoryLeak(t *testing.T) {
	// 运行多次创建和销毁连接管理器
	for i := 0; i < 1000; i++ {
		cm := NewConnectionManager(10)

		// 添加一些连接
		for j := 0; j < 5; j++ {
			conn := &mockConn{}
			cm.AddConnection(conn)
		}

		// 清理
		cm.CloseAll()
		cm.Wait()
	}
}
