package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS 操作",
}

var dnsFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "清除 DNS 缓存",
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		if err := c.FlushDNS(); err != nil {
			fmt.Fprintf(os.Stderr, "清除 DNS 缓存失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ DNS 缓存已清除")
	},
}

func init() {
	dnsCmd.AddCommand(dnsFlushCmd)
	rootCmd.AddCommand(dnsCmd)
}
