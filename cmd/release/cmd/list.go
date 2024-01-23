package cmd

import (
	"errors"
	"fmt"

	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/spf13/cobra"
)

var rancherImageTag *string

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		//
	},
}

var rancherListSubCmd = &cobra.Command{
	Use:   "rancher",
	Short: "List Rancher resources",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("arguments required")
		}
		switch args[0] {
		case "nonmirrored-rc-images":
			if *rancherImageTag == "" {
				return errors.New("invalid list rancher ...command")
			}

			imagesRC, err := rancher.ListRancherImagesRC(*rancherImageTag)
			if err != nil {
				return err
			}

			fmt.Println(imagesRC)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(rancherListSubCmd)

	rancherImageTag = listCmd.Flags().StringP("tag", "t", "", "release tag to validate images")
}
