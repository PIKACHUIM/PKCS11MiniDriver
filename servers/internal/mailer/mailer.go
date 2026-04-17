// Package mailer 提供 SMTP 邮件发送能力。
// 当 SMTP 未启用时，Send 方法直接把内容写入日志，不发邮件（便于本地/测试环境）。
package mailer

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/globaltrusts/server-card/configs"
)

// Mailer 是邮件发送器接口。
type Mailer interface {
	// Send 向 to 发送主题为 subject、正文为 body（text/plain）的邮件。
	Send(to, subject, body string) error
}

// New 根据配置创建 Mailer 实例。
// - 若 cfg.Enabled=false，返回 nopMailer（仅写日志）；
// - 否则返回 smtpMailer。
func New(cfg configs.SMTPConfig) Mailer {
	if !cfg.Enabled || cfg.Host == "" {
		return &nopMailer{}
	}
	return &smtpMailer{cfg: cfg}
}

// nopMailer 不真实发送，仅 info 级别日志。
type nopMailer struct{}

func (m *nopMailer) Send(to, subject, body string) error {
	slog.Info("mailer: SMTP 未启用，仅记录日志",
		"to", to, "subject", subject, "body", body)
	return nil
}

// smtpMailer 使用 net/smtp 发送（支持 STARTTLS 基础账户）。
type smtpMailer struct {
	cfg configs.SMTPConfig
}

func (m *smtpMailer) Send(to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("收件人为空")
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	from := m.cfg.From
	if from == "" {
		from = m.cfg.Username
	}
	fromHeader := from
	if m.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", m.cfg.FromName, from)
	}

	msg := buildMessage(fromHeader, to, subject, body)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if err := smtp.SendMail(addr, auth, from, []string{to}, msg); err != nil {
		return fmt.Errorf("SMTP 发送失败: %w", err)
	}
	return nil
}

// buildMessage 拼接最小可用的邮件消息（UTF-8 + 纯文本）。
func buildMessage(from, to, subject, body string) []byte {
	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n",
		from, to, subject,
	)
	return []byte(headers + body)
}
