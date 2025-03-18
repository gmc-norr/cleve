package platform

import (
	"github.com/spf13/cobra"
)

var PlatformCmd = &cobra.Command{
	Use:   "platform [command]",
	Short: "Manage sequencing platforms",
}

func init() {
	PlatformCmd.AddCommand(listCmd)
}
