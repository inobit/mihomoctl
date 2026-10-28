package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/inobit/mihomoctl/internal/client"
	"github.com/inobit/mihomoctl/internal/interact"
	"github.com/inobit/mihomoctl/internal/model"
	"github.com/spf13/cobra"
)

// providerUpdateTimeout provider 拉取是同步操作，订阅大/服务器慢时 15s 不够
const providerUpdateTimeout = 120 * time.Second

// providerUpdateHint 拉取完成后的生效提示
const providerUpdateHint = "提示: 如未生效，执行 mihomoctl reload 重新加载配置"

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "远程订阅管理",
	Long: `管理 proxy-providers（远程代理节点订阅）与 rule-providers（远程规则集）。

子命令默认作用于 proxy providers，加 --rule 切换到 rule providers:
  provider list                 # 列出 proxy providers
  provider list --rule          # 列出 rule providers
  provider update <name>        # 强制拉取 proxy provider
  provider update <name> --rule # 强制拉取 rule provider

只处理真正从外部拉取的 provider（vehicle 为 HTTP/File）；mihomo 由策略组或
内联节点派生的 Compatible/Inline provider 无法远程更新，不列出也不更新。`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 proxy providers（--rule 列出 rule providers）",
	Run: func(cmd *cobra.Command, args []string) {
		kind, err := providerKindFromFlags(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		c := newClient()
		providers, err := getProviders(c, kind)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取 providers 失败: %v\n", err)
			os.Exit(1)
		}

		if len(providers) == 0 {
			fmt.Println("无 provider")
			return
		}

		if kind == providerRule {
			fmt.Printf("%-28s %-10s %-10s %-6s %s\n", "NAME", "VEHICLE", "BEHAVIOR", "RULES", "UPDATED")
			for _, name := range sortedKeys(providers) {
				p := providers[name]
				fmt.Printf("%-28s %-10s %-10s %-6d %s\n", p.Name, p.VehicleType, p.Behavior, p.RuleCount, formatUpdatedAt(p.UpdatedAt))
			}
			return
		}

		fmt.Printf("%-28s %-10s %-6s %s\n", "NAME", "VEHICLE", "NODES", "UPDATED")
		for _, name := range sortedKeys(providers) {
			p := providers[name]
			fmt.Printf("%-28s %-10s %-6d %s\n", p.Name, p.VehicleType, len(p.Proxies), formatUpdatedAt(p.UpdatedAt))
		}
	},
}

var providerUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "强制重新拉取 provider（--rule 操作 rule providers）",
	Long: `强制重新拉取 provider，绕过配置里的 interval 定时器。

  update               拉取全部 proxy providers（等同 --all）
  update --rule        拉取全部 rule providers
  update <name>        拉取指定 proxy provider
  update <name> --rule 拉取指定 rule provider`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kind, err := providerKindFromFlags(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if allFlag, _ := cmd.Flags().GetBool("all"); allFlag && len(args) > 0 {
			fmt.Fprintln(os.Stderr, "错误: --all 与指定名称不能同时使用")
			os.Exit(1)
		}
		// 不指定名字就是全量，--all 只是显式写法
		all := len(args) == 0

		c := newClient()
		providers, err := getProviders(c, kind)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取 providers 失败: %v\n", err)
			os.Exit(1)
		}

		c.SetTimeout(providerUpdateTimeout)

		if all {
			jobs := make([]interact.Task, 0, len(providers))
			for _, name := range sortedKeys(providers) {
				jobs = append(jobs, newProviderUpdateTask(c, kind, name))
			}
			if interact.RunTasks(jobs, interact.ProgressOptions{Hint: providerUpdateHint}).Failed() > 0 {
				os.Exit(1)
			}
			return
		}

		name := args[0]
		if _, ok := providers[name]; !ok {
			fmt.Fprintf(os.Stderr, "未找到%s: %s\n", kind.label(), name)
			fmt.Fprintf(os.Stderr, "可选: %v\n", sortedKeys(providers))
			os.Exit(1)
		}
		if interact.RunTasks([]interact.Task{newProviderUpdateTask(c, kind, name)}, interact.ProgressOptions{Hint: providerUpdateHint}).Failed() > 0 {
			os.Exit(1)
		}
	},
}

type providerKind int

const (
	providerProxy providerKind = iota
	providerRule
)

func (k providerKind) label() string {
	if k == providerRule {
		return "rule provider"
	}
	return "proxy provider"
}

// providerKindFromFlags 默认 proxy provider，--rule 切到 rule provider
func providerKindFromFlags(cmd *cobra.Command) (providerKind, error) {
	proxy, _ := cmd.Flags().GetBool("proxy")
	rule, _ := cmd.Flags().GetBool("rule")
	if proxy && rule {
		return providerProxy, errors.New("错误: --proxy 与 --rule 不能同时使用")
	}
	if rule {
		return providerRule, nil
	}
	return providerProxy, nil
}

func getProviders(c *client.Client, kind providerKind) (map[string]model.ProviderInfo, error) {
	if kind == providerRule {
		return c.GetRuleProviders()
	}
	providers, err := c.GetProxyProviders()
	if err != nil {
		return nil, err
	}
	return filterUpdatableProviders(providers), nil
}

// filterUpdatableProviders 丢掉 Compatible/Inline —— 它们由策略组、内联节点派生，没有远程可拉取的内容
func filterUpdatableProviders(providers map[string]model.ProviderInfo) map[string]model.ProviderInfo {
	out := make(map[string]model.ProviderInfo, len(providers))
	for name, p := range providers {
		if p.VehicleType == "HTTP" || p.VehicleType == "File" {
			out[name] = p
		}
	}
	return out
}

func updateProvider(c *client.Client, kind providerKind, name string) error {
	if kind == providerRule {
		return c.UpdateRuleProvider(name)
	}
	return c.UpdateProxyProvider(name)
}

// newProviderUpdateTask 把一次拉取包成进度任务，标签即界面上的进度行标题
func newProviderUpdateTask(c *client.Client, kind providerKind, name string) interact.Task {
	return interact.Task{
		Label: kind.label() + ": " + name,
		Run:   func() error { return updateProvider(c, kind, name) },
	}
}

func sortedKeys(m map[string]model.ProviderInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatUpdatedAt(s string) string {
	if s == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	if t.IsZero() || t.Year() == 1 { // inline/compatible provider 返回零值时间
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func addProviderKindFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("proxy", false, "操作 proxy providers（默认）")
	cmd.Flags().Bool("rule", false, "操作 rule providers")
}

func init() {
	addProviderKindFlags(providerListCmd)
	providerUpdateCmd.Flags().Bool("all", false, "拉取该类型下全部 providers（不指定名称时默认行为）")
	addProviderKindFlags(providerUpdateCmd)
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerUpdateCmd)
	rootCmd.AddCommand(providerCmd)
}
