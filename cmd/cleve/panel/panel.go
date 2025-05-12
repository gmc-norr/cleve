package panel

import "github.com/spf13/cobra"

func init() {
	PanelCmd.AddCommand(addCmd)
	PanelCmd.AddCommand(archiveCmd)
	PanelCmd.AddCommand(listCmd)
}

var PanelCmd = &cobra.Command{
	Use:   "panel",
	Short: "Manage in-silico gene panels",
}
