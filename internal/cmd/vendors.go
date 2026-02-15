package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
)

type VendorsCmd struct {
	List VendorsListCmd `cmd:"" help:"List vendors"`
}

type VendorsListCmd struct{}

func (cmd *VendorsListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Vendors().List(ctx)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "NAME", "BALANCE"}

		var rows [][]string
		for _, v := range result.Vendors {
			rows = append(rows, []string{v.ID, v.DisplayName, fmt.Sprintf("%.2f", v.Balance)})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.Vendors) == 0 {
		fmt.Fprintln(os.Stderr, "No vendors found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d vendors\n\n", len(result.Vendors))

	for _, v := range result.Vendors {
		fmt.Printf("ID: %s  Name: %s  Balance: %.2f\n",
			v.ID, v.DisplayName, v.Balance)
	}

	return nil
}
