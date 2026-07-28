package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/inobit/mihomoctl/internal/interact"
)

var switchCmd = &cobra.Command{
	Use:   "switch <group> <node>",
	Short: "快捷切换代理节点",
	Long:  `非交互式切换指定策略组到指定节点。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		group := args[0]
		node := args[1]

		c := newClient()
		if err := c.SwitchProxy(group, node); err != nil {
			fmt.Fprintf(os.Stderr, "切换失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ 已切换 %s → %s\n", group, node)
	},
}

var connectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "管理活跃连接",
}

var connectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出活跃连接（交互式）",
	Long: `交互式连接管理页面:
  r        — 手动刷新
  a        — 切换自动刷新（默认 5s）
  d        — 关闭当前选中连接
  D        — 关闭全部连接
  ↑↓/jk    — 导航
  Esc/q    — 退出`,
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		if err := interact.ConnectionList(c); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	connectionCmd.AddCommand(connectionListCmd)
	rootCmd.AddCommand(connectionCmd)
	rootCmd.AddCommand(switchCmd)
}
