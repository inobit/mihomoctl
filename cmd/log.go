package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "查看 Mihomo 实时日志",
	Long:  `实时流式输出 Mihomo 日志，支持按级别过滤。`,
	Run: func(cmd *cobra.Command, args []string) {
		level, _ := cmd.Flags().GetString("level")
		validLevels := map[string]bool{"info": true, "warning": true, "error": true, "debug": true}
		if level != "" && !validLevels[level] {
			fmt.Fprintf(os.Stderr, "错误: 无效的日志级别 %s，可选: info, warning, error, debug\n", level)
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		c := newClient()
		c.HTTPClient.Timeout = 0 // 流式请求不设超时

		server := c.BaseURL
		fmt.Fprintf(os.Stderr, "正在连接 %s/logs?level=%s ...\n", server, level)

		if err := c.StreamLogs(ctx, level, os.Stdout); err != nil {
			if ctx.Err() != nil {
				return // 用户中断
			}
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	logCmd.Flags().StringP("level", "l", "info", "日志级别过滤 (info|warning|error|debug)")
	rootCmd.AddCommand(logCmd)
}
