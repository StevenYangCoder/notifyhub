package notify

import "context"

// Sender 定义单一渠道发送行为。
// 使用场景：基础设施层实现具体渠道，领域与应用层仅依赖抽象接口。
type Sender interface {
	// Send 按指定渠道配置发送消息。
	// 主要逻辑：实现方根据渠道协议将 Message 转为远程接口请求。
	Send(ctx context.Context, message Message, channel ChannelConfig) error
}

// SenderFactory 定义发送器工厂接口。
// 使用场景：应用层仅根据渠道类型获取对应实现，支持后续渠道横向扩展。
type SenderFactory interface {
	// Build 根据渠道类型创建发送器。
	// 主要逻辑：在工厂内部做渠道路由，不把实现细节暴露给应用层。
	Build(channelType ChannelType) (Sender, error)
}
