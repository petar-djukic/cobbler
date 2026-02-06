// Package main provides the cobbler CLI entry point.
// Implements: docs/ARCHITECTURE § System Components (CLI), docs/road-map.yaml release 01.0.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "cobbler",
	Short: "Cobbler cobbles together context for AI coding agents",
	Long: `Cobbler manages AI agent workflows using the crumbs cupboard for task storage.

Commands follow shoemaking terminology:
  measure   Assess project state and propose tasks
  stitch    Execute work via AI agents
  inspect   Evaluate output quality
  mend      Fix issues found by inspect
  pattern   Propose design changes`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print cobbler version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cobbler version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
