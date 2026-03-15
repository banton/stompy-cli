package cmd

import (
	"fmt"
	"strconv"

	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage uploaded files and documents",
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload <path>",
	Short: "Upload a document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		label, _ := cmd.Flags().GetString("label")

		resp, err := apiClient.UploadFile(project, args[0], label)
		if err != nil {
			return err
		}

		f := getFormatter()
		fmt.Print(f.FormatSingle([]output.KeyValue{
			{Key: "ID", Value: fmt.Sprintf("%d", resp.ID)},
			{Key: "Filename", Value: resp.Filename},
			{Key: "Size", Value: formatBytes(resp.SizeBytes)},
			{Key: "Created", Value: resp.CreatedAt.Local().Format("2006-01-02 15:04:05")},
		}))
		return nil
	},
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List uploaded documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		search, _ := cmd.Flags().GetString("search")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		resp, err := apiClient.ListFiles(project, search, limit, offset)
		if err != nil {
			return err
		}

		f := getFormatter()
		headers := []string{"ID", "FILENAME", "LABEL", "SIZE", "CREATED"}
		var rows [][]string
		for _, file := range resp.Files {
			rows = append(rows, []string{
				fmt.Sprintf("%d", file.ID),
				file.Filename,
				file.Label,
				formatBytes(file.SizeBytes),
				file.CreatedAt.Local().Format("2006-01-02"),
			})
		}

		fmt.Print(f.FormatTable(headers, rows))
		if isTableOutput() {
			fmt.Printf("\nTotal: %d files\n", resp.Total)
		}
		return nil
	},
}

var fileGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show file details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid file ID: %s", args[0])
		}

		resp, err := apiClient.GetFile(project, id)
		if err != nil {
			return err
		}

		f := getFormatter()
		fields := []output.KeyValue{
			{Key: "ID", Value: fmt.Sprintf("%d", resp.ID)},
			{Key: "Filename", Value: resp.Filename},
			{Key: "Label", Value: resp.Label},
			{Key: "MIME Type", Value: resp.MimeType},
			{Key: "Size", Value: formatBytes(resp.SizeBytes)},
			{Key: "Created", Value: resp.CreatedAt.Local().Format("2006-01-02 15:04:05")},
		}

		fmt.Print(f.FormatSingle(fields))
		return nil
	},
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			return fmt.Errorf("must pass --confirm to delete file")
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid file ID: %s", args[0])
		}

		if err := apiClient.DeleteFile(project, id); err != nil {
			return err
		}

		fmt.Printf("%s File %d deleted\n", output.Success("✓"), id)
		return nil
	},
}

func init() {
	fileUploadCmd.Flags().String("label", "", "Label/description for the file")

	fileListCmd.Flags().String("search", "", "Search files by name")
	fileListCmd.Flags().Int("limit", 0, "Limit results")
	fileListCmd.Flags().Int("offset", 0, "Offset for pagination")

	fileDeleteCmd.Flags().Bool("confirm", false, "Confirm deletion (required)")

	fileCmd.AddCommand(fileUploadCmd)
	fileCmd.AddCommand(fileListCmd)
	fileCmd.AddCommand(fileGetCmd)
	fileCmd.AddCommand(fileDeleteCmd)
	rootCmd.AddCommand(fileCmd)
}
