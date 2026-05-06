package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
)

type BillPaymentsCmd struct {
	List BillPaymentsListCmd `cmd:"" help:"List vendor bill payments"`
	Get  BillPaymentsGetCmd  `cmd:"" help:"Get vendor bill payment by ID"`
}

type BillPaymentsListCmd struct {
	Query string `help:"SQL-like query (e.g. SELECT * FROM BillPayment WHERE TotalAmt > '100')"`
}

func (cmd *BillPaymentsListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.BillPayments().List(ctx, cmd.Query)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "VENDOR", "PAY_TYPE", "DATE", "TOTAL"}

		var rows [][]string
		for _, payment := range result.BillPayments {
			rows = append(rows, []string{
				payment.ID,
				payment.DocNumber,
				payment.VendorRef.Name,
				payment.PayType,
				payment.TxnDate,
				fmt.Sprintf("%.2f", payment.TotalAmt),
			})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.BillPayments) == 0 {
		fmt.Fprintln(os.Stderr, "No vendor bill payments found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d vendor bill payments\n\n", len(result.BillPayments))

	for _, payment := range result.BillPayments {
		fmt.Printf("ID: %s  Doc#: %s  Vendor: %s  Pay Type: %s  Total: %.2f  Date: %s\n",
			payment.ID, payment.DocNumber, payment.VendorRef.Name, payment.PayType, payment.TotalAmt, payment.TxnDate)
	}

	return nil
}

type BillPaymentsGetCmd struct {
	ID string `arg:"" required:"" help:"Bill payment ID"`
}

func (cmd *BillPaymentsGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.BillPayments().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "VENDOR", "PAY_TYPE", "DATE", "TOTAL"}
		rows := [][]string{{result.ID, result.DocNumber, result.VendorRef.Name, result.PayType, result.TxnDate, fmt.Sprintf("%.2f", result.TotalAmt)}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Doc Number: %s\n", result.DocNumber)
	fmt.Printf("Vendor: %s (%s)\n", result.VendorRef.Name, result.VendorRef.Value)
	fmt.Printf("Pay Type: %s\n", result.PayType)
	fmt.Printf("Date: %s\n", result.TxnDate)
	fmt.Printf("Total: %.2f\n", result.TotalAmt)

	if len(result.Line) > 0 {
		fmt.Println("\nLinked Transactions:")

		for _, line := range result.Line {
			for _, linked := range line.LinkedTxn {
				fmt.Printf("  - %s %s: %.2f\n", linked.TxnType, linked.TxnID, line.Amount)
			}
		}
	}

	return nil
}
