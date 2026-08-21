# AGENTS.md

## 项目目标

`mihomoctl` — Mihomo (Clash Meta) 的 Go CLI + TUI 控制工具，通过 RESTful API 实现代理切换、测速、配置与连接管理。无参启动进入交互式选择（策略组 → 节点），支持 `cobra` 子命令直调。

## 源码结构

```
mihomoctl/
├── main.go                   # 入口：XDG config 加载 + viper + cmd.Execute()
├── cmd/                      # cobra 命令
│   ├── root.go               # 根命令（无参进 interact.Run）+ newClient() 工厂
│   ├── config.go             # config set/show
│   ├── mode.go               # mode / mode set
│   ├── status.go             # status
│   ├── connection.go         # connection list + switch <group> <node>
│   ├── log.go                # log (StreamLogs)
│   ├── dns.go                # dns flush
│   └── rule.go               # rule list
└── internal/
    ├── client/client.go      # HTTP 客户端（doRequest + 全部 API 封装）
    ├── model/model.go        # 数据结构（GroupInfo/ProxyInfo/DelayHistory 等）
    └── interact/
        ├── interact.go       # Run / runGroupSelect / runNodeSelect 流程 + 测速触发
        ├── selector.go       # SelectModel 通用选择器（bubbletea, t 测速, extra 渲染）
        └── connection.go     # connectionModel 连接列表 TUI
```

- `internal/client` 唯一 HTTP 封装，`cmd/mode.go:newClient()` 唯一工厂
- `internal/interact` 负责所有 bubbletea 交互，`selector.go:Update` 处理 `speedTestResultMsg`

## 常用命令

```bash
go build -o mihomoctl .       # 构建
go vet ./...                  # 检查（无测试套件，以此为准）

./mihomoctl                   # 交互式：选策略组 → 选节点（t 测速）
./mihomoctl status            # 状态概览
./mihomoctl mode [set <mode>] # 查看/切换模式 (rule/global/direct)
./mihomoctl switch <group> <node> # 直切节点
./mihomoctl connection list   # 连接列表
./mihomoctl log               # 实时日志
./mihomoctl rule list         # 规则列表
./mihomoctl config show/set   # 配置查看/写入
./mihomoctl dns flush         # 清 DNS 缓存
```
