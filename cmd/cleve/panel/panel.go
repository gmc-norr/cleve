package panel

import "github.com/spf13/cobra"

func init() {
	PanelCmd.AddCommand(addCmd)
	PanelCmd.AddCommand(archiveCmd)
	PanelCmd.AddCommand(listCmd)
	PanelCmd.AddCommand(deleteCmd)
}

var PanelCmd = &cobra.Command{
	Use:   "panel",
	Short: "Manage in-silico gene panels",
}
