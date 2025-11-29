package commands

import (
	"context"
	"log"
	"time"

	"github.com/spf13/cobra"
)

// newPruneLogsCmd creates the "prune-logs" command to delete old log entries.
func newPruneLogsCmd(state *cliState) *cobra.Command {
	var daysToKeep int

	cmd := &cobra.Command{
		Use:   "prune-logs",
		Short: "Delete system logs older than X days",
		Run: func(cmd *cobra.Command, args []string) {
			if daysToKeep < 0 {
				log.Fatal("Days cannot be negative")
			}

			log.Printf("Pruning logs older than %d days...", daysToKeep)

			duration := time.Duration(daysToKeep) * 24 * time.Hour

			count, err := state.DB.PruneLogs(context.Background(), duration)
			if err != nil {
				log.Fatalf("Failed to prune logs: %v", err)
			}

			log.Printf("Success: Deleted %d old log entries.", count)
		},
	}

	cmd.Flags().IntVar(&daysToKeep, "days", 30, "Number of days of logs to keep")

	return cmd
}
