package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"notifyhub/internal/domain/notify"
)

// RootConfig 表示项目根配置对象。
// 使用场景：承接 README 约定的 YAML 结构，统一反序列化入口。
type RootConfig struct {
	// Application 是应用级配置。
	Application ApplicationConfig `yaml:"application"`
}

// ApplicationConfig 表示 application 节点配置。
// 使用场景：聚合应用内不同子模块配置，这里仅使用 notify 子模块。
type ApplicationConfig struct {
	// Notify 是通知模块配置。
	Notify NotifyConfig `yaml:"notify"`
}

// NotifyConfig 表示 notify 节点配置。
// 使用场景：维护全部通知渠道列表，供应用层编排使用。
type NotifyConfig struct {
	// Channels 是渠道配置列表。
	Channels []notify.ChannelConfig `yaml:"channels"`
	// Templates 是消息模板列表。
	Templates []notify.MessageTemplate `yaml:"templates"`
}

// LoadChannelsFromFile 从 YAML 文件加载渠道配置。
// 主要逻辑：
// 1. 读取文件。
// 2. 解析 YAML。
// 3. 返回渠道配置列表给领域层进一步校验。
func LoadChannelsFromFile(path string) ([]notify.ChannelConfig, error) {
	cfg, err := LoadNotifyConfigFromFile(path)
	if err != nil {
		return nil, err
	}
	return cfg.Channels, nil
}

// LoadNotifyConfigFromFile 从 YAML 文件加载通知模块完整配置。
// 主要逻辑：
// 1. 读取并解析 YAML。
// 2. 返回 notify 节点下的全部配置（渠道与模板）。
// 使用场景：命令行需要同时加载渠道和消息模板时使用。
func LoadNotifyConfigFromFile(path string) (*NotifyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg RootConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析YAML配置失败: %w", err)
	}
	return &cfg.Application.Notify, nil
}
