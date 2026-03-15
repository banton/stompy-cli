package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/banton/stompy-cli/internal/api"
	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var conflictCmd = &cobra.Command{
	Use:   "conflict",
	Short: "Manage conflicts between contexts",
}

var conflictListCmd = &cobra.Command{
	Use:   "list",
	Short: "List detected conflicts",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		resp, err := apiClient.ListConflicts(project, status, limit, offset)
		if err != nil {
			return err
		}

		f := getFormatter()
		headers := []string{"ID", "CONTEXT A", "CONTEXT B", "TYPE", "SEVERITY", "STATUS"}
		var rows [][]string
		for _, c := range resp.Conflicts {
			row := []string{
				fmt.Sprintf("%d", c.ID),
				c.ContextATopic,
				c.ContextBTopic,
				c.ConflictType,
			}
			if isTableOutput() {
				row = append(row,
					output.ColorPriority(c.Severity),
					output.ColorStatus(c.Status),
				)
			} else {
				row = append(row, c.Severity, c.Status)
			}
			rows = append(rows, row)
		}

		fmt.Print(f.FormatTable(headers, rows))
		if isTableOutput() {
			fmt.Printf("\nTotal: %d conflicts\n", resp.Total)
		}
		return nil
	},
}

var conflictGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show conflict details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid conflict ID: %s", args[0])
		}

		resp, err := apiClient.GetConflict(project, id)
		if err != nil {
			return err
		}

		f := getFormatter()
		fields := []output.KeyValue{
			{Key: "ID", Value: fmt.Sprintf("%d", resp.ID)},
			{Key: "Context A", Value: resp.ContextATopic},
			{Key: "Context B", Value: resp.ContextBTopic},
			{Key: "Type", Value: resp.ConflictType},
			{Key: "Severity", Value: resp.Severity},
			{Key: "Status", Value: resp.Status},
			{Key: "Description", Value: resp.Description},
			{Key: "Created", Value: resp.CreatedAt.Local().Format("2006-01-02 15:04:05")},
		}
		if resp.Resolution != nil {
			fields = append(fields, output.KeyValue{Key: "Resolution", Value: *resp.Resolution})
		}
		if resp.ResolvedAt != nil {
			fields = append(fields, output.KeyValue{Key: "Resolved At", Value: resp.ResolvedAt.Local().Format("2006-01-02 15:04:05")})
		}

		fmt.Print(f.FormatSingle(fields))
		return nil
	},
}

var conflictDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Trigger conflict detection",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		scope, _ := cmd.Flags().GetString("scope")
		req := api.ConflictDetectRequest{Scope: scope}

		resp, err := apiClient.DetectConflicts(project, req)
		if err != nil {
			return err
		}

		fmt.Printf("%s Scanned %d contexts, found %d conflicts\n",
			output.Success("✓"),
			resp.Scanned,
			resp.ConflictsFound)
		return nil
	},
}

var conflictResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "Resolve a conflict",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid conflict ID: %s", args[0])
		}

		resolution, _ := cmd.Flags().GetString("resolution")
		if resolution == "" {
			return fmt.Errorf("--resolution is required (dismiss, keep_a, keep_b, merge)")
		}

		valid := map[string]bool{"dismiss": true, "keep_a": true, "keep_b": true, "merge": true}
		if !valid[strings.ToLower(resolution)] {
			return fmt.Errorf("invalid resolution %q: must be one of dismiss, keep_a, keep_b, merge", resolution)
		}

		req := api.ConflictResolveRequest{Resolution: resolution}
		resp, err := apiClient.ResolveConflict(project, id, req)
		if err != nil {
			return err
		}

		fmt.Printf("%s Conflict %d resolved (%s)\n", output.Success("✓"), resp.ID, resp.Status)
		return nil
	},
}

func init() {
	conflictListCmd.Flags().String("status", "", "Filter by status (unresolved, resolved, dismissed)")
	conflictListCmd.Flags().Int("limit", 0, "Limit results")
	conflictListCmd.Flags().Int("offset", 0, "Offset for pagination")

	conflictDetectCmd.Flags().String("scope", "", "Detection scope (all, recent)")

	conflictResolveCmd.Flags().String("resolution", "", "Resolution: dismiss, keep_a, keep_b, merge (required)")

	conflictCmd.AddCommand(conflictListCmd)
	conflictCmd.AddCommand(conflictGetCmd)
	conflictCmd.AddCommand(conflictDetectCmd)
	conflictCmd.AddCommand(conflictResolveCmd)
	rootCmd.AddCommand(conflictCmd)
}
