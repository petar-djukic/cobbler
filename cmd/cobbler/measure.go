package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var measureCmd = &cobra.Command{
	Use:   "measure",
	Short: "Assess project state and propose tasks",
	Long: `Measure reads project state (VISION, ARCHITECTURE, ROADMAP, cupboard)
and invokes an AI agent to analyze the state and propose new work items.

Output is a set of proposed crumbs that the user reviews before import.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("measure: not implemented")
	},
}

func init() {
	rootCmd.AddCommand(measureCmd)
}
