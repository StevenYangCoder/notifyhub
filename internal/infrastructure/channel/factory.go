package channel

import (
	"fmt"
	"log/slog"

	"notifyhub/internal/domain/notify"
	"notifyhub/internal/infrastructure/httpx"
)

// Factory 是渠道发送器工厂。
// 使用场景：应用层通过工厂获取发送器，不直接依赖具体渠道实现。
type Factory struct {
	senders map[notify.ChannelType]notify.Sender
	logger  *slog.Logger
}

// NewFactory 创建发送器工厂。
// 主要逻辑：在初始化阶段注册已实现渠道，后续新增渠道仅需在这里扩展映射。
func NewFactory(client httpx.JSONClient, logger *slog.Logger) *Factory {
	return &Factory{
		senders: map[notify.ChannelType]notify.Sender{
			notify.ChannelTypeChuckFang: NewChuckFangSender(logger),
			notify.ChannelTypeDingTalk:  NewDingTalkSender(client, logger),
			notify.ChannelTypeSMTP:      NewSMTPSender(logger),
		},
		logger: logger,
	}
}

// Build 按渠道类型返回发送器实例。
// 主要逻辑：若渠道未注册，返回明确错误并输出告警日志。
func (f *Factory) Build(channelType notify.ChannelType) (notify.Sender, error) {
	sender, ok := f.senders[channelType]
	if !ok {
		f.logger.Warn("发送器工厂未找到渠道实现", "渠道类型", channelType)
		return nil, fmt.Errorf("渠道类型[%s]未实现发送器", channelType)
	}
	return sender, nil
}
