package db

import (
	"github.com/spf13/cobra"
)

func init() {
	DbCmd.AddCommand(indexCmd)
}

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management",
}
