package channel

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"notifyhub/internal/domain/notify"
)

// SMTPSender 是 SMTP 邮件发送器。
// 使用场景：通过标准 SMTP 协议向邮箱发送通知，适合企业邮件告警场景。
type SMTPSender struct {
	logger *slog.Logger
}

// NewSMTPSender 创建 SMTP 发送器。
// 主要逻辑：注入日志对象，便于在关键节点记录请求状态和错误信息。
func NewSMTPSender(logger *slog.Logger) *SMTPSender {
	return &SMTPSender{
		logger: logger,
	}
}

// Send 执行 SMTP 发送。
// 主要逻辑：
// 1. 组装 SMTP 认证信息和全部收件人列表。
// 2. 构建标准邮件头与 UTF-8 正文。
// 3. 调用 smtp.SendMail 执行发送并输出关键日志。
func (s *SMTPSender) Send(ctx context.Context, message notify.Message, channel notify.ChannelConfig) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("SMTP发送前上下文已取消: %w", err)
	}

	recipients := channel.AllSMTPRecipients()
	if len(recipients) == 0 {
		return fmt.Errorf("SMTP收件人为空")
	}

	subject := strings.TrimSpace(message.Title)
	if subject == "" {
		subject = "notifyhub通知"
	}
	body := message.FullText()
	mailContent := buildSMTPMessage(channel, subject, body)

	addr := fmt.Sprintf("%s:%d", channel.SMTPHost, channel.SMTPPort)

	tlsMode := resolveSMTPTLSMode(channel.SMTPPort, channel.SMTPTLSMode)
	s.logger.Info("SMTP发送开始", "渠道", channel.Name, "smtp地址", addr, "收件人数", len(recipients), "tls模式", tlsMode)
	if err := s.sendWithSMTPClient(ctx, addr, tlsMode, channel, recipients, []byte(mailContent)); err != nil {
		return fmt.Errorf("SMTP发送失败: %w", err)
	}
	s.logger.Info("SMTP发送成功", "渠道", channel.Name, "smtp地址", addr, "收件人数", len(recipients), "tls模式", tlsMode)
	return nil
}

// buildSMTPMessage 组装完整 SMTP 邮件报文。
// 主要逻辑：构建 RFC822 头部，正文采用 UTF-8 + Base64 编码，提升中文兼容性。
func buildSMTPMessage(channel notify.ChannelConfig, subject string, body string) string {
	headers := make([]string, 0, 10)
	headers = append(headers, fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)))
	headers = append(headers, fmt.Sprintf("From: %s", channel.SMTPFrom))
	if len(channel.SMTPTo) > 0 {
		headers = append(headers, fmt.Sprintf("To: %s", strings.Join(channel.SMTPTo, ", ")))
	}
	if len(channel.SMTPCc) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(channel.SMTPCc, ", ")))
	}
	headers = append(headers, fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("utf-8", subject)))
	headers = append(headers, "MIME-Version: 1.0")
	headers = append(headers, "Content-Type: text/plain; charset=UTF-8")
	headers = append(headers, "Content-Transfer-Encoding: base64")

	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + encodedBody
}

// resolveSMTPTLSMode 解析 SMTP TLS 模式。
// 主要逻辑：优先使用显式配置；当配置为空或 auto 时，端口465默认 ssl，其它端口默认 starttls。
func resolveSMTPTLSMode(port int, configured string) string {
	mode := strings.ToLower(strings.TrimSpace(configured))
	if mode != "" && mode != "auto" {
		return mode
	}
	if port == 465 {
		return "ssl"
	}
	return "starttls"
}

// sendWithSMTPClient 使用 smtp.Client 执行分步骤发送。
// 主要逻辑：
// 1. 建立连接（普通TCP或SSL）。
// 2. 根据模式执行 STARTTLS。
// 3. 执行可选认证。
// 4. 完成 MAIL/RCPT/DATA 投递。
func (s *SMTPSender) sendWithSMTPClient(ctx context.Context, addr string, tlsMode string, channel notify.ChannelConfig, recipients []string, content []byte) error {
	client, err := s.openSMTPClient(ctx, addr, tlsMode, channel.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()
	defer client.Quit()

	if err := s.maybeAuth(client, channel); err != nil {
		return err
	}
	if err := client.Mail(channel.SMTPFrom); err != nil {
		return fmt.Errorf("SMTP设置发件人失败: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP设置收件人失败[%s]: %w", rcpt, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP进入DATA阶段失败: %w", err)
	}
	if _, err := writer.Write(content); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP写入邮件内容失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP提交邮件内容失败: %w", err)
	}
	return nil
}

// openSMTPClient 按 TLS 模式建立 SMTP 客户端连接。
// 主要逻辑：支持 ssl、starttls、plain 三种模式；starttls 在服务端不支持时返回明确错误。
func (s *SMTPSender) openSMTPClient(ctx context.Context, addr string, tlsMode string, host string) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	if tlsMode == "ssl" {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, fmt.Errorf("SMTP建立SSL连接失败: %w", err)
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("SMTP创建客户端失败(SSL): %w", err)
		}
		return client, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("SMTP建立TCP连接失败: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SMTP创建客户端失败: %w", err)
	}

	if tlsMode == "starttls" {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP服务端不支持STARTTLS，请改用 smtp_tls_mode=ssl 或 plain")
		}
		if err := client.StartTLS(&tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		}); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP执行STARTTLS失败: %w", err)
		}
	}
	return client, nil
}

// maybeAuth 根据配置决定是否进行 SMTP 认证。
// 主要逻辑：当用户名为空则跳过认证；否则使用 PLAIN 认证并返回认证阶段错误。
func (s *SMTPSender) maybeAuth(client *smtp.Client, channel notify.ChannelConfig) error {
	if strings.TrimSpace(channel.SMTPUsername) == "" {
		s.logger.Debug("SMTP未配置认证信息，跳过AUTH阶段")
		return nil
	}

	auth := smtp.PlainAuth("", channel.SMTPUsername, channel.SMTPPassword, channel.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败，请检查 smtp_username/smtp_password 或邮箱安全策略: %w", err)
	}
	return nil
}
