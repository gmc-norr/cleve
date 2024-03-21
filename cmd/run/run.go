package run

import (
	"github.com/spf13/cobra"
)

func init() {
	RunCmd.AddCommand(addCmd)
	RunCmd.AddCommand(listCmd)
	RunCmd.AddCommand(updateCmd)
	RunCmd.AddCommand(deleteCmd)
}

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Interact with sequencing runs",
}
