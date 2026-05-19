package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/joho/godotenv"
)

func runCleanConversationLogsCommand(args []string) int {
	flags := flag.NewFlagSet("clean-conversation-logs", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	outputPath := flags.String("output", "", "write cleaned data to this file instead of stdout")
	outputFormat := flags.String("format", "jsonl", "output format: jsonl or qae-json")
	sourcePrefix := flags.String("source-prefix", "tokenhub", "source prefix for QAE import ids and source fields")
	afterId := flags.Int("after-id", 0, "only scan source logs with id greater than this value")
	limit := flags.Int("limit", 1000, "maximum source logs to scan")
	startTimestamp := flags.Int64("start-timestamp", 0, "only scan source logs created at or after this unix timestamp")
	endTimestamp := flags.Int64("end-timestamp", 0, "only scan source logs created at or before this unix timestamp")
	username := flags.String("username", "", "only scan source logs for this username")
	modelName := flags.String("model-name", "", "only scan source logs for this model name")
	unexportedOnly := flags.Bool("unexported", false, "only scan source logs whose exported_at is 0")
	markExported := flags.Bool("mark-exported", false, "mark all scanned source logs as exported after a successful clean export")
	includeNonOK := flags.Bool("include-non-ok", false, "include source logs whose status is not ok")
	includeReasoning := flags.Bool("include-reasoning", false, "include response_reasoning_body in cleaned JSONL")
	keepInternal := flags.Bool("keep-internal", false, "do not filter memory, title, observation, and other internal calls")

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

	var exported *bool
	if *unexportedOnly {
		value := false
		exported = &value
	}

	logs, err := model.ExportConversationLogs(model.ConversationLogQuery{
		StartTimestamp: *startTimestamp,
		EndTimestamp:   *endTimestamp,
		AfterId:        *afterId,
		Limit:          *limit,
		Exported:       exported,
		Username:       *username,
		ModelName:      *modelName,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load conversation logs:", err)
		return 1
	}

	result := model.CleanConversationLogs(logs, model.CleanConversationOptions{
		IncludeNonOK:      *includeNonOK,
		IncludeReasoning:  *includeReasoning,
		SkipInternalCalls: !*keepInternal,
	})

	writer, closeWriter, err := cleanConversationOutputWriter(*outputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open output:", err)
		return 1
	}

	if err := writeCleanConversationOutput(writer, result, *outputFormat, *sourcePrefix); err != nil {
		closeWriter()
		fmt.Fprintln(os.Stderr, "failed to write cleaned output:", err)
		return 1
	}
	if err := closeWriter(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to close output:", err)
		return 1
	}

	if *markExported {
		if err := model.MarkConversationLogsExported(result.SeenIds, common.GetTimestamp()); err != nil {
			fmt.Fprintln(os.Stderr, "failed to mark source logs exported:", err)
			return 1
		}
	}

	writeCleanConversationSummary(os.Stderr, result)
	return 0
}

func initConversationCleanCommandResources() error {
	_ = godotenv.Load(".env")

	originalArgs := os.Args
	os.Args = []string{originalArgs[0]}
	common.InitEnv()
	os.Args = originalArgs

	logger.SetupLogger()
	if err := model.InitDB(); err != nil {
		return err
	}
	return model.InitLogDB()
}

func cleanConversationOutputWriter(outputPath string) (io.Writer, func() error, error) {
	if outputPath == "" || outputPath == "-" {
		return os.Stdout, func() error { return nil }, nil
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return nil, nil, err
	}
	return file, file.Close, nil
}

func writeCleanConversationOutput(writer io.Writer, result model.CleanConversationResult, outputFormat string, sourcePrefix string) error {
	switch outputFormat {
	case "jsonl":
		for _, record := range result.Records {
			if err := writeCleanConversationJSONLine(writer, record); err != nil {
				return err
			}
		}
		return nil
	case "qae-json":
		data, err := common.Marshal(model.CleanConversationRecordsToQAE(result.Records, sourcePrefix))
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func writeCleanConversationJSONLine(writer io.Writer, value any) error {
	data, err := common.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = writer.Write(data)
	return err
}

func writeCleanConversationSummary(writer io.Writer, result model.CleanConversationResult) {
	reasonCounts := map[string]int{}
	for _, skip := range result.Skips {
		reasonCounts[skip.Reason]++
	}

	fmt.Fprintf(writer, "clean_conversation_logs: scanned=%d exported=%d skipped=%d\n", len(result.SeenIds), len(result.Records), len(result.Skips))
	if len(reasonCounts) == 0 {
		return
	}

	reasons := make([]string, 0, len(reasonCounts))
	for reason := range reasonCounts {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		fmt.Fprintf(writer, "skip.%s=%d\n", reason, reasonCounts[reason])
	}
}
