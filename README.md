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

### 订阅 (providers)

```bash
mihomoctl provider list                  # 列出 proxy-providers（默认）
mihomoctl provider list --rule           # 列出 rule-providers
mihomoctl provider update                # 强制拉取全部 proxy providers（等同 --all）
mihomoctl provider update <name>         # 强制拉取指定 proxy provider
mihomoctl provider update <name> --rule  # 强制拉取指定 rule provider
```

子命令默认作用于 proxy providers，`--rule` 切到 rule providers，`--proxy` 仅用于显式声明（与 `--rule` 同时给出会报错）。
只处理真正从外部拉取的 provider（vehicle 为 `HTTP`/`File`）；mihomo 由策略组、内联节点派生的 `Compatible`/`Inline` provider（含 `default`）不列出也不更新。
`update` 走 `PUT /providers/{proxies|rules}/{name}`，绕过配置里的 `interval` 定时器，同步执行（超时放宽到 120s）。
多个 provider 会并发拉取（上限 4），每项一行 spinner + 已耗时，完成变 `✓` 并给出总耗时；输出被重定向时降级为逐行打印。

### 重载配置

```bash
mihomoctl reload                        # 重载 mihomo 启动时的配置文件
mihomoctl reload -f /etc/mihomo/config.yaml # 重载指定配置文件（绝对路径）
mihomoctl reload --force                # 同时重建全部入站监听
```

进程内热重载，不重启 mihomo；但会重建代理/规则/Provider 与 TUN，已有连接会断。

### TUN

```bash
mihomoctl tun                   # 查看 TUN 状态
mihomoctl tun on                # 开启 TUN
mihomoctl tun off               # 关闭 TUN
```

### 配置管理

```bash
mihomoctl config set -s 127.0.0.1:9090 --secret xxx
mihomoctl config show
```

## 注意事项

- 所有运行时修改（`mode set`、`switch`、`tun on/off`）均为临时修改，不会写入 mihomo 的配置文件，重启后恢复
- `reload` 会读取配置文件并应用（进程内热重载），会重建代理/规则/TUN，已有连接会断
- 测速默认使用 `https://www.gstatic.com/generate_204`，超时 5000ms
