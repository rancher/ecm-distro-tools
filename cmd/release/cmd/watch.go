package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rancher/ecm-distro-tools/cmd/release/watch"
	"github.com/spf13/cobra"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "watch release resources",
	Long:  `open a terminal UI to watch builds and pull requests`,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		listFile := filepath.Join(home, ".ecm-distro-tools", "watch_list.json")
		m, err := watch.New(*rootConfig.Auth, listFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			fmt.Println("failed to run program:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
