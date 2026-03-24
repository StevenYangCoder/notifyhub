package channel

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"notifyhub/internal/domain/notify"
)

// ChuckFangSender 是 ChuckFang 渠道发送器。
// 使用场景：按 RESTful 风格拼接 URL 后，直接发起 GET 请求。
type ChuckFangSender struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewChuckFangSender 创建 ChuckFang 发送器。
// 主要逻辑：内部维护独立 GET 客户端，避免与 JSON POST 客户端耦合。
func NewChuckFangSender(logger *slog.Logger) *ChuckFangSender {
	return &ChuckFangSender{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// Send 执行 ChuckFang 通知发送。
// 主要逻辑：
// 1. 在 URL 中替换 ${title} 与 ${content} 占位符。
// 2. 直接使用 GET 请求调用 ChuckFang 接口。
// 3. 根据状态码判断是否成功。
func (s *ChuckFangSender) Send(ctx context.Context, message notify.Message, channel notify.ChannelConfig) error {
	finalURL := strings.ReplaceAll(channel.URL, "${title}", url.PathEscape(message.Title))
	finalURL = strings.ReplaceAll(finalURL, "${content}", url.PathEscape(message.Content))

	s.logger.Debug("ChuckFang发送器开始GET请求", "渠道", channel.Name, "url", finalURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalURL, nil)
	if err != nil {
		return fmt.Errorf("创建ChuckFang请求失败: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("执行ChuckFang请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取ChuckFang响应失败: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ChuckFang返回非成功状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	s.logger.Info("ChuckFang发送成功", "渠道", channel.Name, "状态码", resp.StatusCode)
	return nil
}
