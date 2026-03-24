package notify

import (
	"fmt"
	"strings"
)

// AtConfig 表示钉钉 @ 配置。
// 使用场景：当渠道类型为钉钉时，控制是否 @ 全员、以及 @ 哪些手机号。
type AtConfig struct {
	// IsAtAll 表示是否 @ 全员。
	IsAtAll bool `yaml:"isAtAll"`
	// AtMobiles 表示需要 @ 的手机号列表。
	AtMobiles []string `yaml:"atMobiles"`
}

// ChannelConfig 表示单个通知渠道配置。
// 使用场景：由配置文件加载后作为领域对象进入应用层进行发送编排。
type ChannelConfig struct {
	// Name 是渠道实例名，用于业务按名称路由发送。
	Name string `yaml:"name"`
	// ChannelType 是渠道类型，例如 dingtalk、chuckfang。
	ChannelType ChannelType `yaml:"channel_type"`
	// URL 是渠道请求地址。
	URL string `yaml:"url"`
	// AccessToken 是渠道访问令牌（主要用于钉钉）。
	AccessToken string `yaml:"access_token"`
	// Sign 是渠道加签密钥（主要用于钉钉）。
	Sign string `yaml:"sign"`
	// Keyword 是渠道关键字（主要用于钉钉关键字校验）。
	Keyword string `yaml:"keyword"`
	// At 表示 @ 相关配置（主要用于钉钉）。
	At AtConfig `yaml:"at"`
	// SMTPHost 是 SMTP 服务主机地址。
	SMTPHost string `yaml:"smtp_host"`
	// SMTPPort 是 SMTP 服务端口。
	SMTPPort int `yaml:"smtp_port"`
	// SMTPUsername 是 SMTP 登录用户名。
	SMTPUsername string `yaml:"smtp_username"`
	// SMTPPassword 是 SMTP 登录密码或授权码。
	SMTPPassword string `yaml:"smtp_password"`
	// SMTPTLSMode 是 SMTP TLS 模式，可选 auto/starttls/ssl/plain。
	SMTPTLSMode string `yaml:"smtp_tls_mode"`
	// SMTPFrom 是邮件发件人地址。
	SMTPFrom string `yaml:"smtp_from"`
	// SMTPTo 是主收件人列表。
	SMTPTo []string `yaml:"smtp_to"`
	// SMTPCc 是抄送收件人列表。
	SMTPCc []string `yaml:"smtp_cc"`
	// SMTPBcc 是密送收件人列表。
	SMTPBcc []string `yaml:"smtp_bcc"`
}

// Validate 对渠道配置做业务校验。
// 主要逻辑：
// 1. 校验名称、类型、URL 等通用必填字段。
// 2. 按渠道类型校验特有字段，保证请求发送前配置是完整的。
// 使用场景：配置加载后立即执行，尽早失败。
func (c ChannelConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("渠道名称不能为空")
	}
	if err := c.ChannelType.Validate(); err != nil {
		return err
	}

	switch c.ChannelType {
	case ChannelTypeChuckFang:
		if strings.TrimSpace(c.URL) == "" {
			return fmt.Errorf("渠道[%s]为ChuckFang时 URL 不能为空", c.Name)
		}
	case ChannelTypeDingTalk:
		if strings.TrimSpace(c.URL) == "" {
			return fmt.Errorf("渠道[%s]为钉钉时 URL 不能为空", c.Name)
		}
		if strings.TrimSpace(c.AccessToken) == "" {
			return fmt.Errorf("渠道[%s]为钉钉时 access_token 不能为空", c.Name)
		}
	case ChannelTypeSMTP:
		if strings.TrimSpace(c.SMTPHost) == "" {
			return fmt.Errorf("渠道[%s]为SMTP时 smtp_host 不能为空", c.Name)
		}
		if c.SMTPPort <= 0 {
			return fmt.Errorf("渠道[%s]为SMTP时 smtp_port 必须大于0", c.Name)
		}
		if strings.TrimSpace(c.SMTPFrom) == "" {
			return fmt.Errorf("渠道[%s]为SMTP时 smtp_from 不能为空", c.Name)
		}
		if len(c.AllSMTPRecipients()) == 0 {
			return fmt.Errorf("渠道[%s]为SMTP时至少要配置一个收件人（smtp_to/smtp_cc/smtp_bcc）", c.Name)
		}
		username := strings.TrimSpace(c.SMTPUsername)
		password := strings.TrimSpace(c.SMTPPassword)
		if (username == "" && password != "") || (username != "" && password == "") {
			return fmt.Errorf("渠道[%s]为SMTP时 smtp_username 和 smtp_password 必须同时配置或同时留空", c.Name)
		}
		tlsMode := strings.ToLower(strings.TrimSpace(c.SMTPTLSMode))
		if tlsMode != "" && tlsMode != "auto" && tlsMode != "starttls" && tlsMode != "ssl" && tlsMode != "plain" {
			return fmt.Errorf("渠道[%s]为SMTP时 smtp_tls_mode 仅支持 auto/starttls/ssl/plain", c.Name)
		}
	}
	return nil
}

// AllSMTPRecipients 返回 SMTP 全部收件人列表。
// 主要逻辑：合并 To/Cc/Bcc 并去除空字符串，供发送器作为 envelope recipients 使用。
// 使用场景：SMTP 发送前统一计算收件人列表。
func (c ChannelConfig) AllSMTPRecipients() []string {
	merged := make([]string, 0, len(c.SMTPTo)+len(c.SMTPCc)+len(c.SMTPBcc))
	merged = append(merged, c.SMTPTo...)
	merged = append(merged, c.SMTPCc...)
	merged = append(merged, c.SMTPBcc...)

	result := make([]string, 0, len(merged))
	for _, item := range merged {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
