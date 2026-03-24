package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"notifyhub/internal/application"
	"notifyhub/internal/domain/notify"
	infrachannel "notifyhub/internal/infrastructure/channel"
	"notifyhub/internal/infrastructure/config"
	"notifyhub/internal/infrastructure/httpx"
)

var (
	// Version 是程序版本号，采用硬编码维护。
	// 使用场景：启动日志与排障时快速确认当前二进制版本。
	Version = "v0.1.0"
	// BuildTime 是构建时间，默认 unknown，构建时通过 -ldflags 注入。
	// 使用场景：定位构建批次，排查部署版本不一致问题。
	BuildTime = "unknown"
)

// main 是命令行入口。
// 使用场景：在机器人、流水线、脚本中快速发起单渠道或全渠道通知。
func main() {
	normalizeBuildInfo()

	cfgPath := flag.String("config", "configs/application.yaml", "配置文件路径")
	channelName := flag.String("channel", "", "目标渠道名称，不传且开启-broadcast时表示全渠道广播")
	templateName := flag.String("template", "", "消息模板名称，模板在YAML中配置")
	title := flag.String("title", "", "通知标题")
	content := flag.String("content", "", "通知正文")
	markdown := flag.Bool("markdown", false, "是否按Markdown发送")
	broadcast := flag.Bool("broadcast", false, "是否广播到所有渠道")
	debug := flag.Bool("debug", false, "是否开启DEBUG日志")
	versionOnly := flag.Bool("version", false, "输出版本与构建时间")
	var templateVars templateVarMap
	flag.Var(&templateVars, "var", "模板变量，格式 key=value，可重复输入")
	flag.Parse()

	if *versionOnly {
		fmt.Printf("notifyhub version=%s build_time=%s\n", Version, BuildTime)
		return
	}

	logger := newLogger(*debug)
	logger.Info("notifyhub启动", "配置文件", *cfgPath, "版本", Version, "构建时间", BuildTime)

	notifyCfg, err := config.LoadNotifyConfigFromFile(*cfgPath)
	if err != nil {
		logger.Error("加载配置失败", "错误", err)
		exitWithError(err)
	}

	hub, err := notify.NewHub(notifyCfg.Channels)
	if err != nil {
		logger.Error("构建渠道聚合失败", "错误", err)
		exitWithError(err)
	}
	templateLibrary, err := notify.NewTemplateLibrary(notifyCfg.Templates)
	if err != nil {
		logger.Error("构建模板仓储失败", "错误", err)
		exitWithError(err)
	}

	client := httpx.NewClient(logger)
	factory := infrachannel.NewFactory(client, logger)
	service := application.NewNotifyService(hub, factory, logger)

	msg := notify.Message{
		Title:    *title,
		Content:  *content,
		Markdown: *markdown,
	}
	if strings.TrimSpace(*templateName) != "" {
		msg, err = buildMessageByTemplate(templateLibrary, *templateName, templateVars, *markdown)
		if err != nil {
			logger.Error("解析模板失败", "模板", *templateName, "错误", err)
			exitWithError(err)
		}
		logger.Info(
			"模板解析完成",
			"模板", *templateName,
			"变量数量", len(templateVars),
			"标题", msg.Title,
			"正文", msg.Content,
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if *broadcast {
		if err := service.Broadcast(ctx, msg); err != nil {
			logger.Error("广播发送失败", "错误", err)
			exitWithError(err)
		}
		logger.Info("广播发送完成")
		return
	}

	if *channelName == "" {
		err := fmt.Errorf("未开启广播时，必须指定-channel参数")
		logger.Error("参数校验失败", "错误", err)
		exitWithError(err)
	}

	if err := service.SendToChannel(ctx, *channelName, msg); err != nil {
		logger.Error("单渠道发送失败", "渠道", *channelName, "错误", err)
		exitWithError(err)
	}
	logger.Info("单渠道发送完成", "渠道", *channelName)
}

// normalizeBuildInfo 规范化构建元数据。
// 主要逻辑：当链接注入为空字符串时，回退到默认值，避免日志出现空值。
func normalizeBuildInfo() {
	if strings.TrimSpace(Version) == "" {
		Version = "dev"
	}
	if strings.TrimSpace(BuildTime) == "" {
		BuildTime = "unknown"
	}
}

// templateVarMap 表示命令行传入的模板变量集合。
// 使用场景：通过重复 -var 参数传入占位符值，如 -var env=prod -var task=backup。
type templateVarMap map[string]string

// String 返回变量集合字符串。
// 使用场景：满足 flag.Value 接口要求。
func (m *templateVarMap) String() string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("%v", map[string]string(*m))
}

// Set 解析单条 key=value 变量。
// 主要逻辑：校验格式、提取键值并放入 map，重复键后者覆盖前者。
func (m *templateVarMap) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("模板变量不能为空")
	}
	items := strings.SplitN(value, "=", 2)
	if len(items) != 2 {
		return fmt.Errorf("模板变量格式错误，必须是 key=value: %s", value)
	}
	key := strings.TrimSpace(items[0])
	if key == "" {
		return fmt.Errorf("模板变量key不能为空: %s", value)
	}
	if *m == nil {
		*m = make(map[string]string)
	}
	(*m)[key] = items[1]
	return nil
}

// buildMessageByTemplate 按模板名和变量构建消息对象。
// 主要逻辑：查找模板并执行渲染，同时透传 markdown 选项。
func buildMessageByTemplate(library *notify.TemplateLibrary, name string, vars templateVarMap, markdown bool) (notify.Message, error) {
	tpl, err := library.Get(name)
	if err != nil {
		return notify.Message{}, err
	}
	msg, err := tpl.Render(map[string]string(vars))
	if err != nil {
		return notify.Message{}, err
	}
	msg.Markdown = markdown
	return msg, nil
}

// newLogger 创建结构化日志对象。
// 主要逻辑：基于参数选择 INFO 或 DEBUG 级别，统一输出到标准输出。
func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

// exitWithError 统一错误退出。
// 主要逻辑：向标准错误输出中文错误信息，并使用非0状态码退出进程。
func exitWithError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "执行失败: %v\n", err)
	os.Exit(1)
}
