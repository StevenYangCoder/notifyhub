package notify

import "fmt"

// ChannelType 表示通知渠道类型。
// 使用场景：用于在工厂中路由不同渠道实现，避免上层出现大量 if-else。
type ChannelType string

const (
	// ChannelTypeChuckFang 表示 ChuckFang 渠道。
	ChannelTypeChuckFang ChannelType = "chuckfang"
	// ChannelTypeDingTalk 表示钉钉机器人渠道。
	ChannelTypeDingTalk ChannelType = "dingtalk"
	// ChannelTypeSMTP 表示邮件 SMTP 渠道。
	ChannelTypeSMTP ChannelType = "smtp"
)

// Validate 校验渠道类型是否合法。
// 主要逻辑：仅允许项目已实现的渠道类型，未知类型直接返回错误。
// 使用场景：配置加载后进行防御性校验，避免运行时才发现配置错误。
func (c ChannelType) Validate() error {
	switch c {
	case ChannelTypeChuckFang, ChannelTypeDingTalk, ChannelTypeSMTP:
		return nil
	default:
		return fmt.Errorf("不支持的渠道类型: %s", c)
	}
}
