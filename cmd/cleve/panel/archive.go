package panel

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [flags] PANEL",
	Short: "Archive or unarchive a gene panel",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		unarchive, _ := cmd.Flags().GetBool("unarchive")
		slog.Info("modifying panel", "id", args[0], "archive", !unarchive)
		db, err := mongo.Connect()
		cobra.CheckErr(err)
		if unarchive {
			err = db.UnarchivePanel(args[0])
		} else {
			err = db.ArchivePanel(args[0])
		}
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				cobra.CheckErr(fmt.Sprintf("panel with ID %q not found", args[0]))
			}
			cobra.CheckErr(err)
		}
	},
}

func init() {
	archiveCmd.Flags().BoolP("unarchive", "u", false, "unarchive a panel")
}
