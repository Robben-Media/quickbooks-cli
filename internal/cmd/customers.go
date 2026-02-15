package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
	"github.com/builtbyrobben/quickbooks-cli/internal/quickbooks"
)

type CustomersCmd struct {
	List   CustomersListCmd   `cmd:"" help:"List customers"`
	Get    CustomersGetCmd    `cmd:"" help:"Get customer by ID"`
	Create CustomersCreateCmd `cmd:"" help:"Create a customer"`
}

type CustomersListCmd struct{}

func (cmd *CustomersListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Customers().List(ctx)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.Customers) == 0 {
		fmt.Fprintln(os.Stderr, "No customers found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d customers\n\n", len(result.Customers))

	for _, cust := range result.Customers {
		email := ""
		if cust.PrimaryEmailAddr != nil {
			email = cust.PrimaryEmailAddr.Address
		}

		fmt.Printf("ID: %s  Name: %s  Balance: %.2f  Email: %s\n",
			cust.ID, cust.DisplayName, cust.Balance, email)
	}

	return nil
}

type CustomersGetCmd struct {
	ID string `arg:"" required:"" help:"Customer ID"`
}

func (cmd *CustomersGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.Customers().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Display Name: %s\n", result.DisplayName)

	if result.CompanyName != "" {
		fmt.Printf("Company: %s\n", result.CompanyName)
	}

	if result.GivenName != "" || result.FamilyName != "" {
		fmt.Printf("Name: %s %s\n", result.GivenName, result.FamilyName)
	}

	if result.PrimaryEmailAddr != nil {
		fmt.Printf("Email: %s\n", result.PrimaryEmailAddr.Address)
	}

	if result.PrimaryPhone != nil {
		fmt.Printf("Phone: %s\n", result.PrimaryPhone.FreeFormNumber)
	}

	fmt.Printf("Balance: %.2f\n", result.Balance)
	fmt.Printf("Active: %v\n", result.Active)

	return nil
}

type CustomersCreateCmd struct {
	Name  string `required:"" help:"Display name"`
	Email string `help:"Email address"`
	Phone string `help:"Phone number"`
}

func (cmd *CustomersCreateCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	req := quickbooks.CreateCustomerRequest{
		DisplayName: cmd.Name,
		Email:       cmd.Email,
		Phone:       cmd.Phone,
	}

	result, err := client.Customers().Create(ctx, req)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Customer created\n\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Name: %s\n", result.DisplayName)

	return nil
}
