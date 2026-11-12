package geneqc


import (
	"log/slog"
	"fmt"
	"os"
	"encoding/json"
	"io"
	"errors"
	
	"github.com/gmc-norr/cleve"
	"github.com/spf13/cobra"
	"github.com/gmc-norr/cleve/mongo"
)


var updateCmd = &cobra.Command{


	Use:   "update [flags] run_id geneqc_file",
	Short: "Add gene QC data for a runID",
	Args: func(cmd *cobra.Command, args []string) error {
		
		if len(args) != 2 {
			return fmt.Errorf("RunID and GeneQc File is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}

		f, err := os.Open(args[1])
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

		var geneqc cleve.GeneQc
		err = json.Unmarshal([]byte(data), &geneqc)
		if err != nil {
			slog.Error("failed to Unmarshal JSON file", "error", err)
		}

		err = geneqc.ValidateUpdate()
		cobra.CheckErr(err)
		
		// Check if RunID exists
		filter := cleve.PanelQcFilter {
		RunId: args[0],
		}
		_, err = db.GenesQCs(filter)
		if err != nil {
			slog.Error("failed to extract data of RunId", args[0], "error", err)
			os.Exit(1)
		}

		// Check if data for the Gene already exists
		filter = cleve.PanelQcFilter {
		Symbol: geneqc.Symbol,
		RunId: args[0],
		}
		_, err = db.GenesQCs(filter)
		if err != mongo.ErrNoDocuments {
			slog.Error("Data for gene already exists", geneqc.Symbol)
			os.Exit(1)
		}
		
		if err := db.UpdateGeneQc(args[0], geneqc); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				cobra.CheckErr("the RunID does not exist")
			}
			slog.Error("failed to update run", "run", args[0], "error", err)
			os.Exit(1)
		}
},

}


