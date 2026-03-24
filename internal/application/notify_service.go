package application

import (
	"context"
	"fmt"
	"log/slog"

	"notifyhub/internal/domain/notify"
)

// NotifyService 是通知应用服务，负责业务编排。
// 使用场景：上层命令行、机器人或流水线通过该服务统一调用通知能力。
type NotifyService struct {
	hub     *notify.Hub
	factory notify.SenderFactory
	logger  *slog.Logger
}

// NewNotifyService 创建通知应用服务。
// 主要逻辑：注入聚合根、发送器工厂与日志对象，保证服务具备完整依赖。
func NewNotifyService(hub *notify.Hub, factory notify.SenderFactory, logger *slog.Logger) *NotifyService {
	return &NotifyService{
		hub:     hub,
		factory: factory,
		logger:  logger,
	}
}

// SendToChannel 向指定渠道发送消息。
// 主要逻辑：
// 1. 校验消息。
// 2. 获取渠道配置。
// 3. 根据渠道类型路由发送器并执行发送。
// 4. 在关键节点输出分级日志，便于追踪链路。
func (s *NotifyService) SendToChannel(ctx context.Context, channelName string, message notify.Message) error {
	if err := message.Validate(); err != nil {
		s.logger.Warn("消息校验失败", "渠道", channelName, "错误", err)
		return err
	}

	channelConfig, err := s.hub.GetChannel(channelName)
	if err != nil {
		s.logger.Error("查询渠道配置失败", "渠道", channelName, "错误", err)
		return err
	}

	sender, err := s.factory.Build(channelConfig.ChannelType)
	if err != nil {
		s.logger.Error("构建发送器失败", "渠道", channelName, "类型", channelConfig.ChannelType, "错误", err)
		return err
	}

	s.logger.Info("开始发送通知", "渠道", channelName, "类型", channelConfig.ChannelType)
	if err := sender.Send(ctx, message, channelConfig); err != nil {
		s.logger.Error("发送通知失败", "渠道", channelName, "类型", channelConfig.ChannelType, "错误", err)
		return err
	}
	s.logger.Info("发送通知成功", "渠道", channelName, "类型", channelConfig.ChannelType)
	return nil
}

// Broadcast 向全部渠道广播消息。
// 主要逻辑：遍历聚合内所有渠道，逐个发送并记录失败信息，最终聚合错误返回。
// 使用场景：统一告警、系统级通知。
func (s *NotifyService) Broadcast(ctx context.Context, message notify.Message) error {
	if err := message.Validate(); err != nil {
		s.logger.Warn("广播消息校验失败", "错误", err)
		return err
	}

	var failed int
	for _, channel := range s.hub.ListChannels() {
		if err := s.SendToChannel(ctx, channel.Name, message); err != nil {
			failed++
			s.logger.Warn("广播中单个渠道发送失败", "渠道", channel.Name, "错误", err)
		}
	}
	if failed > 0 {
		return fmt.Errorf("广播结束，共有%d个渠道发送失败", failed)
	}
	return nil
}
