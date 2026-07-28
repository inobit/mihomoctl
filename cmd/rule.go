package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "规则管理",
}

var ruleListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有规则",
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		rules, err := c.GetRules()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取规则失败: %v\n", err)
			os.Exit(1)
		}

		if len(rules) == 0 {
			fmt.Println("无规则")
			return
		}

		for i, r := range rules {
			fmt.Printf("[%3d] %-12s  %-40s  →  %s\n", i, r.Type, truncateRule(r.Payload, 40), r.Proxy)
		}
	},
}

func truncateRule(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func init() {
	ruleCmd.AddCommand(ruleListCmd)
	rootCmd.AddCommand(ruleCmd)
}
