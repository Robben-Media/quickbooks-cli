package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
	"github.com/builtbyrobben/quickbooks-cli/internal/quickbooks"
)

type PurchasesCmd struct {
	List PurchasesListCmd `cmd:"" help:"List expenses and purchases"`
	Get  PurchasesGetCmd  `cmd:"" help:"Get purchase by ID"`
}

type PurchasesListCmd struct {
	Query string `help:"SQL-like query (e.g. SELECT * FROM Purchase WHERE TotalAmt > '100')"`
}

func (cmd *PurchasesListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Purchases().List(ctx, cmd.Query)
	if err != nil {
		return err
	}

	return writePurchases(ctx, result.Purchases, "purchases")
}

type PurchasesGetCmd struct {
	ID string `arg:"" required:"" help:"Purchase ID"`
}

func (cmd *PurchasesGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Purchases().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	return writePurchase(ctx, result)
}

type ChecksCmd struct {
	List ChecksListCmd `cmd:"" help:"List checks"`
	Get  ChecksGetCmd  `cmd:"" help:"Get check by purchase ID"`
}

type ChecksListCmd struct {
	Query string `help:"SQL-like query (defaults to SELECT * FROM Purchase WHERE PaymentType = 'Check')"`
}

func (cmd *ChecksListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	query := cmd.Query
	if query == "" {
		query = "SELECT * FROM Purchase WHERE PaymentType = 'Check'"
	}

	result, err := client.Purchases().List(ctx, query)
	if err != nil {
		return err
	}

	return writePurchases(ctx, result.Purchases, "checks")
}

type ChecksGetCmd struct {
	ID string `arg:"" required:"" help:"Purchase ID"`
}

func (cmd *ChecksGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Purchases().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	return writePurchase(ctx, result)
}

type CreditCardChargesCmd struct {
	List CreditCardChargesListCmd `cmd:"" help:"List credit card charges"`
	Get  CreditCardChargesGetCmd  `cmd:"" help:"Get credit card charge by purchase ID"`
}

type CreditCardChargesListCmd struct {
	Query string `help:"SQL-like query (defaults to SELECT * FROM Purchase WHERE PaymentType = 'CreditCard')"`
}

func (cmd *CreditCardChargesListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	query := cmd.Query
	if query == "" {
		query = "SELECT * FROM Purchase WHERE PaymentType = 'CreditCard'"
	}

	result, err := client.Purchases().List(ctx, query)
	if err != nil {
		return err
	}

	return writePurchases(ctx, result.Purchases, "credit card charges")
}

type CreditCardChargesGetCmd struct {
	ID string `arg:"" required:"" help:"Purchase ID"`
}

func (cmd *CreditCardChargesGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Purchases().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	return writePurchase(ctx, result)
}

func writePurchases(ctx context.Context, purchases []quickbooks.Purchase, label string) error {
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{"Purchase": purchases})
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "PAYMENT_TYPE", "PAYEE", "ACCOUNT", "TOTAL", "DATE"}

		var rows [][]string
		for _, purchase := range purchases {
			rows = append(rows, []string{
				purchase.ID,
				purchase.DocNumber,
				purchase.PaymentType,
				purchase.EntityRef.Name,
				purchase.AccountRef.Name,
				fmt.Sprintf("%.2f", purchase.TotalAmt),
				purchase.TxnDate,
			})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(purchases) == 0 {
		fmt.Fprintf(os.Stderr, "No %s found\n", label)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d %s\n\n", len(purchases), label)

	for _, purchase := range purchases {
		fmt.Printf("ID: %s  Doc#: %s  Type: %s  Payee: %s  Account: %s  Total: %.2f  Date: %s\n",
			purchase.ID,
			purchase.DocNumber,
			purchase.PaymentType,
			purchase.EntityRef.Name,
			purchase.AccountRef.Name,
			purchase.TotalAmt,
			purchase.TxnDate,
		)
	}

	return nil
}

func writePurchase(ctx context.Context, purchase *quickbooks.Purchase) error {
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, purchase)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "PAYMENT_TYPE", "PAYEE", "ACCOUNT", "DATE", "TOTAL"}
		rows := [][]string{{
			purchase.ID,
			purchase.DocNumber,
			purchase.PaymentType,
			purchase.EntityRef.Name,
			purchase.AccountRef.Name,
			purchase.TxnDate,
			fmt.Sprintf("%.2f", purchase.TotalAmt),
		}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	fmt.Printf("ID: %s\n", purchase.ID)
	fmt.Printf("Doc Number: %s\n", purchase.DocNumber)
	fmt.Printf("Payment Type: %s\n", purchase.PaymentType)
	fmt.Printf("Payee: %s (%s)\n", purchase.EntityRef.Name, purchase.EntityRef.Value)
	fmt.Printf("Account: %s (%s)\n", purchase.AccountRef.Name, purchase.AccountRef.Value)
	fmt.Printf("Date: %s\n", purchase.TxnDate)
	fmt.Printf("Total: %.2f\n", purchase.TotalAmt)

	if purchase.PrivateNote != "" {
		fmt.Printf("Private Note: %s\n", purchase.PrivateNote)
	}

	if len(purchase.Line) > 0 {
		fmt.Println("\nLines:")

		for _, line := range purchase.Line {
			fmt.Printf("  - %s %.2f %s\n", line.DetailType, line.Amount, line.Description)
		}
	}

	return nil
}
