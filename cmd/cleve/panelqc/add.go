package panelqc


import (
	"log/slog"
	"fmt"
	"os"
	"encoding/json"
	"io"

	"github.com/gmc-norr/cleve"
	"github.com/spf13/cobra"
	"github.com/gmc-norr/cleve/mongo"
)

var addCmd = &cobra.Command{


	Use:   "add [flags] panelqc file",
	Short: "Add panel QC data for a runID",
	Args: func(cmd *cobra.Command, args []string) error {
		
		if len(args) != 1 {
			return fmt.Errorf("PanelQc File is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}

		f, err := os.Open(args[0])
		cobra.CheckErr(err)

		defer func() {
			if err := f.Close(); err != nil {
				slog.Error("failed to close file", "error", err)
			}
		}()

		data, err := io.ReadAll(f)
		if err != nil {
			slog.Error("failed to read the file", "error", err)
		}
		
		var panelqc cleve.PanelQc
		err = json.Unmarshal([]byte(data), &panelqc)
		if err != nil {
			slog.Error("failed to Unmarshal JSON file", "error", err)
		}

		err = panelqc.Validate()
		cobra.CheckErr(err)

		err = db.CreatePanelQc(panelqc)
		if mongo.IsDuplicateKeyError(err) {
			cobra.CheckErr("a panel with this id and version already exists")
		}
		cobra.CheckErr(err)
},

}



