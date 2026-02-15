package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
	"github.com/builtbyrobben/quickbooks-cli/internal/quickbooks"
)

type ReportsCmd struct {
	ProfitAndLoss ReportsProfitAndLossCmd `cmd:"" name:"profit-and-loss" help:"Profit and Loss report"`
	BalanceSheet  ReportsBalanceSheetCmd  `cmd:"" name:"balance-sheet" help:"Balance Sheet report"`
}

type ReportsProfitAndLossCmd struct {
	From   string `help:"Start date (YYYY-MM-DD)"`
	To     string `help:"End date (YYYY-MM-DD)"`
	Method string `help:"Accounting method: Accrual or Cash" enum:"Accrual,Cash," default:""`
}

func (cmd *ReportsProfitAndLossCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	report, err := client.Reports().ProfitAndLoss(ctx, cmd.From, cmd.To, cmd.Method)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, report)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"REPORT", "BASIS", "START", "END"}
		rows := [][]string{{report.Header.ReportName, report.Header.ReportBasis, report.Header.StartPeriod, report.Header.EndPeriod}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	printReport(report)

	return nil
}

type ReportsBalanceSheetCmd struct {
	Date string `help:"Report date (YYYY-MM-DD)"`
}

func (cmd *ReportsBalanceSheetCmd) Run(ctx context.Context) error {
	client, err := getQuickBooksClient()
	if err != nil {
		return err
	}

	report, err := client.Reports().BalanceSheet(ctx, cmd.Date)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, report)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"REPORT", "BASIS", "START", "END"}
		rows := [][]string{{report.Header.ReportName, report.Header.ReportBasis, report.Header.StartPeriod, report.Header.EndPeriod}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	printReport(report)

	return nil
}

func printReport(report *quickbooks.Report) {
	fmt.Printf("%s\n", report.Header.ReportName)
	fmt.Printf("Period: %s to %s\n", report.Header.StartPeriod, report.Header.EndPeriod)

	if report.Header.ReportBasis != "" {
		fmt.Printf("Basis: %s\n", report.Header.ReportBasis)
	}

	fmt.Println(strings.Repeat("-", 60))
	printRows(report.Rows.Row, 0)
}

func printRows(rows []quickbooks.ReportRow, depth int) {
	indent := strings.Repeat("  ", depth)

	for _, row := range rows {
		if row.Header != nil && len(row.Header.ColData) > 0 {
			fmt.Printf("%s%s\n", indent, row.Header.ColData[0].Value)
		}

		if len(row.ColData) > 0 {
			values := make([]string, 0, len(row.ColData))
			for _, col := range row.ColData {
				values = append(values, col.Value)
			}

			fmt.Printf("%s%-40s %s\n", indent, values[0], strings.Join(values[1:], "  "))
		}

		if row.Rows != nil {
			printRows(row.Rows.Row, depth+1)
		}

		if row.Summary != nil && len(row.Summary.ColData) > 0 {
			values := make([]string, 0, len(row.Summary.ColData))
			for _, col := range row.Summary.ColData {
				values = append(values, col.Value)
			}

			fmt.Printf("%s%-40s %s\n", indent, values[0], strings.Join(values[1:], "  "))
		}
	}
}
