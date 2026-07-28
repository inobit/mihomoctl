package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/inobit/mihomoctl/internal/client"
	"github.com/inobit/mihomoctl/internal/interact"
)

var validModes = map[string]bool{"rule": true, "global": true, "direct": true}

var modeCmd = &cobra.Command{
	Use:   "mode",
	Short: "查看当前代理模式",
	Long:  `显示当前 Mihomo 代理模式 (rule / global / direct)。`,
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		cfg, err := c.GetConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取配置失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Mode: %s\n", cfg.Mode)
	},
}

var modeSetCmd = &cobra.Command{
	Use:   "set [rule|global|direct]",
	Short: "切换代理模式",
	Long: `切换 Mihomo 代理模式（运行时临时修改，不写配置文件）:
  rule    — 规则模式
  global  — 全局模式
  direct  — 直连模式`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()

		if len(args) == 0 {
			if err := interact.ModeSelect(c); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}

		mode := strings.ToLower(args[0])
		if !validModes[mode] {
			fmt.Fprintf(os.Stderr, "无效模式: %s (可选: rule, global, direct)\n", args[0])
			os.Exit(1)
		}

		if err := c.PatchConfig(map[string]any{"mode": mode}); err != nil {
			fmt.Fprintf(os.Stderr, "切换模式失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ 已切换到 %s 模式\n", mode)
	},
}

func init() {
	modeCmd.AddCommand(modeSetCmd)
	rootCmd.AddCommand(modeCmd)
}

func newClient() *client.Client {
	server := viper.GetString("server")
	secret := viper.GetString("secret")

	if server == "" {
		fmt.Fprintln(os.Stderr, "错误: 未配置 server，请先运行 mihomoctl config set")
		os.Exit(1)
	}

	if !strings.HasPrefix(server, "http") {
		server = "http://" + server
	}

	return client.New(server, secret)
}
