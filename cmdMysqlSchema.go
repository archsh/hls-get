package main

import (
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Display MySQL Schema",
	Run: func(cmd *cobra.Command, args []string) {
		ShowMySQLSchema()
	},
}
