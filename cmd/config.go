package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "配置 Mihomo 连接信息",
	Long:  `设置 Mihomo API 服务器地址和密钥，按 XDG 规范保存到 $XDG_CONFIG_HOME/mihomoctl/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		server, _ := cmd.Flags().GetString("server")
		secret, _ := cmd.Flags().GetString("secret")

		if server == "" {
			fmt.Fprintln(os.Stderr, "错误: --server 不能为空")
			os.Exit(1)
		}

		viper.Set("server", server)
		viper.Set("secret", secret)

		configPath := configFilePath()
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建配置目录失败: %v\n", err)
			os.Exit(1)
		}

		if err := viper.WriteConfigAs(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ 配置已保存到 %s\n", configPath)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "查看当前配置",
	Run: func(cmd *cobra.Command, args []string) {
		server := viper.GetString("server")
		secret := viper.GetString("secret")
		masked := ""
		if secret != "" {
			if len(secret) > 4 {
				masked = secret[:2] + "****" + secret[len(secret)-2:]
			} else {
				masked = "****"
			}
		}
		fmt.Printf("server: %s\n", server)
		fmt.Printf("secret: %s\n", masked)
		configPath := viper.ConfigFileUsed()
		if configPath != "" {
			fmt.Printf("config: %s\n", configPath)
		}
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 Mihomo 连接配置",
}

func init() {
	configSetCmd.Flags().StringP("server", "s", "", "API 地址 (如 127.0.0.1:9090)")
	configSetCmd.Flags().String("secret", "", "API 密钥")
	configSetCmd.MarkFlagRequired("server")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func configFilePath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "mihomoctl", "config.yaml")
}
