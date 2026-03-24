# notifyhub

`notifyhub`全渠道通知工具类，目标支持所有主流渠道通知，方便在各种机器人、流水线、AI agent中使用。



# 一、项目要求

1. 使用 DDD 的模式进行开发，领域对象使用充血模型。
2. 所有文件格式均使用 UTF-8。
3. 每个文件、每个结构体、每个方法等均要详细备注，包括业务逻辑、方法内主要逻辑、以及使用场景等。
4. 所有备注、注释、异常提示、日志等均使用中文，在重要节点、关键节点多打印完善的日志。
5. 日志使用`log/slog`，区分`DEBUG`、`INFO`、`WARN`、`ERROR`
6. 要求设计需要支持多渠道，后续可能还会支持企业微信、飞书、钉钉等渠道
7. 期望配置文件`yaml`如下，按照配置文件进行开发

[点击查看配置文件](./configs/application.yaml)

# 二、命令行使用模板发送

1. 在配置文件 `application.notify.templates` 下定义模板，使用 `${变量名}` 作为占位符。
2. 命令行通过 `-template` 指定模板名，并通过可重复的 `-var key=value` 输入变量。
3. `channel_type: chuckfang` 会直接发送 GET 请求，发送前会把渠道 URL 中的 `${title}` 和 `${content}` 替换为最终消息（会进行 URL 编码）。
4. `channel_type: smtp` 会通过 SMTP 协议发送邮件，邮件主题使用消息标题，正文使用消息正文（或标题+正文组合），并支持 `smtp_tls_mode`：`auto/starttls/ssl/plain`。

示例：

```sh
go run ./cmd/notifyhub \
  -config configs/application.yaml \
  -channel dingtalk-robot \
  -template msg01 \
  -var env=prod \
  -var task=mysql-backup \
  -var status=成功 \
  -var duration=21s
```

# 三、编译跨平台二进制

## Bash
```bash
# Linux (amd64)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.BuildTime=$(date '+%Y-%m-%dT%H:%M:%S%z')" -o bin/notifyhub-linux-amd64 ./cmd/notifyhub

# macOS Intel (amd64)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.BuildTime=$(date '+%Y-%m-%dT%H:%M:%S%z')" -o bin/notifyhub-darwin-amd64 ./cmd/notifyhub

# macOS Apple Silicon (arm64)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-X main.BuildTime=$(date '+%Y-%m-%dT%H:%M:%S%z')" -o bin/notifyhub-darwin-arm64 ./cmd/notifyhub

# Windows (amd64)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.BuildTime=$(date '+%Y-%m-%dT%H:%M:%S%z')" -o bin/notifyhub-windows-amd64.exe ./cmd/notifyhub
```

## Windows 10 PowerShell
```powershell
# Linux (amd64)
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -ldflags "-X main.BuildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK')" -o bin/notifyhub-linux-amd64 ./cmd/notifyhub

# macOS Intel (amd64)
$env:GOOS="darwin"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -ldflags "-X main.BuildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK')" -o bin/notifyhub-darwin-amd64 ./cmd/notifyhub

# macOS Apple Silicon (arm64)
$env:GOOS="darwin"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"; go build -ldflags "-X main.BuildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK')" -o bin/notifyhub-darwin-arm64 ./cmd/notifyhub

# Windows (amd64)
$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -ldflags "-X main.BuildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK')" -o bin/notifyhub-windows-amd64.exe ./cmd/notifyhub
```

查看版本与构建时间：

```sh
./bin/notifyhub-darwin-amd64 -version
```
