package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stitchType string

var stitchCmd = &cobra.Command{
	Use:   "stitch",
	Short: "Execute work via AI agents",
	Long: `Stitch picks a ready crumb from the cupboard, claims it, builds a prompt,
runs an AI agent, validates the output, and closes the crumb.

Task types:
  --type docs   Execute documentation tasks (write or update markdown)
  --type code   Execute code tasks (git worktree, implement, merge)`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stitch: not implemented (type=%s)\n", stitchType)
	},
}

func init() {
	stitchCmd.Flags().StringVar(&stitchType, "type", "docs", "Task type: docs or code")
	rootCmd.AddCommand(stitchCmd)
}
