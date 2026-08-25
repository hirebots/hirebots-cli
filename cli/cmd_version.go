package main

// cmd_version.go — `hirebots version` command and --version flag.
//
// The version variable is set at build time via ldflags:
//
//	go build -ldflags "-X main.version=0.2.0" -o hirebots ./cli/hirebots

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the CLI version. Overridden at build time via ldflags.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

// versionFlag is bound to the root --version flag.
var versionFlag bool

func init() {
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Print version and exit.")
	rootCmd.AddCommand(versionCmd)
}