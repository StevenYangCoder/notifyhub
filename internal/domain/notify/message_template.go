package notify

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\$\{([a-zA-Z0-9_.-]+)\}`)

// MessageTemplate 表示通知消息模板。
// 使用场景：在 YAML 中维护可复用消息内容，由命令行输入变量后动态渲染。
type MessageTemplate struct {
	// Name 是模板名称，供命令行通过 -template 参数引用。
	Name string `yaml:"name"`
	// Title 是消息标题模板，支持 ${变量名} 占位符。
	Title string `yaml:"title"`
	// Content 是消息正文模板，支持 ${变量名} 占位符。
	Content string `yaml:"content"`
}

// Validate 校验模板定义合法性。
// 主要逻辑：要求模板名非空，且标题和正文至少存在一个。
// 使用场景：配置加载后进行防御性校验，尽早暴露问题。
func (t MessageTemplate) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	if strings.TrimSpace(t.Title) == "" && strings.TrimSpace(t.Content) == "" {
		return fmt.Errorf("模板[%s]标题和正文不能同时为空", t.Name)
	}
	return nil
}

// Render 使用输入变量渲染模板，输出最终消息对象。
// 主要逻辑：
// 1. 分别渲染标题和正文中的 ${变量}。
// 2. 若存在未传入变量，返回明确错误。
// 使用场景：命令行在发送前将模板转换为具体消息。
func (t MessageTemplate) Render(vars map[string]string) (Message, error) {
	title, err := renderTextWithVars(t.Title, vars, t.Name, "title")
	if err != nil {
		return Message{}, err
	}
	content, err := renderTextWithVars(t.Content, vars, t.Name, "content")
	if err != nil {
		return Message{}, err
	}
	return Message{
		Title:   title,
		Content: content,
	}, nil
}

// renderTextWithVars 渲染单段模板文本。
// 主要逻辑：
// 1. 扫描文本中的占位符键。
// 2. 校验每个占位符都有对应变量。
// 3. 完成文本替换并返回。
func renderTextWithVars(text string, vars map[string]string, templateName string, field string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}

	matches := placeholderRegex.FindAllStringSubmatch(text, -1)
	missing := make([]string, 0)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		key := m[1]
		if _, ok := vars[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("模板[%s]字段[%s]缺少占位符参数: %s", templateName, field, strings.Join(uniqueStrings(missing), ", "))
	}

	return placeholderRegex.ReplaceAllStringFunc(text, func(ph string) string {
		groups := placeholderRegex.FindStringSubmatch(ph)
		if len(groups) < 2 {
			return ph
		}
		return vars[groups[1]]
	}), nil
}

// uniqueStrings 对字符串切片去重。
// 使用场景：整理缺失占位符列表，避免重复输出影响可读性。
func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, item := range input {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
