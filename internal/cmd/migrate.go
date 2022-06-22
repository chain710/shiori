package cmd

import (
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	var migrateDown bool
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the database to the latest version",
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.Migrate(migrateDown); err != nil {
				cError.Printf("Error during migration: %s", err)
			}
		},
	}

	cmd.Flags().BoolVar(&migrateDown, "down", false, "migrate all the way down (applying all down migrations)")
	return cmd
}
