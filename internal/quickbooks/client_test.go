package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/builtbyrobben/quickbooks-cli/internal/api"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewClient("test-token", "12345", api.WithBaseURL(server.URL))
}

func TestInvoices_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			t.Error("expected query parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Invoices: []Invoice{
					{ID: "1", DocNumber: "1001", TotalAmt: 150.00},
					{ID: "2", DocNumber: "1002", TotalAmt: 250.00},
				},
				MaxResults: 2,
			},
		})
	}))

	result, err := client.Invoices().List(context.Background(), "", 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Invoices) != 2 {
		t.Errorf("expected 2 invoices, got %d", len(result.Invoices))
	}

	if result.Invoices[0].ID != "1" {
		t.Errorf("expected invoice ID '1', got %q", result.Invoices[0].ID)
	}
}

func TestInvoices_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceEnvelope{
			Invoice: Invoice{
				ID:        "42",
				DocNumber: "1001",
				TotalAmt:  500.00,
				Balance:   500.00,
			},
		})
	}))

	result, err := client.Invoices().Get(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "42" {
		t.Errorf("expected ID '42', got %q", result.ID)
	}

	if result.TotalAmt != 500.00 {
		t.Errorf("expected TotalAmt 500, got %.2f", result.TotalAmt)
	}
}

func TestInvoices_Get_EmptyID(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("should not reach server")
	}))

	_, err := client.Invoices().Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestInvoices_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceEnvelope{
			Invoice: Invoice{ID: "99", TotalAmt: 200.00},
		})
	}))

	req := CreateInvoiceRequest{
		CustomerID: "10",
		ItemID:     "5",
		Qty:        2,
		Rate:       100,
	}

	result, err := client.Invoices().Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "99" {
		t.Errorf("expected ID '99', got %q", result.ID)
	}
}

func TestInvoices_Create_MissingCustomer(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("should not reach server")
	}))

	req := CreateInvoiceRequest{ItemID: "5", Qty: 1, Rate: 100}

	_, err := client.Invoices().Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing customer")
	}
}

func TestBills_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Bills: []Bill{
					{ID: "1", DocNumber: "B001", TotalAmt: 300.00},
				},
			},
		})
	}))

	result, err := client.Bills().List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Bills) != 1 {
		t.Errorf("expected 1 bill, got %d", len(result.Bills))
	}
}

func TestBills_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(billEnvelope{
			Bill: Bill{ID: "10", TotalAmt: 450.00},
		})
	}))

	result, err := client.Bills().Get(context.Background(), "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "10" {
		t.Errorf("expected ID '10', got %q", result.ID)
	}
}

func TestPayments_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Payments: []Payment{
					{ID: "1", TotalAmt: 100.00},
				},
			},
		})
	}))

	result, err := client.Payments().List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Payments) != 1 {
		t.Errorf("expected 1 payment, got %d", len(result.Payments))
	}
}

func TestPayments_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(paymentEnvelope{
			Payment: Payment{ID: "77", TotalAmt: 150.00},
		})
	}))

	req := CreatePaymentRequest{
		CustomerID: "10",
		Amount:     150,
		InvoiceID:  "42",
	}

	result, err := client.Payments().Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "77" {
		t.Errorf("expected ID '77', got %q", result.ID)
	}
}

func TestPayments_Create_MissingCustomer(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("should not reach server")
	}))

	req := CreatePaymentRequest{Amount: 100, InvoiceID: "42"}

	_, err := client.Payments().Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing customer")
	}
}

func TestPurchases_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "SELECT * FROM Purchase" {
			t.Errorf("expected default Purchase query, got %q", r.URL.Query().Get("query"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Purchases: []Purchase{
					{ID: "1", PaymentType: "CreditCard", TotalAmt: 82.45},
				},
			},
		})
	}))

	result, err := client.Purchases().List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Purchases) != 1 {
		t.Errorf("expected 1 purchase, got %d", len(result.Purchases))
	}
}

func TestPurchases_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/company/12345/purchase/10" {
			t.Errorf("expected purchase path, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(purchaseEnvelope{
			Purchase: Purchase{ID: "10", PaymentType: "Check", TotalAmt: 25.00},
		})
	}))

	result, err := client.Purchases().Get(context.Background(), "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PaymentType != "Check" {
		t.Errorf("expected Check payment type, got %q", result.PaymentType)
	}
}

func TestBillPayments_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "SELECT * FROM BillPayment" {
			t.Errorf("expected default BillPayment query, got %q", r.URL.Query().Get("query"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				BillPayments: []BillPayment{
					{ID: "1", PayType: "Check", TotalAmt: 100.00},
				},
			},
		})
	}))

	result, err := client.BillPayments().List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.BillPayments) != 1 {
		t.Errorf("expected 1 bill payment, got %d", len(result.BillPayments))
	}
}

func TestBillPayments_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/company/12345/billpayment/20" {
			t.Errorf("expected bill payment path, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(billPaymentEnvelope{
			BillPayment: BillPayment{ID: "20", PayType: "CreditCard", TotalAmt: 200.00},
		})
	}))

	result, err := client.BillPayments().Get(context.Background(), "20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PayType != "CreditCard" {
		t.Errorf("expected CreditCard pay type, got %q", result.PayType)
	}
}

func TestJournalEntries_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "SELECT * FROM JournalEntry" {
			t.Errorf("expected default JournalEntry query, got %q", r.URL.Query().Get("query"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				JournalEntries: []JournalEntry{
					{ID: "1", DocNumber: "JE-1"},
				},
			},
		})
	}))

	result, err := client.JournalEntries().List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.JournalEntries) != 1 {
		t.Errorf("expected 1 journal entry, got %d", len(result.JournalEntries))
	}
}

func TestJournalEntries_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/company/12345/journalentry/30" {
			t.Errorf("expected journal entry path, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(journalEntryEnvelope{
			JournalEntry: JournalEntry{ID: "30", DocNumber: "JE-30"},
		})
	}))

	result, err := client.JournalEntries().Get(context.Background(), "30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DocNumber != "JE-30" {
		t.Errorf("expected JE-30 doc number, got %q", result.DocNumber)
	}
}

func TestCustomers_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Customers: []Customer{
					{ID: "1", DisplayName: "Acme Corp"},
				},
			},
		})
	}))

	result, err := client.Customers().List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Customers) != 1 {
		t.Errorf("expected 1 customer, got %d", len(result.Customers))
	}
}

func TestCustomers_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(customerEnvelope{
			Customer: Customer{ID: "55", DisplayName: "New Corp"},
		})
	}))

	req := CreateCustomerRequest{DisplayName: "New Corp", Email: "test@example.com"}

	result, err := client.Customers().Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DisplayName != "New Corp" {
		t.Errorf("expected 'New Corp', got %q", result.DisplayName)
	}
}

func TestCustomers_Create_MissingName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("should not reach server")
	}))

	_, err := client.Customers().Create(context.Background(), CreateCustomerRequest{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestVendors_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Vendors: []Vendor{
					{ID: "1", DisplayName: "Supplier Inc"},
				},
			},
		})
	}))

	result, err := client.Vendors().List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Vendors) != 1 {
		t.Errorf("expected 1 vendor, got %d", len(result.Vendors))
	}
}

func TestItems_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(queryEnvelope{
			QueryResponse: QueryResponse{
				Items: []Item{
					{ID: "1", Name: "Widget", Type: "Service", UnitPrice: 50},
				},
			},
		})
	}))

	result, err := client.Items().List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestReports_ProfitAndLoss(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("start_date") != "2025-01-01" {
			t.Errorf("expected start_date 2025-01-01, got %q", r.URL.Query().Get("start_date"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Report{
			Header: ReportHeader{
				ReportName:  "ProfitAndLoss",
				StartPeriod: "2025-01-01",
				EndPeriod:   "2025-12-31",
			},
		})
	}))

	result, err := client.Reports().ProfitAndLoss(context.Background(), "2025-01-01", "2025-12-31", "Accrual")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header.ReportName != "ProfitAndLoss" {
		t.Errorf("expected report name ProfitAndLoss, got %q", result.Header.ReportName)
	}
}

func TestReports_BalanceSheet(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Report{
			Header: ReportHeader{
				ReportName: "BalanceSheet",
			},
		})
	}))

	result, err := client.Reports().BalanceSheet(context.Background(), "2025-12-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header.ReportName != "BalanceSheet" {
		t.Errorf("expected report name BalanceSheet, got %q", result.Header.ReportName)
	}
}

func TestReports_GeneralLedger(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/company/12345/reports/GeneralLedger" {
			t.Errorf("expected GeneralLedger path, got %q", r.URL.Path)
		}

		if r.URL.Query().Get("accounting_method") != "Cash" {
			t.Errorf("expected Cash method, got %q", r.URL.Query().Get("accounting_method"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Report{
			Header: ReportHeader{
				ReportName: "GeneralLedger",
			},
		})
	}))

	result, err := client.Reports().GeneralLedger(context.Background(), "2026-01-01", "2026-01-31", "Cash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header.ReportName != "GeneralLedger" {
		t.Errorf("expected report name GeneralLedger, got %q", result.Header.ReportName)
	}
}

func TestReports_TransactionList(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/company/12345/reports/TransactionList" {
			t.Errorf("expected TransactionList path, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Report{
			Header: ReportHeader{
				ReportName: "TransactionList",
			},
		})
	}))

	result, err := client.Reports().TransactionList(context.Background(), "2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header.ReportName != "TransactionList" {
		t.Errorf("expected report name TransactionList, got %q", result.Header.ReportName)
	}
}

func TestInvoices_Void(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Query().Get("operation") != "void" {
			t.Error("expected operation=void query parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceEnvelope{
			Invoice: Invoice{ID: "42"},
		})
	}))

	result, err := client.Invoices().Void(context.Background(), "42", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "42" {
		t.Errorf("expected ID '42', got %q", result.ID)
	}
}

func TestInvoices_Void_MissingSyncToken(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("should not reach server")
	}))

	_, err := client.Invoices().Void(context.Background(), "42", "")
	if err == nil {
		t.Fatal("expected error for missing sync token")
	}
}

func TestInvoices_Send(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceEnvelope{
			Invoice: Invoice{ID: "42", EmailStatus: "EmailSent"},
		})
	}))

	result, err := client.Invoices().Send(context.Background(), "42", "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "42" {
		t.Errorf("expected ID '42', got %q", result.ID)
	}
}
