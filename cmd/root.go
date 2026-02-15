package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "operator-directory-service",
	Short: "Operator/agent directory: list, filters by role, region, status",
	RunE:  runAPI,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(migrateCmd)
}
