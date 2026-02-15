package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
	"github.com/builtbyrobben/quickbooks-cli/internal/quickbooks"
	"github.com/builtbyrobben/quickbooks-cli/internal/secrets"
)

type InvoicesCmd struct {
	List   InvoicesListCmd   `cmd:"" help:"List invoices"`
	Get    InvoicesGetCmd    `cmd:"" help:"Get invoice by ID"`
	Create InvoicesCreateCmd `cmd:"" help:"Create a new invoice"`
	Send   InvoicesSendCmd   `cmd:"" help:"Send invoice by email"`
	Void   InvoicesVoidCmd   `cmd:"" help:"Void an invoice"`
}

type InvoicesListCmd struct {
	Query    string `help:"SQL-like query (e.g. SELECT * FROM Invoice WHERE TotalAmt > '100')"`
	Page     int    `help:"Page number" default:"1"`
	PageSize int    `help:"Results per page" default:"50"`
}

func (cmd *InvoicesListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Invoices().List(ctx, cmd.Query, cmd.Page, cmd.PageSize)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.Invoices) == 0 {
		fmt.Fprintln(os.Stderr, "No invoices found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d invoices\n\n", len(result.Invoices))

	for _, inv := range result.Invoices {
		fmt.Printf("ID: %s  Doc#: %s  Customer: %s  Total: %.2f  Balance: %.2f  Date: %s\n",
			inv.ID, inv.DocNumber, inv.CustomerRef.Name, inv.TotalAmt, inv.Balance, inv.TxnDate)
	}

	return nil
}

type InvoicesGetCmd struct {
	ID string `arg:"" required:"" help:"Invoice ID"`
}

func (cmd *InvoicesGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Invoices().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Doc Number: %s\n", result.DocNumber)
	fmt.Printf("Customer: %s (%s)\n", result.CustomerRef.Name, result.CustomerRef.Value)
	fmt.Printf("Date: %s\n", result.TxnDate)
	fmt.Printf("Due Date: %s\n", result.DueDate)
	fmt.Printf("Total: %.2f\n", result.TotalAmt)
	fmt.Printf("Balance: %.2f\n", result.Balance)
	fmt.Printf("Email Status: %s\n", result.EmailStatus)

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

type InvoicesCreateCmd struct {
	Customer string  `required:"" help:"Customer ID"`
	Item     string  `required:"" help:"Item/service ID"`
	Qty      float64 `required:"" help:"Quantity"`
	Rate     float64 `required:"" help:"Unit rate/price"`
	DueDate  string  `help:"Due date (YYYY-MM-DD)"`
}

func (cmd *InvoicesCreateCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	req := quickbooks.CreateInvoiceRequest{
		CustomerID: cmd.Customer,
		ItemID:     cmd.Item,
		Qty:        cmd.Qty,
		Rate:       cmd.Rate,
		DueDate:    cmd.DueDate,
	}

	result, err := client.Invoices().Create(ctx, req)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Invoice created\n\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Total: %.2f\n", result.TotalAmt)

	return nil
}

type InvoicesSendCmd struct {
	ID string `arg:"" required:"" help:"Invoice ID"`
	To string `help:"Email address to send to (overrides customer email)"`
}

func (cmd *InvoicesSendCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Invoices().Send(ctx, cmd.ID, cmd.To)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Invoice %s sent\n", result.ID)

	return nil
}

type InvoicesVoidCmd struct {
	ID        string `arg:"" required:"" help:"Invoice ID"`
	SyncToken string `required:"" help:"Sync token (from invoice get)"`
}

func (cmd *InvoicesVoidCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Invoices().Void(ctx, cmd.ID, cmd.SyncToken)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Invoice %s voided\n", result.ID)

	return nil
}

// getQuickBooksClient builds a QuickBooks client from stored credentials.
func getQuickBooksClient() (*quickbooks.Client, error) {
	store, err := secrets.OpenDefault()
	if err != nil {
		return nil, fmt.Errorf("open credential store: %w", err)
	}

	clientID, err := resolveCredential(store, secrets.KeyClientID, "QUICKBOOKS_CLIENT_ID")
	if err != nil {
		return nil, fmt.Errorf("client ID not configured: %w", err)
	}

	clientSecret, err := resolveCredential(store, secrets.KeyClientSecret, "QUICKBOOKS_CLIENT_SECRET")
	if err != nil {
		return nil, fmt.Errorf("client secret not configured: %w", err)
	}

	refreshToken, err := resolveCredential(store, secrets.KeyRefreshToken, "QUICKBOOKS_REFRESH_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("refresh token not configured (run 'quickbooks-cli auth login'): %w", err)
	}

	realmID, err := resolveCredential(store, secrets.KeyRealmID, "QUICKBOOKS_REALM_ID")
	if err != nil {
		return nil, fmt.Errorf("realm ID not configured (run 'quickbooks-cli auth set-realm <id>'): %w", err)
	}

	accessToken, _ := resolveCredential(store, secrets.KeyAccessToken, "")

	return quickbooks.NewClientWithOAuth(accessToken, realmID, clientID, clientSecret, refreshToken, store), nil
}
