package cmd

import (
	"fmt"
	"strconv"

	"github.com/banton/stompy-cli/internal/api"
	"github.com/spf13/cobra"
)

var ticketLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage ticket links",
}

var ticketLinkAddCmd = &cobra.Command{
	Use:   "add <ticket-id>",
	Short: "Add a link between tickets",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid ticket ID: %s", args[0])
		}

		target, _ := cmd.Flags().GetInt("target")
		if target == 0 {
			return fmt.Errorf("--target is required")
		}
		linkType, _ := cmd.Flags().GetString("type")
		if linkType == "" {
			return fmt.Errorf("--type is required (blocks, parent, related, duplicate)")
		}

		req := api.LinkCreate{
			TargetID: target,
			LinkType: linkType,
		}

		resp, err := apiClient.AddLink(project, id, req)
		if err != nil {
			return err
		}

		fmt.Printf("Link created: #%d -[%s]-> #%d\n", resp.SourceID, resp.LinkType, resp.TargetID)
		return nil
	},
}

var ticketLinkListCmd = &cobra.Command{
	Use:   "list <ticket-id>",
	Short: "List links for a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid ticket ID: %s", args[0])
		}

		links, err := apiClient.ListLinks(project, id)
		if err != nil {
			return err
		}

		if len(links) == 0 {
			fmt.Println("No links found.")
			return nil
		}

		f := getFormatter()
		headers := []string{"LINK ID", "TYPE", "TARGET ID", "TARGET TITLE", "TARGET STATUS"}
		var rows [][]string
		for _, l := range links {
			rows = append(rows, []string{
				fmt.Sprintf("%d", l.ID),
				l.LinkType,
				fmt.Sprintf("%d", l.TargetID),
				l.TargetTitle,
				l.TargetStatus,
			})
		}

		fmt.Print(f.FormatTable(headers, rows))
		return nil
	},
}

var ticketLinkRemoveCmd = &cobra.Command{
	Use:   "remove <ticket-id> <link-id>",
	Short: "Remove a ticket link",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := getProject()
		if err != nil {
			return err
		}

		ticketID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid ticket ID: %s", args[0])
		}
		linkID, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid link ID: %s", args[1])
		}

		if err := apiClient.RemoveLink(project, ticketID, linkID); err != nil {
			return err
		}

		fmt.Printf("Link %d removed from ticket #%d\n", linkID, ticketID)
		return nil
	},
}

func init() {
	ticketLinkAddCmd.Flags().Int("target", 0, "Target ticket ID (required)")
	ticketLinkAddCmd.Flags().String("type", "", "Link type: blocks, parent, related, duplicate (required)")

	ticketLinkCmd.AddCommand(ticketLinkAddCmd)
	ticketLinkCmd.AddCommand(ticketLinkListCmd)
	ticketLinkCmd.AddCommand(ticketLinkRemoveCmd)
	ticketCmd.AddCommand(ticketLinkCmd)
}
