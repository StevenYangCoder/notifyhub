package notify

import (
	"fmt"
	"strings"
)

// Message 表示通知消息领域对象（充血模型）。
// 使用场景：业务层创建消息后，将校验和格式化逻辑封装在对象内部，避免贫血模型。
type Message struct {
	// Title 为通知标题，用于辅助阅读和聚合告警信息。
	Title string
	// Content 为通知正文，允许较长文本。
	Content string
	// Markdown 为 true 时，发送器可按 Markdown 模式发送。
	Markdown bool
}

// Validate 校验消息对象完整性。
// 主要逻辑：要求标题和正文至少有一个非空，避免无意义空消息。
// 使用场景：应用服务在发送前统一调用。
func (m Message) Validate() error {
	if strings.TrimSpace(m.Title) == "" && strings.TrimSpace(m.Content) == "" {
		return fmt.Errorf("通知标题和正文不能同时为空")
	}
	return nil
}

// FullText 返回聚合后的完整文本。
// 主要逻辑：将标题与正文按可读格式合并，便于不同渠道复用。
// 使用场景：对只支持纯文本的渠道可直接使用该文本发送。
func (m Message) FullText() string {
	title := strings.TrimSpace(m.Title)
	content := strings.TrimSpace(m.Content)
	switch {
	case title == "":
		return content
	case content == "":
		return title
	default:
		return title + "\n" + content
	}
}
