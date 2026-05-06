package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
)

type JournalEntriesCmd struct {
	List JournalEntriesListCmd `cmd:"" help:"List general journal entries"`
	Get  JournalEntriesGetCmd  `cmd:"" help:"Get journal entry by ID"`
}

type JournalEntriesListCmd struct {
	Query string `help:"SQL-like query (e.g. SELECT * FROM JournalEntry WHERE TxnDate >= '2026-01-01')"`
}

func (cmd *JournalEntriesListCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.JournalEntries().List(ctx, cmd.Query)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "DATE", "ADJUSTMENT", "NOTE"}

		var rows [][]string
		for _, entry := range result.JournalEntries {
			rows = append(rows, []string{entry.ID, entry.DocNumber, entry.TxnDate, boolString(entry.Adjustment), entry.PrivateNote})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.JournalEntries) == 0 {
		fmt.Fprintln(os.Stderr, "No journal entries found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d journal entries\n\n", len(result.JournalEntries))

	for _, entry := range result.JournalEntries {
		fmt.Printf("ID: %s  Doc#: %s  Date: %s  Adjustment: %t  Note: %s\n",
			entry.ID, entry.DocNumber, entry.TxnDate, entry.Adjustment, entry.PrivateNote)
	}

	return nil
}

type JournalEntriesGetCmd struct {
	ID string `arg:"" required:"" help:"Journal entry ID"`
}

func (cmd *JournalEntriesGetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	result, err := client.JournalEntries().Get(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "DOC_NUM", "DATE", "ADJUSTMENT", "NOTE"}
		rows := [][]string{{result.ID, result.DocNumber, result.TxnDate, boolString(result.Adjustment), result.PrivateNote}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Doc Number: %s\n", result.DocNumber)
	fmt.Printf("Date: %s\n", result.TxnDate)
	fmt.Printf("Adjustment: %t\n", result.Adjustment)

	if result.PrivateNote != "" {
		fmt.Printf("Private Note: %s\n", result.PrivateNote)
	}

	if len(result.Line) > 0 {
		fmt.Println("\nLines:")

		for _, line := range result.Line {
			account := ""
			postingType := ""
			if line.JournalEntryLineDetail != nil {
				account = line.JournalEntryLineDetail.AccountRef.Name
				postingType = line.JournalEntryLineDetail.PostingType
			}

			fmt.Printf("  - %s %s %.2f %s\n", postingType, account, line.Amount, line.Description)
		}
	}

	return nil
}
