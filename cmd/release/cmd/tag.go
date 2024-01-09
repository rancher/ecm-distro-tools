package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// tagCmd represents the tag command
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			rootCmd.Help()
			os.Exit(0)
		}

		switch args[0] {
		case "k3s":
		case "rke2":
		default:
			rootCmd.Help()
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tagCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tagCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
