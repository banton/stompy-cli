package cmd

import (
	"fmt"
	"strconv"

	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var bugCmd = &cobra.Command{
	Use:   "bug",
	Short: "View bug reports",
}

var bugListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bug reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		resp, err := apiClient.ListBugReports(project, status, limit, offset)
		if err != nil {
			return err
		}

		f := getFormatter()
		headers := []string{"ID", "TITLE", "STATUS", "SEVERITY", "CREATED"}
		var rows [][]string
		for _, b := range resp.BugReports {
			statusStr := b.Status
			severityStr := b.Severity
			if isTableOutput() {
				statusStr = output.ColorStatus(b.Status)
				severityStr = output.ColorPriority(b.Severity)
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", b.ID),
				b.Title,
				statusStr,
				severityStr,
				b.CreatedAt.Local().Format("2006-01-02"),
			})
		}

		fmt.Print(f.FormatTable(headers, rows))
		if isTableOutput() {
			fmt.Printf("\nTotal: %d bug reports\n", resp.Total)
		}
		return nil
	},
}

var bugGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show bug report details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid bug report ID: %s", args[0])
		}

		resp, err := apiClient.GetBugReport(project, id)
		if err != nil {
			return err
		}

		f := getFormatter()
		fields := []output.KeyValue{
			{Key: "ID", Value: fmt.Sprintf("%d", resp.ID)},
			{Key: "Title", Value: resp.Title},
			{Key: "Status", Value: resp.Status},
			{Key: "Severity", Value: resp.Severity},
			{Key: "Created", Value: resp.CreatedAt.Local().Format("2006-01-02 15:04:05")},
		}
		if resp.Description != "" {
			fields = append(fields, output.KeyValue{Key: "Description", Value: resp.Description})
		}
		if resp.Steps != "" {
			fields = append(fields, output.KeyValue{Key: "Steps to Reproduce", Value: resp.Steps})
		}
		if resp.Expected != "" {
			fields = append(fields, output.KeyValue{Key: "Expected Behavior", Value: resp.Expected})
		}
		if resp.Actual != "" {
			fields = append(fields, output.KeyValue{Key: "Actual Behavior", Value: resp.Actual})
		}

		fmt.Print(f.FormatSingle(fields))
		return nil
	},
}

func init() {
	bugListCmd.Flags().String("status", "", "Filter by status (new, confirmed, in_progress, fixed, wont_fix)")
	bugListCmd.Flags().Int("limit", 0, "Limit results")
	bugListCmd.Flags().Int("offset", 0, "Offset for pagination")

	bugCmd.AddCommand(bugListCmd)
	bugCmd.AddCommand(bugGetCmd)
	rootCmd.AddCommand(bugCmd)
}
