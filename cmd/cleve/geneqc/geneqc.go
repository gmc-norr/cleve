package geneqc

import (
	"github.com/spf13/cobra"
)

func init() {
	GeneQcCmd.AddCommand(addCmd)
	GeneQcCmd.AddCommand(deleteCmd)
	GeneQcCmd.AddCommand(updateCmd)
}


var GeneQcCmd = &cobra.Command{
	Use:   "geneqc",
	Short: "Interact with gene QC data",
}
