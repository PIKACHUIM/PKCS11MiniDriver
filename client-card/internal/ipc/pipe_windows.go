//go:build windows

package ipc

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

// listenPipe 在 Windows 上创建 Named Pipe 监听。
func listenPipe(path string) (net.Listener, error) {
	ln, err := winio.ListenPipe(path, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;WD)", // 允许所有用户访问（本地进程通信）
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Named Pipe 失败 [%s]: %w", path, err)
	}
	return ln, nil
}
