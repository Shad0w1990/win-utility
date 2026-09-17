package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "sysguard", Short: "Win11 Overlay Optimizer",
	Run: func(cmd *cobra.Command, args []string) { createAndShowGUI() },
}

func main() {
	if len(os.Args) <= 1 {
		createAndShowGUI()
		return
	}
	rootCmd.Execute()
}
