package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// JSONClient 是 HTTP JSON 客户端抽象。
// 使用场景：隔离具体 http.Client 细节，便于渠道发送器复用与测试替换。
type JSONClient interface {
	// PostJSON 发送 JSON POST 请求并返回响应体。
	PostJSON(ctx context.Context, url string, headers map[string]string, payload any) (*http.Response, []byte, error)
}

// Client 是 JSONClient 的默认实现。
// 使用场景：生产环境统一通过该对象发送 HTTP 请求并记录日志。
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient 创建基础 HTTP 客户端。
// 主要逻辑：设置合理超时时间，避免通知请求无限阻塞。
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// PostJSON 执行 JSON POST 请求。
// 主要逻辑：
// 1. 将 payload 序列化成 JSON。
// 2. 设置默认及自定义请求头。
// 3. 执行请求并读取响应，记录关键日志。
func (c *Client) PostJSON(ctx context.Context, url string, headers map[string]string, payload any) (*http.Response, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	c.logger.Debug("准备发送HTTP请求", "url", url, "请求体字节数", len(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("执行HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("读取HTTP响应失败: %w", err)
	}
	c.logger.Debug("HTTP请求完成", "url", url, "状态码", resp.StatusCode, "响应体字节数", len(respBody))
	return resp, respBody, nil
}
