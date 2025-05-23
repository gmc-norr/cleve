package panel

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	deleteId      string
	deleteVersion string
	deleteCmd     = &cobra.Command{
		Use:   "delete [flags] ID [VERSION]",
		Short: "Delete panel(s)",
		Long:  "Delete on or more panels. If VERSION is omitted, all versions with the specified ID will be deleted.",
		PreRun: func(cmd *cobra.Command, args []string) {
			r := bufio.NewReader(os.Stdin)
			deleteAll := deleteVersion == ""
			for {
				if deleteAll {
					fmt.Printf("this will delete all versions of %s, continue? [Y/n] ", deleteId)
				} else {
					fmt.Printf("this will delete version %s of %s, continue? [Y/n] ", deleteVersion, deleteId)
				}
				s, _ := r.ReadString('\n')
				s = strings.TrimSpace(s)
				s = strings.ToLower(s)
				if s == "n" {
					fmt.Fprintln(os.Stderr, "canceled")
					os.Exit(0)
					return
				} else if s == "y" {
					break
				} else {
					fmt.Fprintln(os.Stderr, "answer with Y or N")
					continue
				}
			}
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}
			deleteId = args[0]
			if err := cobra.MaximumNArgs(2)(cmd, args); err != nil {
				return err
			}
			if len(args) > 1 {
				deleteVersion = args[1]
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			cobra.CheckErr(err)
			n, err := db.DeletePanel(deleteId, deleteVersion)
			cobra.CheckErr(err)
			if n == 0 && deleteVersion == "" {
				fmt.Printf("no entries found for %s\n", deleteId)
				os.Exit(1)
			} else if n == 0 {
				fmt.Printf("version %s not found for %s\n", deleteVersion, deleteId)
				os.Exit(1)
			} else if n == 1 {
				fmt.Printf("deleted %d entry of %s\n", n, deleteId)
			} else {
				fmt.Printf("deleted %d entries of %s\n", n, deleteId)
			}
		},
	}
)
