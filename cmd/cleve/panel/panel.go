package panel

import "github.com/spf13/cobra"

func init() {
	PanelCmd.AddCommand(addCmd)
}

var PanelCmd = &cobra.Command{
	Use:   "panel",
	Short: "Manage in-silico gene panels",
}
