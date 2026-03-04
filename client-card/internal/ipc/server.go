package ipc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"sync"
	"time"
)

// Handler 是 IPC 命令处理函数类型。
type Handler func(ctx context.Context, req *Frame) (interface{}, uint32)

// Server 是 IPC 服务端，监听 Named Pipe 或 Unix Socket。
type Server struct {
	pipePath string
	handlers map[CmdCode]Handler
	mu       sync.RWMutex
	listener net.Listener
	done     chan struct{}
}

// NewServer 创建 IPC 服务端。
func NewServer(pipePath string) *Server {
	return &Server{
		pipePath: pipePath,
		handlers: make(map[CmdCode]Handler),
		done:     make(chan struct{}),
	}
}

// Register 注册命令处理函数。
func (s *Server) Register(cmd CmdCode, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[cmd] = h
}

// Start 启动 IPC 服务端监听。
func (s *Server) Start() error {
	ln, err := listenPipe(s.pipePath)
	if err != nil {
		return fmt.Errorf("启动 IPC 监听失败 [%s]: %w", s.pipePath, err)
	}
	s.listener = ln
	slog.Info("IPC 服务已启动", "path", s.pipePath, "platform", runtime.GOOS)

	go s.acceptLoop()
	return nil
}

// Stop 停止 IPC 服务端。
func (s *Server) Stop() {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}
}

// acceptLoop 接受新连接。
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				slog.Error("IPC 接受连接失败", "error", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

// handleConn 处理单个连接的请求循环。
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	slog.Debug("IPC 新连接", "remote", conn.RemoteAddr())

	for {
		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		frame, err := ReadFrame(conn)
		if err != nil {
			// 连接关闭或超时，正常退出
			return
		}

		// 重置写入超时
		conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

		s.mu.RLock()
		h, ok := s.handlers[frame.Cmd]
		s.mu.RUnlock()

		if !ok {
			slog.Warn("未知 IPC 命令", "cmd", frame.Cmd)
			// 返回 CKR_FUNCTION_NOT_SUPPORTED = 0x54
			if err := WriteResponse(conn, frame.Cmd, 0x54, nil); err != nil {
				slog.Error("写入响应失败", "error", err)
				return
			}
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		data, rv := h(ctx, frame)
		cancel()

		if err := WriteResponse(conn, frame.Cmd, rv, data); err != nil {
			slog.Error("写入响应失败", "cmd", frame.Cmd, "error", err)
			return
		}
	}
}
