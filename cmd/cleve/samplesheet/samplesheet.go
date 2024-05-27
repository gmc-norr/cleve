package samplesheet

import (
	"github.com/spf13/cobra"
)

func init() {
	SampleSheetCmd.AddCommand(addCmd)
}

var SampleSheetCmd = &cobra.Command{
	Use:   "samplesheet",
	Short: "Interact with sample sheets",
}
