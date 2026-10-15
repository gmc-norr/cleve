package geneqc

import (
	"github.com/spf13/cobra"
)

func init() {
	GeneQcCmd.AddCommand(addCmd)
	GeneQcCmd.AddCommand(deleteCmd)
}


var GeneQcCmd = &cobra.Command{
	Use:   "geneqc",
	Short: "Interact with gene QC data",
}
