package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/inobit/mihomoctl/cmd"
)

func main() {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
			os.Exit(1)
		}
		configDir = filepath.Join(home, ".config")
	}
	configDir = filepath.Join(configDir, "mihomoctl")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("MIHOMO")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "读取配置文件失败: %v\n", err)
			os.Exit(1)
		}
	}

	cmd.Execute()
}
