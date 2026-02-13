package commands

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available components",
		RunE: func(cmd *cobra.Command, args []string) error {
			writeList("Models", []string{"mock"})
			writeList("Scorers", []string{"exact", "includes"})
			writeList("Solvers", []string{"basic", "chain-of-thought", "few-shot", "multi-step", "self-consistency", "tool-use"})
			writeList("Providers", []string{"mock", "openai", "anthropic", "ollama"})
			writeList("Formats", []string{"table", "json", "html", "markdown", "csv"})
			return nil
		},
	}
	return cmd
}

func writeList(title string, items []string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{title})
	for _, item := range items {
		table.Append([]string{item})
	}
	table.Render()
}
