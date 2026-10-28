package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "重新加载 Mihomo 配置",
	Long: `重新加载 Mihomo 配置（PUT /configs），进程内热重载，不重启 mihomo:
  -f, --file <path>  指定配置文件（mihomo 所在主机上的绝对路径）；不指定则由 mihomo
                     使用启动时记录的配置文件
      --force        除代理/规则外还重建全部入站监听（含 allow-lan、bind-address、
                     各入站端口）；不指定则保留现有监听设置`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()

		path, _ := cmd.Flags().GetString("file")
		force, _ := cmd.Flags().GetBool("force")

		if err := c.ReloadConfig(path, force); err != nil {
			fmt.Fprintf(os.Stderr, "重载配置失败: %v\n", err)
			os.Exit(1)
		}

		if path != "" {
			fmt.Printf("✓ 已重载配置: %s\n", path)
		} else {
			fmt.Println("✓ 已重载默认配置文件")
		}
	},
}

func init() {
	reloadCmd.Flags().StringP("file", "f", "", "指定配置文件路径（mihomo 主机上的绝对路径）")
	reloadCmd.Flags().Bool("force", false, "同时重建全部入站监听")
	rootCmd.AddCommand(reloadCmd)
}
