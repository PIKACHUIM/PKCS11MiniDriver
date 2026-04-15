//go:build windows

package ipc

import (
	"fmt"
	"log/slog"
	"net"
	"os/user"

	"github.com/Microsoft/go-winio"
)

// listenPipe 在 Windows 上创建 Named Pipe 监听。
// DACL 仅允许当前用户和 SYSTEM 账户访问，防止非授权进程连接。
func listenPipe(path string) (net.Listener, error) {
	sd := buildSecurityDescriptor()

	ln, err := winio.ListenPipe(path, &winio.PipeConfig{
		SecurityDescriptor: sd,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Named Pipe 失败 [%s]: %w", path, err)
	}
	slog.Info("Named Pipe 已创建（受 DACL 保护）", "path", path)
	return ln, nil
}

// buildSecurityDescriptor 构建 SDDL 安全描述符。
// D:P - DACL Protected（不继承父级权限）
// (A;;GA;;;SY) - 允许 SYSTEM 完全访问
// (A;;GA;;;BA) - 允许 Administrators 完全访问
// (A;;GA;;;<SID>) - 允许当前用户完全访问
func buildSecurityDescriptor() string {
	// 默认：仅 SYSTEM + Administrators
	base := "D:P(A;;GA;;;SY)(A;;GA;;;BA)"

	u, err := user.Current()
	if err != nil {
		slog.Warn("获取当前用户 SID 失败，使用默认 DACL", "error", err)
		return base
	}

	// 追加当前用户 SID 的完全访问权限
	return fmt.Sprintf("%s(A;;GA;;;%s)", base, u.Uid)
}
