# mihomoctl

mihomoctl 是一个 Mihomo（Clash.Meta）的 CLI 控制工具，通过 RESTful API（`external-controller`）管理代理、查看日志和运行状态。

## 安装

### Go

```bash
go install github.com/inobit/mihomoctl@latest
```

### Scoop (Windows)

```bash
scoop bucket add inosoft https://github.com/inobit/inosoft
scoop install inosoft/mihomoctl
```

或从源码构建：

```bash
git clone https://github.com/inobit/mihomoctl.git
cd mihomoctl
go build -o mihomoctl .
```

## 配置

```bash
mihomoctl config set --server 127.0.0.1:9090 --secret your-secret
```

配置文件遵循 XDG 规范，保存在 `$XDG_CONFIG_HOME/mihomoctl/config.yaml`（默认 `~/.config/mihomoctl/config.yaml`）。

也可通过环境变量覆盖：`MIHOMO_SERVER` / `MIHOMO_SECRET`。

## 命令

### 交互式切换节点

```bash
mihomoctl
```

列出策略组 → 选组 → 列出节点和延迟 → 选节点 → 切换。
- `↑↓/jk` 导航
- 数字键直接跳转
- `Tab` 切换过滤/全部
- `t` 测速
- `Esc` 回退 / 退出

### 代理模式

```bash
mihomoctl mode                     # 查看当前模式
mihomoctl mode set                 # 交互式选择
mihomoctl mode set global          # 直接切换（rule/global/direct）
```

### 连接管理

```bash
mihomoctl connection list          # 交互式连接页面
```

- `r` 手动刷新
- `a` 开关自动刷新（默认 5s）
- `d` 关闭当前选中
- `D` 关闭全部

### 快捷切换

```bash
mihomoctl switch "🚀 自动选择" "🇯🇵 日本"
```

### 运行状态

```bash
mihomoctl status
```

### 日志

```bash
mihomoctl log                   # info 级别
mihomoctl log -l debug          # debug 级别
mihomoctl log -l error          # 仅错误
```

### 规则

```bash
mihomoctl rule list
```

### DNS

```bash
mihomoctl dns flush             # 清除 DNS 缓存
```

### 配置管理

```bash
mihomoctl config set -s 127.0.0.1:9090 --secret xxx
mihomoctl config show
```

## 注意事项

- 所有运行时修改（`mode set`、`switch`）均为临时修改，不会写入 mihomo 的配置文件，重启后恢复
- 测速默认使用 `https://www.gstatic.com/generate_204`，超时 5000ms
