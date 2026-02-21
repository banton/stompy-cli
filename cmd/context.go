package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/banton/stompy-cli/internal/api"
	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage contexts (persistent memory)",
}

var contextLockCmd = &cobra.Command{
	Use:   "lock <topic>",
	Short: "Lock (create) a context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		content, err := resolveContent(cmd)
		if err != nil {
			return err
		}

		tags, _ := cmd.Flags().GetString("tags")
		priority, _ := cmd.Flags().GetString("priority")
		force, _ := cmd.Flags().GetBool("force")

		req := api.ContextCreateRequest{
			Topic:      args[0],
			Content:    content,
			Tags:       tags,
			Priority:   priority,
			ForceStore: force,
		}

		resp, err := apiClient.LockContext(project, req)
		if err != nil {
			return err
		}

		fmt.Printf("Context locked: %s (version %s)\n", resp.Topic, resp.Version)
		return nil
	},
}

var contextRecallCmd = &cobra.Command{
	Use:   "recall <topic>",
	Short: "Recall (read) a context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		version, _ := cmd.Flags().GetString("version")

		resp, err := apiClient.GetContext(project, args[0], version)
		if err != nil {
			return err
		}

		f := getFormatter()
		fields := []output.KeyValue{
			{Key: "Topic", Value: resp.Topic},
			{Key: "Version", Value: resp.Version},
			{Key: "Priority", Value: resp.Priority},
		}
		if len(resp.Tags) > 0 {
			fields = append(fields, output.KeyValue{Key: "Tags", Value: strings.Join(resp.Tags, ", ")})
		}
		fields = append(fields, output.KeyValue{Key: "Content", Value: resp.Content})

		fmt.Print(f.FormatSingle(fields))
		return nil
	},
}

var contextUnlockCmd = &cobra.Command{
	Use:   "unlock <topic>",
	Short: "Unlock (delete) a context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		version, _ := cmd.Flags().GetString("version")
		force, _ := cmd.Flags().GetBool("force")
		noArchive, _ := cmd.Flags().GetBool("no-archive")

		resp, err := apiClient.UnlockContext(project, args[0], version, force, noArchive)
		if err != nil {
			return err
		}

		archivedStr := ""
		if resp.Archived {
			archivedStr = " (archived)"
		}
		fmt.Printf("Context unlocked: %s%s\n", resp.Topic, archivedStr)
		return nil
	},
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		priority, _ := cmd.Flags().GetString("priority")
		tags, _ := cmd.Flags().GetString("tags")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		resp, err := apiClient.ListContexts(project, priority, tags, limit, offset)
		if err != nil {
			return err
		}

		f := getFormatter()
		headers := []string{"ID", "TOPIC", "VERSION", "PRIORITY", "TAGS", "ACCESS COUNT"}
		var rows [][]string
		for _, c := range resp.Contexts {
			rows = append(rows, []string{
				fmt.Sprintf("%d", c.ID),
				c.Topic,
				c.Version,
				c.Priority,
				strings.Join(c.Tags, ", "),
				fmt.Sprintf("%d", c.AccessCount),
			})
		}

		fmt.Print(f.FormatTable(headers, rows))
		fmt.Printf("\nTotal: %d contexts\n", resp.Total)
		return nil
	},
}

var contextSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search contexts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")

		resp, err := apiClient.SearchContexts(project, args[0], limit)
		if err != nil {
			return err
		}

		f := getFormatter()
		headers := []string{"ID", "TOPIC", "PRIORITY", "PREVIEW"}
		var rows [][]string
		for _, c := range resp.Contexts {
			preview := ""
			if c.Preview != nil {
				preview = *c.Preview
				if len(preview) > 60 {
					preview = preview[:57] + "..."
				}
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", c.ID),
				c.Topic,
				c.Priority,
				preview,
			})
		}

		fmt.Print(f.FormatTable(headers, rows))
		fmt.Printf("\nFound: %d contexts\n", resp.Total)
		return nil
	},
}

var contextUpdateCmd = &cobra.Command{
	Use:   "update <topic>",
	Short: "Update a context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		content, err := resolveContent(cmd)
		if err != nil {
			return err
		}

		priority, _ := cmd.Flags().GetString("priority")
		tags, _ := cmd.Flags().GetString("tags")

		req := api.ContextUpdateRequest{
			Content:  content,
			Priority: priority,
			Tags:     tags,
		}

		resp, err := apiClient.UpdateContext(project, args[0], req)
		if err != nil {
			return err
		}

		fmt.Printf("Context updated: %s (version %s)\n", resp.Topic, resp.Version)
		return nil
	},
}

var contextMoveCmd = &cobra.Command{
	Use:   "move <topic>",
	Short: "Move a context to another project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		target, _ := cmd.Flags().GetString("to")
		if target == "" {
			return fmt.Errorf("--to flag is required")
		}

		resp, err := apiClient.MoveContext(project, args[0], target)
		if err != nil {
			return err
		}

		fmt.Printf("Context %q moved to project %q\n", resp.Topic, resp.TargetProject)
		return nil
	},
}

func init() {
	contextLockCmd.Flags().String("content", "", "Context content (use @file to read from file)")
	contextLockCmd.Flags().String("tags", "", "Comma-separated tags")
	contextLockCmd.Flags().String("priority", "", "Priority: always_check, important, reference, nice_to_have")
	contextLockCmd.Flags().Bool("force", false, "Force store even if similar content exists")

	contextRecallCmd.Flags().String("version", "", "Specific version to recall")

	contextUnlockCmd.Flags().String("version", "", "Version to unlock: all, latest, or specific version")
	contextUnlockCmd.Flags().Bool("force", false, "Force unlock without confirmation")
	contextUnlockCmd.Flags().Bool("no-archive", false, "Delete without archiving")

	contextListCmd.Flags().String("priority", "", "Filter by priority")
	contextListCmd.Flags().String("tags", "", "Filter by tags")
	contextListCmd.Flags().Int("limit", 0, "Limit results")
	contextListCmd.Flags().Int("offset", 0, "Offset for pagination")

	contextSearchCmd.Flags().Int("limit", 0, "Limit results")

	contextUpdateCmd.Flags().String("content", "", "New content (use @file to read from file)")
	contextUpdateCmd.Flags().String("priority", "", "New priority")
	contextUpdateCmd.Flags().String("tags", "", "New tags")

	contextMoveCmd.Flags().String("to", "", "Target project name (required)")

	contextCmd.AddCommand(contextLockCmd)
	contextCmd.AddCommand(contextRecallCmd)
	contextCmd.AddCommand(contextUnlockCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextSearchCmd)
	contextCmd.AddCommand(contextUpdateCmd)
	contextCmd.AddCommand(contextMoveCmd)
	rootCmd.AddCommand(contextCmd)
}

// resolveContent reads content from --content flag, @file reference, or stdin.
func resolveContent(cmd *cobra.Command) (string, error) {
	contentFlag, _ := cmd.Flags().GetString("content")

	// Check if stdin has data (pipe)
	if contentFlag == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", fmt.Errorf("reading stdin: %w", err)
			}
			return string(data), nil
		}
		return "", fmt.Errorf("--content flag is required (or pipe content via stdin)")
	}

	// @file reference
	if strings.HasPrefix(contentFlag, "@") {
		filePath := strings.TrimPrefix(contentFlag, "@")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("reading file %q: %w", filePath, err)
		}
		return string(data), nil
	}

	return contentFlag, nil
}
