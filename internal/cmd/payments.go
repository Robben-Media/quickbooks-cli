package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
	"github.com/builtbyrobben/quickbooks-cli/internal/quickbooks"
)

type PaymentsCmd struct {
	List   PaymentsListCmd   `cmd:"" help:"List payments"`
	Create PaymentsCreateCmd `cmd:"" help:"Create a payment"`
}

type PaymentsListCmd struct {
	Query string `help:"SQL-like query (e.g. SELECT * FROM Payment)"`
}

func (cmd *PaymentsListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Payments().List(ctx, cmd.Query)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.Payments) == 0 {
		fmt.Fprintln(os.Stderr, "No payments found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d payments\n\n", len(result.Payments))

	for _, pmt := range result.Payments {
		fmt.Printf("ID: %s  Customer: %s  Amount: %.2f  Date: %s\n",
			pmt.ID, pmt.CustomerRef.Name, pmt.TotalAmt, pmt.TxnDate)
	}

	return nil
}

type PaymentsCreateCmd struct {
	Customer string  `required:"" help:"Customer ID"`
	Amount   float64 `required:"" help:"Payment amount"`
	Invoice  string  `required:"" help:"Invoice ID to apply payment to"`
}

func (cmd *PaymentsCreateCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	req := quickbooks.CreatePaymentRequest{
		CustomerID: cmd.Customer,
		Amount:     cmd.Amount,
		InvoiceID:  cmd.Invoice,
	}

	result, err := client.Payments().Create(ctx, req)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Payment created\n\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Amount: %.2f\n", result.TotalAmt)

	return nil
}
