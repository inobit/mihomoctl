package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"go.yaml.in/yaml/v3"
)

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "配置 Mihomo 连接信息",
	Long:  `设置 Mihomo API 服务器地址和密钥，按 XDG 规范保存到 $XDG_CONFIG_HOME/mihomoctl/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		server, _ := cmd.Flags().GetString("server")
		secret, _ := cmd.Flags().GetString("secret")
		serverChanged := cmd.Flags().Changed("server")
		secretChanged := cmd.Flags().Changed("secret")

		if !serverChanged && !secretChanged {
			fmt.Fprintln(os.Stderr, "错误: 请至少指定 --server 或 --secret")
			os.Exit(1)
		}

		configPath := configFilePath()

		// 读取已有配置文件
		var cfg map[string]any
		if data, err := os.ReadFile(configPath); err == nil {
			yaml.Unmarshal(data, &cfg)
		}
		if cfg == nil {
			cfg = make(map[string]any)
		}

		// 应用修改
		if serverChanged {
			if server == "" {
				fmt.Fprintln(os.Stderr, "错误: --server 不能为空")
				os.Exit(1)
			}
			cfg["server"] = server
		}

		if secretChanged {
			if secret == "__prompt__" {
				fmt.Print("输入 API 密钥: ")
				password, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					fmt.Fprintf(os.Stderr, "\n读取密钥失败: %v\n", err)
					os.Exit(1)
				}
				fmt.Println()
				secret = string(password)
			}
			cfg["secret"] = secret
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化配置失败: %v\n", err)
			os.Exit(1)
		}

		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建配置目录失败: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
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
