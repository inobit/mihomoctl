# AGENTS.md

## 项目概述

mihomoctl 是一个 Go 编写的 Mihomo CLI 控制工具，通过 cobra 命令行框架 + bubbletea TUI 框架，调用 Mihomo 的 RESTful API 实现代理控制。

## 构建与检查

```bash
go build -o mihomoctl .          # 构建
go vet ./...                     # 代码检查
```

没有测试套件，以 `go build` 和 `go vet` 通过为准。

## 技术栈

| 组件 | 库 |
|---|---|
| CLI 框架 | `github.com/spf13/cobra` |
| 配置管理 | `github.com/spf13/viper` |
| TUI 交互 | `github.com/charmbracelet/bubbletea` |
| 终端样式 | `github.com/charmbracelet/lipgloss` |
| HTTP 客户端 | `net/http` 标准库 |

## 项目结构

```
mihomoctl/
├── main.go                     # 入口，viper 初始化
├── cmd/                        # cobra 命令定义
│   ├── root.go                 # 根命令（无参数时启动交互式）
│   ├── config.go               # config set/show
│   ├── mode.go                 # mode / mode set
│   ├── status.go               # status
│   ├── connection.go           # connection list
│   ├── log.go                  # log
│   ├── dns.go                  # dns flush
│   ├── rule.go                 # rule list
│   └── switch.go               # switch（定义在 connection.go）
├── internal/
│   ├── client/client.go        # HTTP API 客户端封装
│   ├── model/model.go          # API 数据结构
│   └── interact/
│       ├── selector.go         # 通用选择器 bubbletea 模型
│       ├── interact.go         # 策略组/节点交互流程 + ModeSelect
│       └── connection.go       # 连接列表 bubbletea 模型
├── go.mod
└── go.sum
```

## 代码约定

- 注释使用中文
- 日志消息使用英文
- cobra 命令在 `cmd/` 目录，交互式 TUI 在 `internal/interact/`
- HTTP 客户端统一在 `internal/client/`，数据结构在 `internal/model/`
- 创建 API 客户端统一使用 `cmd/mode.go` 中的 `newClient()` 函数
- bubbletea 模型统一使用指针接收者 `*SelectModel`、`*connectionModel`

## API 端点覆盖

| 端点 | 对应功能 |
|---|---|
| `GET /group` | 获取策略组列表 |
| `GET /proxies/:name` | 获取组内节点 |
| `PUT /proxies/:name` | 切换节点 |
| `GET /group/:name/delay` | 测速 |
| `GET/PATCH /configs` | 查看/修改配置 |
| `GET /traffic` | 流量信息 |
| `GET /memory` | 内存信息 |
| `GET/DELETE /connections` | 连接管理 |
| `DELETE /connections/:id` | 关闭连接 |
| `GET /logs` | 实时日志 |
| `GET /rules` | 规则列表 |
| `POST /cache/dns/flush` | 清除 DNS 缓存 |
