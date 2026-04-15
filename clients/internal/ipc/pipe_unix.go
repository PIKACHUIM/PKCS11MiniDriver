//go:build !windows

package ipc

import (
	"fmt"
	"net"
	"os"
)

// listenPipe 在 macOS/Linux 上创建 Unix Domain Socket 监听。
func listenPipe(path string) (net.Listener, error) {
	// 清理旧的 socket 文件
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("清理旧 socket 文件失败: %w", err)
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("创建 Unix Socket 失败 [%s]: %w", path, err)
	}

	// 设置权限，只允许当前用户访问
	if err := os.Chmod(path, 0600); err != nil {
		ln.Close()
		return nil, fmt.Errorf("设置 socket 权限失败: %w", err)
	}

	return ln, nil
}
