package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
)

type SalesReceiptsCmd struct {
	List SalesReceiptsListCmd `cmd:"" help:"List sales receipts"`
	Get  SalesReceiptsGetCmd  `cmd:"" help:"Get sales receipt by ID"`
}

type SalesReceiptsListCmd struct {
	Query string `help:"SQL-like query (e.g. SELECT * FROM SalesReceipt WHERE TotalAmt > '100')"`
}

func (cmd *SalesReceiptsListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.SalesReceipts().List(ctx, cmd.Query)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "CUSTOMER", "PAYMENT_METHOD", "DEPOSIT_ACCOUNT", "TOTAL", "DATE"}

		var rows [][]string
		for _, receipt := range result.SalesReceipts {
			rows = append(rows, []string{
				receipt.ID,
				receipt.DocNumber,
				receipt.CustomerRef.Name,
				receipt.PaymentMethodRef.Name,
				receipt.DepositToAccountRef.Name,
				fmt.Sprintf("%.2f", receipt.TotalAmt),
				receipt.TxnDate,
			})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.SalesReceipts) == 0 {
		fmt.Fprintln(os.Stderr, "No sales receipts found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d sales receipts\n\n", len(result.SalesReceipts))

	for _, receipt := range result.SalesReceipts {
		fmt.Printf("ID: %s  Doc#: %s  Customer: %s  Payment: %s  Deposit: %s  Total: %.2f  Date: %s\n",
			receipt.ID,
			receipt.DocNumber,
			receipt.CustomerRef.Name,
			receipt.PaymentMethodRef.Name,
			receipt.DepositToAccountRef.Name,
			receipt.TotalAmt,
			receipt.TxnDate,
		)
	}

	return nil
}

type SalesReceiptsGetCmd struct {
	ID string `arg:"" required:"" help:"Sales receipt ID"`
}

func (cmd *SalesReceiptsGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.SalesReceipts().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "CUSTOMER", "PAYMENT_METHOD", "DEPOSIT_ACCOUNT", "DATE", "TOTAL"}
		rows := [][]string{{
			result.ID,
			result.DocNumber,
			result.CustomerRef.Name,
			result.PaymentMethodRef.Name,
			result.DepositToAccountRef.Name,
			result.TxnDate,
			fmt.Sprintf("%.2f", result.TotalAmt),
		}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Doc Number: %s\n", result.DocNumber)
	fmt.Printf("Customer: %s (%s)\n", result.CustomerRef.Name, result.CustomerRef.Value)
	fmt.Printf("Payment Method: %s (%s)\n", result.PaymentMethodRef.Name, result.PaymentMethodRef.Value)
	fmt.Printf("Deposit Account: %s (%s)\n", result.DepositToAccountRef.Name, result.DepositToAccountRef.Value)
	fmt.Printf("Date: %s\n", result.TxnDate)
	fmt.Printf("Total: %.2f\n", result.TotalAmt)

	if result.PrivateNote != "" {
		fmt.Printf("Private Note: %s\n", result.PrivateNote)
	}

	if len(result.Line) > 0 {
		fmt.Println("\nLine Items:")

		for _, line := range result.Line {
			if line.SalesItemLineDetail != nil {
				fmt.Printf("  - %s: Qty %.0f x $%.2f = $%.2f\n",
					line.SalesItemLineDetail.ItemRef.Name,
					line.SalesItemLineDetail.Qty,
					line.SalesItemLineDetail.UnitPrice,
					line.Amount)
			}
		}
	}

	return nil
}
