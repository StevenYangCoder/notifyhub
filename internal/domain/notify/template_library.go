package notify

import "fmt"

// TemplateLibrary 是消息模板仓储聚合对象。
// 使用场景：集中管理模板集合，提供按名称查询能力。
type TemplateLibrary struct {
	templates map[string]MessageTemplate
}

// NewTemplateLibrary 创建模板仓储。
// 主要逻辑：
// 1. 校验每个模板定义。
// 2. 校验模板名称唯一，防止命令行引用冲突。
func NewTemplateLibrary(templates []MessageTemplate) (*TemplateLibrary, error) {
	result := make(map[string]MessageTemplate, len(templates))
	for _, tpl := range templates {
		if err := tpl.Validate(); err != nil {
			return nil, fmt.Errorf("模板配置校验失败: %w", err)
		}
		if _, exists := result[tpl.Name]; exists {
			return nil, fmt.Errorf("存在重复模板名称: %s", tpl.Name)
		}
		result[tpl.Name] = tpl
	}
	return &TemplateLibrary{templates: result}, nil
}

// Get 按名称获取模板。
// 主要逻辑：当模板不存在时返回明确错误，便于命令行快速定位。
func (l *TemplateLibrary) Get(name string) (MessageTemplate, error) {
	tpl, ok := l.templates[name]
	if !ok {
		return MessageTemplate{}, fmt.Errorf("未找到模板: %s", name)
	}
	return tpl, nil
}
