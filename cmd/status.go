package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看运行状态概览",
	Long:  `一键查看 Mihomo 当前模式、流量速率、内存使用和活跃连接数。`,
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()

		cfg, err := c.GetConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取配置失败: %v\n", err)
			os.Exit(1)
		}

		traffic, err := c.GetTraffic()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取流量失败: %v\n", err)
			os.Exit(1)
		}

		mem, err := c.GetMemory()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取内存失败: %v\n", err)
			os.Exit(1)
		}

		conns, err := c.GetConnections()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取连接失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Mode: %s    LogLevel: %s\n", cfg.Mode, cfg.LogLevel)
		fmt.Printf("↑ %s  ↓ %s\n", formatBytes(float64(traffic.Up)), formatBytes(float64(traffic.Down)))
		fmt.Printf("Total ↑ %s  ↓ %s\n", formatBytes(float64(traffic.UpTotal)), formatBytes(float64(traffic.DownTotal)))
		fmt.Printf("Memory: %s    Active Connections: %d\n", formatBytes(float64(mem.Inuse)), len(conns.Connections))
	},
}

func formatBytes(b float64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%.0f B", b)
	}
	div, exp := float64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", b/div, "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
