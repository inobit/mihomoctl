package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/inobit/mihomoctl/internal/interact"
)

// version 通过 ldflags 注入，格式: -X github.com/inobit/mihomoctl/cmd.version=1.0.0
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "mihomoctl",
	Short: "Mihomo CLI 控制工具",
	Long:  `mihomoctl 通过 Mihomo RESTful API 管理代理策略组和查看日志。`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		if err := interact.Run(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("server", "s", "", "Mihomo API 地址 (如 127.0.0.1:9090)")
	rootCmd.PersistentFlags().StringP("secret", "p", "", "Mihomo API 密钥（-p 不带值则交互式输入）")
	rootCmd.PersistentFlags().Lookup("secret").NoOptDefVal = "__prompt__"

	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("secret", rootCmd.PersistentFlags().Lookup("secret"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
