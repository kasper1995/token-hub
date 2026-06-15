package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/model"
)

func runBackfillConversationLogSummariesCommand(args []string) int {
	flags := flag.NewFlagSet("backfill-conversation-log-summaries", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	afterId := flags.Int("after-id", 0, "only scan conversation logs with id greater than this value")
	limit := flags.Int("limit", 1000, "maximum source logs to scan")
	startTimestamp := flags.Int64("start-timestamp", 0, "only scan source logs created at or after this unix timestamp")
	endTimestamp := flags.Int64("end-timestamp", 0, "only scan source logs created at or before this unix timestamp")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if err := initConversationCleanCommandResources(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize resources:", err)
		return 1
	}
	defer func() {
		if err := model.CloseDB(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to close database:", err)
		}
	}()

	result, err := model.BackfillConversationLogSummaries(model.ConversationLogBackfillOptions{
		AfterId:        *afterId,
		Limit:          *limit,
		StartTimestamp: *startTimestamp,
		EndTimestamp:   *endTimestamp,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to backfill conversation log summaries:", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "backfill_conversation_log_summaries: scanned=%d updated=%d skipped=%d last_id=%d\n", result.Scanned, result.Updated, result.Skipped, result.LastId)
	return 0
}
