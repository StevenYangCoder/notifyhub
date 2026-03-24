package notify

import "fmt"

// Hub 是通知渠道聚合根，管理全部渠道配置。
// 使用场景：应用服务通过 Hub 做渠道查询与合法性校验。
type Hub struct {
	channels map[string]ChannelConfig
}

// NewHub 创建渠道聚合根。
// 主要逻辑：
// 1. 校验每条渠道配置。
// 2. 校验渠道名称唯一性，防止发送路由冲突。
func NewHub(configs []ChannelConfig) (*Hub, error) {
	channels := make(map[string]ChannelConfig, len(configs))
	for _, cfg := range configs {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("渠道配置校验失败: %w", err)
		}
		if _, exists := channels[cfg.Name]; exists {
			return nil, fmt.Errorf("存在重复渠道名称: %s", cfg.Name)
		}
		channels[cfg.Name] = cfg
	}
	return &Hub{channels: channels}, nil
}

// GetChannel 按渠道名称查询配置。
// 主要逻辑：不存在时返回明确错误，便于业务快速定位问题。
// 使用场景：应用层按名称发送时调用。
func (h *Hub) GetChannel(name string) (ChannelConfig, error) {
	cfg, ok := h.channels[name]
	if !ok {
		return ChannelConfig{}, fmt.Errorf("未找到渠道: %s", name)
	}
	return cfg, nil
}

// ListChannels 返回全部渠道配置。
// 主要逻辑：将 map 拍平为切片，供广播等场景使用。
func (h *Hub) ListChannels() []ChannelConfig {
	result := make([]ChannelConfig, 0, len(h.channels))
	for _, cfg := range h.channels {
		result = append(result, cfg)
	}
	return result
}
