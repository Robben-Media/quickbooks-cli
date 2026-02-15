package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
)

type BillsCmd struct {
	List BillsListCmd `cmd:"" help:"List bills"`
	Get  BillsGetCmd  `cmd:"" help:"Get bill by ID"`
}

type BillsListCmd struct {
	Query string `help:"SQL-like query (e.g. SELECT * FROM Bill WHERE TotalAmt > '100')"`
}

func (cmd *BillsListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Bills().List(ctx, cmd.Query)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.Bills) == 0 {
		fmt.Fprintln(os.Stderr, "No bills found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d bills\n\n", len(result.Bills))

	for _, bill := range result.Bills {
		fmt.Printf("ID: %s  Doc#: %s  Vendor: %s  Total: %.2f  Balance: %.2f  Date: %s\n",
			bill.ID, bill.DocNumber, bill.VendorRef.Name, bill.TotalAmt, bill.Balance, bill.TxnDate)
	}

	return nil
}

type BillsGetCmd struct {
	ID string `arg:"" required:"" help:"Bill ID"`
}

func (cmd *BillsGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Bills().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Doc Number: %s\n", result.DocNumber)
	fmt.Printf("Vendor: %s (%s)\n", result.VendorRef.Name, result.VendorRef.Value)
	fmt.Printf("Date: %s\n", result.TxnDate)
	fmt.Printf("Due Date: %s\n", result.DueDate)
	fmt.Printf("Total: %.2f\n", result.TotalAmt)
	fmt.Printf("Balance: %.2f\n", result.Balance)

	return nil
}
