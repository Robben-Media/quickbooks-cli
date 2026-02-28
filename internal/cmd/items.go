package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
)

type ItemsCmd struct {
	List ItemsListCmd `cmd:"" help:"List items/services"`
}

type ItemsListCmd struct{}

func (cmd *ItemsListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Items().List(ctx)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "NAME", "TYPE", "PRICE"}

		var rows [][]string
		for _, item := range result.Items {
			rows = append(rows, []string{item.ID, item.Name, item.Type, fmt.Sprintf("%.2f", item.UnitPrice)})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.Items) == 0 {
		fmt.Fprintln(os.Stderr, "No items found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d items\n\n", len(result.Items))

	for _, item := range result.Items {
		fmt.Printf("ID: %s  Name: %s  Type: %s  Price: %.2f\n",
			item.ID, item.Name, item.Type, item.UnitPrice)
	}

	return nil
}
