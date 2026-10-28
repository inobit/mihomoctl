package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/inobit/mihomoctl/internal/model"
	"github.com/spf13/cobra"
)

var tunCmd = &cobra.Command{
	Use:   "tun [on|off|status]",
	Short: "查看或开关 TUN 模式",
	Long: `查看或开关 Mihomo TUN 模式（运行时临时修改，不写配置文件）:
  tun         — 查看 TUN 状态
  tun status  — 同上
  tun on      — 开启 TUN
  tun off     — 关闭 TUN

只提交 tun.enable，其余 tun 配置项沿用当前值。`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()

		if len(args) == 0 || strings.EqualFold(args[0], "status") {
			cfg, err := c.GetConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "获取配置失败: %v\n", err)
				os.Exit(1)
			}
			printTunStatus(cfg.Tun)
			return
		}

		var enable bool
		switch strings.ToLower(args[0]) {
		case "on":
			enable = true
		case "off":
			enable = false
		default:
			fmt.Fprintf(os.Stderr, "无效参数: %s (可选: on, off, status)\n", args[0])
			os.Exit(1)
		}

		if err := c.PatchConfig(map[string]any{"tun": map[string]any{"enable": enable}}); err != nil {
			fmt.Fprintf(os.Stderr, "切换 TUN 失败: %v\n", err)
			os.Exit(1)
		}

		if enable {
			fmt.Println("✓ TUN ON")
		} else {
			fmt.Println("✓ TUN OFF")
		}
	},
}

func printTunStatus(tun *model.TunConfig) {
	if tun == nil {
		fmt.Println("TUN: 未知（控制器未返回 tun 配置）")
		return
	}

	state := "OFF"
	if tun.Enable {
		state = "ON"
	}
	fmt.Printf("TUN: %s\n", state)
	fmt.Printf("Stack: %s\n", tun.Stack)
	fmt.Printf("Device: %s\n", tun.Device)
	fmt.Printf("AutoRoute: %t\n", tun.AutoRoute)
	fmt.Printf("AutoDetectInterface: %t\n", tun.AutoDetectInterface)
}

func init() {
	rootCmd.AddCommand(tunCmd)
}
