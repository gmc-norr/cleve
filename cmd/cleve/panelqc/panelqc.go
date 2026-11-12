package panelqc

import (
	"github.com/spf13/cobra"
)

func init() {
	PanelQcCmd.AddCommand(listCmd)
	PanelQcCmd.AddCommand(addCmd)
	PanelQcCmd.AddCommand(deleteCmd)
}


var PanelQcCmd = &cobra.Command{
	Use:   "panelqc",
	Short: "Interact with panel QC data",
}
