package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"notifyhub/internal/domain/notify"
	"notifyhub/internal/infrastructure/httpx"
)

// DingTalkSender 是钉钉机器人发送器。
// 使用场景：基于钉钉机器人协议构建请求体、处理签名并发送。
type DingTalkSender struct {
	client httpx.JSONClient
	logger *slog.Logger
}

// NewDingTalkSender 创建钉钉发送器。
// 主要逻辑：注入通用 HTTP 客户端和日志，供发送方法复用。
func NewDingTalkSender(client httpx.JSONClient, logger *slog.Logger) *DingTalkSender {
	return &DingTalkSender{
		client: client,
		logger: logger,
	}
}

// Send 执行钉钉通知发送。
// 主要逻辑：
// 1. 计算完整请求 URL（含 access_token、可选签名）。
// 2. 按消息类型组装 payload。
// 3. 发送请求并校验响应状态码。
func (s *DingTalkSender) Send(ctx context.Context, message notify.Message, channel notify.ChannelConfig) error {
	finalURL, err := s.buildRequestURL(channel)
	if err != nil {
		s.logger.Error("钉钉请求URL构建失败", "渠道", channel.Name, "错误", err)
		return err
	}

	payload := s.buildPayload(message, channel)
	s.logger.Info("钉钉发送请求", "渠道", channel.Name, "url", finalURL)
	resp, body, err := s.client.PostJSON(ctx, finalURL, nil, payload)
	if err != nil {
		s.logger.Error("钉钉请求失败", "渠道", channel.Name, "url", finalURL, "错误", err)
		return fmt.Errorf("钉钉请求失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		s.logger.Error(
			"钉钉返回非成功状态码",
			"渠道", channel.Name,
			"url", finalURL,
			"状态码", resp.StatusCode,
			"响应", string(body),
		)
		return fmt.Errorf("钉钉返回非成功状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	if err := validateDingTalkResponse(body); err != nil {
		s.logger.Error("钉钉业务返回失败", "渠道", channel.Name, "url", finalURL, "响应", string(body), "错误", err)
		return err
	}

	s.logger.Info("钉钉发送成功", "渠道", channel.Name, "url", finalURL, "状态码", resp.StatusCode)
	return nil
}

// buildRequestURL 构建钉钉请求地址。
// 主要逻辑：在原始 URL 上追加 access_token，若存在 sign 则按钉钉规则追加 timestamp 和 sign。
func (s *DingTalkSender) buildRequestURL(channel notify.ChannelConfig) (string, error) {
	parsedURL, err := url.Parse(channel.URL)
	if err != nil {
		return "", fmt.Errorf("解析钉钉URL失败: %w", err)
	}

	query := parsedURL.Query()
	query.Set("access_token", channel.AccessToken)

	sign := strings.TrimSpace(channel.Sign)
	if sign != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		signature, err := signDingTalk(timestamp, sign)
		if err != nil {
			return "", err
		}
		query.Set("timestamp", timestamp)
		query.Set("sign", signature)
	}

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

// buildPayload 构建钉钉消息请求体。
// 主要逻辑：
// 1. 若 Markdown 模式则使用 markdown 消息类型。
// 2. 否则使用 text 消息类型。
// 3. 自动拼接关键字并注入 @ 配置。
func (s *DingTalkSender) buildPayload(message notify.Message, channel notify.ChannelConfig) map[string]any {
	fullText := strings.TrimSpace(message.FullText())
	keyword := strings.TrimSpace(channel.Keyword)
	if keyword != "" && !strings.Contains(fullText, keyword) {
		fullText = keyword + " " + fullText
	}

	at := map[string]any{
		"isAtAll":   channel.At.IsAtAll,
		"atMobiles": channel.At.AtMobiles,
	}

	if message.Markdown {
		return map[string]any{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": message.Title,
				"text":  fullText,
			},
			"at": at,
		}
	}

	return map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": fullText,
		},
		"at": at,
	}
}

// signDingTalk 计算钉钉签名。
// 主要逻辑：按钉钉协议使用 HMAC-SHA256 计算签名并做 URL 编码。
func signDingTalk(timestamp string, secret string) (string, error) {
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(stringToSign)); err != nil {
		return "", fmt.Errorf("生成钉钉签名失败: %w", err)
	}
	rawSign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return url.QueryEscape(rawSign), nil
}

func validateDingTalkResponse(body []byte) error {
	var resp struct {
		ErrCode *int64  `json:"errcode"`
		ErrMsg  string  `json:"errmsg"`
		Code    *int64  `json:"code"`
		Msg     string  `json:"msg"`
		Success *bool   `json:"success"`
		OK      *bool   `json:"ok"`
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}

	if resp.ErrCode != nil && *resp.ErrCode != 0 {
		return fmt.Errorf("钉钉业务失败: errcode=%d errmsg=%s", *resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	if resp.Code != nil && *resp.Code != 0 && *resp.Code != 200 {
		msg := strings.TrimSpace(resp.Msg)
		if msg == "" && resp.Message != nil {
			msg = strings.TrimSpace(*resp.Message)
		}
		return fmt.Errorf("钉钉业务失败: code=%d msg=%s", *resp.Code, msg)
	}
	if resp.Success != nil && !*resp.Success {
		msg := strings.TrimSpace(resp.Msg)
		if msg == "" && resp.Message != nil {
			msg = strings.TrimSpace(*resp.Message)
		}
		return fmt.Errorf("钉钉业务失败: success=false msg=%s", msg)
	}
	if resp.OK != nil && !*resp.OK {
		msg := strings.TrimSpace(resp.Msg)
		if msg == "" && resp.Message != nil {
			msg = strings.TrimSpace(*resp.Message)
		}
		return fmt.Errorf("钉钉业务失败: ok=false msg=%s", msg)
	}

	return nil
}
