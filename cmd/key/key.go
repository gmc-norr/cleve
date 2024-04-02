package key

import (
	"github.com/spf13/cobra"
)

func init() {
	KeyCmd.AddCommand(listCmd)
	KeyCmd.AddCommand(createCmd)
	KeyCmd.AddCommand(deleteCmd)
}

var (
	KeyCmd = &cobra.Command{
		Use:   "key",
		Short: "API key management",
	}
)
