package quickbooks

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/builtbyrobben/quickbooks-cli/internal/api"
)

const (
	defaultBaseURL  = "https://quickbooks.api.intuit.com"
	minorVersion    = "75"
	tokenURL        = "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer" //nolint:gosec // URL, not a credential
	authURL         = "https://appcenter.intuit.com/connect/oauth2"
	defaultRedirect = "https://builtbyrobben.com/callback/"
	defaultScopes   = "com.intuit.quickbooks.accounting openid profile email"
)

var (
	errIDRequired         = errors.New("id is required")
	errCustomerIDRequired = errors.New("customer ID is required")
	errItemIDRequired     = errors.New("item ID is required")
	errAmountRequired     = errors.New("amount is required")
	errInvoiceIDRequired  = errors.New("invoice ID is required")
	errNameRequired       = errors.New("name is required")
	errSyncTokenRequired  = errors.New("sync token is required")
)

// Client wraps the API client with QuickBooks-specific methods.
type Client struct {
	*api.Client
	realmID string
}

// NewClient creates a new QuickBooks API client.
func NewClient(accessToken, realmID string, opts ...api.ClientOption) *Client {
	opts = append([]api.ClientOption{
		api.WithBaseURL(defaultBaseURL),
		api.WithUserAgent("quickbooks-cli/1.0"),
	}, opts...)

	return &Client{
		Client:  api.NewClient(accessToken, opts...),
		realmID: realmID,
	}
}

// NewClientWithOAuth creates a QuickBooks client with OAuth auto-refresh.
func NewClientWithOAuth(accessToken, realmID, clientID, clientSecret, refreshToken string, store api.TokenStore) *Client {
	return &Client{
		Client: api.NewClient(accessToken,
			api.WithBaseURL(defaultBaseURL),
			api.WithUserAgent("quickbooks-cli/1.0"),
			api.WithOAuth(clientID, clientSecret, refreshToken, tokenURL),
			api.WithTokenStore(store),
		),
		realmID: realmID,
	}
}

// TokenURL returns the OAuth token endpoint URL.
func TokenURL() string {
	return tokenURL
}

// AuthURL returns the OAuth authorization URL.
func AuthURL() string {
	return authURL
}

// DefaultRedirectURI returns the default OAuth redirect URI.
func DefaultRedirectURI() string {
	return defaultRedirect
}

// DefaultScopes returns the default OAuth scopes.
func DefaultScopes() string {
	return defaultScopes
}

func (c *Client) companyPath(resource string) string {
	return fmt.Sprintf("/v3/company/%s/%s", c.realmID, resource)
}

func (c *Client) queryPath() string {
	return fmt.Sprintf("/v3/company/%s/query", c.realmID)
}

// --- Invoice types ---

// Invoice represents a QuickBooks invoice.
type Invoice struct {
	ID          string        `json:"Id"`
	SyncToken   string        `json:"SyncToken"`
	DocNumber   string        `json:"DocNumber"`
	TxnDate     string        `json:"TxnDate"`
	DueDate     string        `json:"DueDate"`
	TotalAmt    float64       `json:"TotalAmt"`
	Balance     float64       `json:"Balance"`
	CustomerRef Ref           `json:"CustomerRef"`
	Line        []InvoiceLine `json:"Line"`
	BillEmail   *EmailAddr    `json:"BillEmail,omitempty"`
	EmailStatus string        `json:"EmailStatus"`
	PrintStatus string        `json:"PrintStatus"`
	PrivateNote string        `json:"PrivateNote,omitempty"`
	MetaData    *MetaData     `json:"MetaData,omitempty"`
}

// InvoiceLine represents a line item on an invoice.
type InvoiceLine struct {
	ID                  string               `json:"Id,omitempty"`
	LineNum             int                  `json:"LineNum,omitempty"`
	Amount              float64              `json:"Amount"`
	DetailType          string               `json:"DetailType"`
	SalesItemLineDetail *SalesItemLineDetail `json:"SalesItemLineDetail,omitempty"`
	Description         string               `json:"Description,omitempty"`
}

// SalesItemLineDetail provides details for a sales item line.
type SalesItemLineDetail struct {
	ItemRef   Ref     `json:"ItemRef"`
	Qty       float64 `json:"Qty"`
	UnitPrice float64 `json:"UnitPrice"`
}

// Ref is a QuickBooks reference with value and name.
type Ref struct {
	Value string `json:"value"`
	Name  string `json:"name,omitempty"`
}

// EmailAddr holds an email address.
type EmailAddr struct {
	Address string `json:"Address"`
}

// MetaData contains create/update timestamps.
type MetaData struct {
	CreateTime      string `json:"CreateTime"`
	LastUpdatedTime string `json:"LastUpdatedTime"`
}

// --- Bill types ---

// Bill represents a QuickBooks bill.
type Bill struct {
	ID        string     `json:"Id"`
	SyncToken string     `json:"SyncToken"`
	DocNumber string     `json:"DocNumber"`
	TxnDate   string     `json:"TxnDate"`
	DueDate   string     `json:"DueDate"`
	TotalAmt  float64    `json:"TotalAmt"`
	Balance   float64    `json:"Balance"`
	VendorRef Ref        `json:"VendorRef"`
	Line      []BillLine `json:"Line"`
	MetaData  *MetaData  `json:"MetaData,omitempty"`
}

// BillLine represents a line item on a bill.
type BillLine struct {
	ID         string  `json:"Id,omitempty"`
	Amount     float64 `json:"Amount"`
	DetailType string  `json:"DetailType"`
}

// --- Payment types ---

// Payment represents a QuickBooks payment.
type Payment struct {
	ID          string        `json:"Id"`
	SyncToken   string        `json:"SyncToken"`
	TotalAmt    float64       `json:"TotalAmt"`
	CustomerRef Ref           `json:"CustomerRef"`
	TxnDate     string        `json:"TxnDate"`
	Line        []PaymentLine `json:"Line,omitempty"`
	MetaData    *MetaData     `json:"MetaData,omitempty"`
}

// PaymentLine represents a line on a payment.
type PaymentLine struct {
	Amount    float64     `json:"Amount"`
	LinkedTxn []LinkedTxn `json:"LinkedTxn,omitempty"`
}

// --- Purchase / expense types ---

// Purchase represents a QuickBooks purchase, including expenses, checks, and credit card charges.
type Purchase struct {
	ID          string         `json:"Id"`
	SyncToken   string         `json:"SyncToken"`
	DocNumber   string         `json:"DocNumber"`
	TxnDate     string         `json:"TxnDate"`
	TotalAmt    float64        `json:"TotalAmt"`
	PaymentType string         `json:"PaymentType"`
	AccountRef  Ref            `json:"AccountRef"`
	EntityRef   Ref            `json:"EntityRef"`
	PrivateNote string         `json:"PrivateNote,omitempty"`
	Line        []PurchaseLine `json:"Line,omitempty"`
	MetaData    *MetaData      `json:"MetaData,omitempty"`
}

// PurchaseLine represents a line item on a purchase.
type PurchaseLine struct {
	ID          string  `json:"Id,omitempty"`
	Amount      float64 `json:"Amount"`
	DetailType  string  `json:"DetailType"`
	Description string  `json:"Description,omitempty"`
}

// --- Bill payment types ---

// BillPayment represents a QuickBooks vendor bill payment.
type BillPayment struct {
	ID          string            `json:"Id"`
	SyncToken   string            `json:"SyncToken"`
	DocNumber   string            `json:"DocNumber"`
	TxnDate     string            `json:"TxnDate"`
	TotalAmt    float64           `json:"TotalAmt"`
	PayType     string            `json:"PayType"`
	VendorRef   Ref               `json:"VendorRef"`
	Line        []BillPaymentLine `json:"Line,omitempty"`
	PrivateNote string            `json:"PrivateNote,omitempty"`
	MetaData    *MetaData         `json:"MetaData,omitempty"`
}

// BillPaymentLine represents a line on a bill payment.
type BillPaymentLine struct {
	Amount    float64     `json:"Amount"`
	LinkedTxn []LinkedTxn `json:"LinkedTxn,omitempty"`
}

// --- Journal entry types ---

// JournalEntry represents a QuickBooks general journal entry.
type JournalEntry struct {
	ID          string             `json:"Id"`
	SyncToken   string             `json:"SyncToken"`
	DocNumber   string             `json:"DocNumber"`
	TxnDate     string             `json:"TxnDate"`
	PrivateNote string             `json:"PrivateNote,omitempty"`
	Adjustment  bool               `json:"Adjustment"`
	Line        []JournalEntryLine `json:"Line,omitempty"`
	MetaData    *MetaData          `json:"MetaData,omitempty"`
}

// JournalEntryLine represents a line on a journal entry.
type JournalEntryLine struct {
	ID                     string                  `json:"Id,omitempty"`
	Amount                 float64                 `json:"Amount"`
	DetailType             string                  `json:"DetailType"`
	Description            string                  `json:"Description,omitempty"`
	JournalEntryLineDetail *JournalEntryLineDetail `json:"JournalEntryLineDetail,omitempty"`
}

// JournalEntryLineDetail provides posting details for a journal entry line.
type JournalEntryLineDetail struct {
	PostingType string `json:"PostingType"`
	AccountRef  Ref    `json:"AccountRef"`
	Entity      *Ref   `json:"Entity,omitempty"`
}

// LinkedTxn links a payment to a transaction.
type LinkedTxn struct {
	TxnID   string `json:"TxnId"`
	TxnType string `json:"TxnType"`
}

// --- Customer types ---

// Customer represents a QuickBooks customer.
type Customer struct {
	ID               string     `json:"Id"`
	SyncToken        string     `json:"SyncToken"`
	DisplayName      string     `json:"DisplayName"`
	CompanyName      string     `json:"CompanyName,omitempty"`
	GivenName        string     `json:"GivenName,omitempty"`
	FamilyName       string     `json:"FamilyName,omitempty"`
	PrimaryPhone     *Phone     `json:"PrimaryPhone,omitempty"`
	PrimaryEmailAddr *EmailAddr `json:"PrimaryEmailAddr,omitempty"`
	Balance          float64    `json:"Balance"`
	Active           bool       `json:"Active"`
	MetaData         *MetaData  `json:"MetaData,omitempty"`
}

// Phone holds a phone number.
type Phone struct {
	FreeFormNumber string `json:"FreeFormNumber"`
}

// --- Vendor types ---

// Vendor represents a QuickBooks vendor.
type Vendor struct {
	ID          string    `json:"Id"`
	SyncToken   string    `json:"SyncToken"`
	DisplayName string    `json:"DisplayName"`
	CompanyName string    `json:"CompanyName,omitempty"`
	Balance     float64   `json:"Balance"`
	Active      bool      `json:"Active"`
	MetaData    *MetaData `json:"MetaData,omitempty"`
}

// --- Item types ---

// Item represents a QuickBooks item (product/service).
type Item struct {
	ID        string    `json:"Id"`
	SyncToken string    `json:"SyncToken"`
	Name      string    `json:"Name"`
	Type      string    `json:"Type"`
	UnitPrice float64   `json:"UnitPrice"`
	Active    bool      `json:"Active"`
	MetaData  *MetaData `json:"MetaData,omitempty"`
}

// --- Report types ---

// Report represents a QuickBooks report.
type Report struct {
	Header  ReportHeader  `json:"Header"`
	Columns ReportColumns `json:"Columns"`
	Rows    ReportRows    `json:"Rows"`
}

// ReportHeader contains report metadata.
type ReportHeader struct {
	ReportName  string `json:"ReportName"`
	ReportBasis string `json:"ReportBasis"`
	StartPeriod string `json:"StartPeriod"`
	EndPeriod   string `json:"EndPeriod"`
	Currency    string `json:"Currency"`
	Time        string `json:"Time"`
}

// ReportColumns describes the report columns.
type ReportColumns struct {
	Column []ReportColumn `json:"Column"`
}

// ReportColumn is a single report column.
type ReportColumn struct {
	ColTitle string `json:"ColTitle"`
	ColType  string `json:"ColType"`
}

// ReportRows contains the report data rows.
type ReportRows struct {
	Row []ReportRow `json:"Row"`
}

// ReportRow is a single report row.
type ReportRow struct {
	Header  *RowHeader  `json:"Header,omitempty"`
	Rows    *ReportRows `json:"Rows,omitempty"`
	Summary *RowSummary `json:"Summary,omitempty"`
	ColData []ColData   `json:"ColData,omitempty"`
	Type    string      `json:"type,omitempty"`
	Group   string      `json:"group,omitempty"`
}

// RowHeader is the header for a report section.
type RowHeader struct {
	ColData []ColData `json:"ColData"`
}

// RowSummary is the summary for a report section.
type RowSummary struct {
	ColData []ColData `json:"ColData"`
}

// ColData is a single cell value.
type ColData struct {
	Value string `json:"value"`
	ID    string `json:"id,omitempty"`
}

// --- Query response wrappers ---

// QueryResponse wraps the QuickBooks query response.
type QueryResponse struct {
	Invoices       []Invoice      `json:"Invoice,omitempty"`
	Bills          []Bill         `json:"Bill,omitempty"`
	Payments       []Payment      `json:"Payment,omitempty"`
	Purchases      []Purchase     `json:"Purchase,omitempty"`
	BillPayments   []BillPayment  `json:"BillPayment,omitempty"`
	JournalEntries []JournalEntry `json:"JournalEntry,omitempty"`
	Customers      []Customer     `json:"Customer,omitempty"`
	Vendors        []Vendor       `json:"Vendor,omitempty"`
	Items          []Item         `json:"Item,omitempty"`
	StartPosition  int            `json:"startPosition"`
	MaxResults     int            `json:"maxResults"`
	TotalCount     int            `json:"totalCount"`
}

type queryEnvelope struct {
	QueryResponse QueryResponse `json:"QueryResponse"`
}

type invoiceEnvelope struct {
	Invoice Invoice `json:"Invoice"`
}

type billEnvelope struct {
	Bill Bill `json:"Bill"`
}

type paymentEnvelope struct {
	Payment Payment `json:"Payment"`
}

type purchaseEnvelope struct {
	Purchase Purchase `json:"Purchase"`
}

type billPaymentEnvelope struct {
	BillPayment BillPayment `json:"BillPayment"`
}

type journalEntryEnvelope struct {
	JournalEntry JournalEntry `json:"JournalEntry"`
}

type customerEnvelope struct {
	Customer Customer `json:"Customer"`
}

// --- Service accessors ---

// Invoices returns the invoices service.
func (c *Client) Invoices() *InvoicesService {
	return &InvoicesService{client: c}
}

// Bills returns the bills service.
func (c *Client) Bills() *BillsService {
	return &BillsService{client: c}
}

// Payments returns the payments service.
func (c *Client) Payments() *PaymentsService {
	return &PaymentsService{client: c}
}

// Purchases returns the purchases service.
func (c *Client) Purchases() *PurchasesService {
	return &PurchasesService{client: c}
}

// BillPayments returns the bill payments service.
func (c *Client) BillPayments() *BillPaymentsService {
	return &BillPaymentsService{client: c}
}

// JournalEntries returns the journal entries service.
func (c *Client) JournalEntries() *JournalEntriesService {
	return &JournalEntriesService{client: c}
}

// Customers returns the customers service.
func (c *Client) Customers() *CustomersService {
	return &CustomersService{client: c}
}

// Vendors returns the vendors service.
func (c *Client) Vendors() *VendorsService {
	return &VendorsService{client: c}
}

// Items returns the items service.
func (c *Client) Items() *ItemsService {
	return &ItemsService{client: c}
}

// Reports returns the reports service.
func (c *Client) Reports() *ReportsService {
	return &ReportsService{client: c}
}

// --- Invoices Service ---

// InvoicesService handles invoice operations.
type InvoicesService struct {
	client *Client
}

// List queries invoices with an optional SQL-like query.
func (s *InvoicesService) List(ctx context.Context, query string, page, pageSize int) (*QueryResponse, error) {
	if query == "" {
		query = "SELECT * FROM Invoice"
	}

	if page > 1 {
		startPos := (page-1)*pageSize + 1
		query += fmt.Sprintf(" STARTPOSITION %d MAXRESULTS %d", startPos, pageSize)
	} else if pageSize > 0 {
		query += fmt.Sprintf(" MAXRESULTS %d", pageSize)
	}

	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// Get retrieves a single invoice by ID.
func (s *InvoicesService) Get(ctx context.Context, id string) (*Invoice, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("invoice/"+id) + "?minorversion=" + minorVersion

	var envelope invoiceEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	return &envelope.Invoice, nil
}

// CreateInvoiceRequest holds the data for creating an invoice.
type CreateInvoiceRequest struct {
	CustomerID string
	ItemID     string
	Qty        float64
	Rate       float64
	DueDate    string
}

// Create creates a new invoice.
func (s *InvoicesService) Create(ctx context.Context, req CreateInvoiceRequest) (*Invoice, error) {
	if req.CustomerID == "" {
		return nil, errCustomerIDRequired
	}

	if req.ItemID == "" {
		return nil, errItemIDRequired
	}

	body := map[string]any{
		"CustomerRef": map[string]string{"value": req.CustomerID},
		"Line": []map[string]any{
			{
				"Amount":     req.Qty * req.Rate,
				"DetailType": "SalesItemLineDetail",
				"SalesItemLineDetail": map[string]any{
					"ItemRef":   map[string]string{"value": req.ItemID},
					"Qty":       req.Qty,
					"UnitPrice": req.Rate,
				},
			},
		},
	}

	if req.DueDate != "" {
		body["DueDate"] = req.DueDate
	}

	path := s.client.companyPath("invoice") + "?minorversion=" + minorVersion

	var envelope invoiceEnvelope
	if err := s.client.Post(ctx, path, body, &envelope); err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	return &envelope.Invoice, nil
}

// Send sends an invoice by email.
func (s *InvoicesService) Send(ctx context.Context, id, email string) (*Invoice, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("invoice/"+id+"/send") + "?minorversion=" + minorVersion

	if email != "" {
		path += "&sendTo=" + url.QueryEscape(email)
	}

	var envelope invoiceEnvelope
	if err := s.client.Post(ctx, path, nil, &envelope); err != nil {
		return nil, fmt.Errorf("send invoice: %w", err)
	}

	return &envelope.Invoice, nil
}

// Void voids an invoice.
func (s *InvoicesService) Void(ctx context.Context, id, syncToken string) (*Invoice, error) {
	if id == "" {
		return nil, errIDRequired
	}

	if syncToken == "" {
		return nil, errSyncTokenRequired
	}

	body := map[string]any{
		"Id":        id,
		"SyncToken": syncToken,
		"sparse":    true,
	}

	path := s.client.companyPath("invoice") + "?operation=void&minorversion=" + minorVersion

	var envelope invoiceEnvelope
	if err := s.client.Post(ctx, path, body, &envelope); err != nil {
		return nil, fmt.Errorf("void invoice: %w", err)
	}

	return &envelope.Invoice, nil
}

// --- Bills Service ---

// BillsService handles bill operations.
type BillsService struct {
	client *Client
}

// List queries bills.
func (s *BillsService) List(ctx context.Context, query string) (*QueryResponse, error) {
	if query == "" {
		query = "SELECT * FROM Bill"
	}

	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list bills: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// Get retrieves a single bill by ID.
func (s *BillsService) Get(ctx context.Context, id string) (*Bill, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("bill/"+id) + "?minorversion=" + minorVersion

	var envelope billEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("get bill: %w", err)
	}

	return &envelope.Bill, nil
}

// --- Payments Service ---

// PaymentsService handles payment operations.
type PaymentsService struct {
	client *Client
}

// List queries payments.
func (s *PaymentsService) List(ctx context.Context, query string) (*QueryResponse, error) {
	if query == "" {
		query = "SELECT * FROM Payment"
	}

	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// CreatePaymentRequest holds the data for creating a payment.
type CreatePaymentRequest struct {
	CustomerID string
	Amount     float64
	InvoiceID  string
}

// Create creates a new payment.
func (s *PaymentsService) Create(ctx context.Context, req CreatePaymentRequest) (*Payment, error) {
	if req.CustomerID == "" {
		return nil, errCustomerIDRequired
	}

	if req.Amount <= 0 {
		return nil, errAmountRequired
	}

	if req.InvoiceID == "" {
		return nil, errInvoiceIDRequired
	}

	body := map[string]any{
		"CustomerRef": map[string]string{"value": req.CustomerID},
		"TotalAmt":    req.Amount,
		"Line": []map[string]any{
			{
				"Amount": req.Amount,
				"LinkedTxn": []map[string]string{
					{
						"TxnId":   req.InvoiceID,
						"TxnType": "Invoice",
					},
				},
			},
		},
	}

	path := s.client.companyPath("payment") + "?minorversion=" + minorVersion

	var envelope paymentEnvelope
	if err := s.client.Post(ctx, path, body, &envelope); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	return &envelope.Payment, nil
}

// --- Purchases Service ---

// PurchasesService handles purchase operations.
type PurchasesService struct {
	client *Client
}

// List queries purchases.
func (s *PurchasesService) List(ctx context.Context, query string) (*QueryResponse, error) {
	if query == "" {
		query = "SELECT * FROM Purchase"
	}

	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list purchases: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// Get retrieves a single purchase by ID.
func (s *PurchasesService) Get(ctx context.Context, id string) (*Purchase, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("purchase/"+id) + "?minorversion=" + minorVersion

	var envelope purchaseEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("get purchase: %w", err)
	}

	return &envelope.Purchase, nil
}

// --- Bill Payments Service ---

// BillPaymentsService handles vendor bill payment operations.
type BillPaymentsService struct {
	client *Client
}

// List queries bill payments.
func (s *BillPaymentsService) List(ctx context.Context, query string) (*QueryResponse, error) {
	if query == "" {
		query = "SELECT * FROM BillPayment"
	}

	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list bill payments: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// Get retrieves a single bill payment by ID.
func (s *BillPaymentsService) Get(ctx context.Context, id string) (*BillPayment, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("billpayment/"+id) + "?minorversion=" + minorVersion

	var envelope billPaymentEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("get bill payment: %w", err)
	}

	return &envelope.BillPayment, nil
}

// --- Journal Entries Service ---

// JournalEntriesService handles journal entry operations.
type JournalEntriesService struct {
	client *Client
}

// List queries journal entries.
func (s *JournalEntriesService) List(ctx context.Context, query string) (*QueryResponse, error) {
	if query == "" {
		query = "SELECT * FROM JournalEntry"
	}

	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// Get retrieves a single journal entry by ID.
func (s *JournalEntriesService) Get(ctx context.Context, id string) (*JournalEntry, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("journalentry/"+id) + "?minorversion=" + minorVersion

	var envelope journalEntryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("get journal entry: %w", err)
	}

	return &envelope.JournalEntry, nil
}

// --- Customers Service ---

// CustomersService handles customer operations.
type CustomersService struct {
	client *Client
}

// List queries customers.
func (s *CustomersService) List(ctx context.Context) (*QueryResponse, error) {
	query := "SELECT * FROM Customer"
	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// Get retrieves a single customer by ID.
func (s *CustomersService) Get(ctx context.Context, id string) (*Customer, error) {
	if id == "" {
		return nil, errIDRequired
	}

	path := s.client.companyPath("customer/"+id) + "?minorversion=" + minorVersion

	var envelope customerEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	return &envelope.Customer, nil
}

// CreateCustomerRequest holds the data for creating a customer.
type CreateCustomerRequest struct {
	DisplayName string
	Email       string
	Phone       string
}

// Create creates a new customer.
func (s *CustomersService) Create(ctx context.Context, req CreateCustomerRequest) (*Customer, error) {
	if req.DisplayName == "" {
		return nil, errNameRequired
	}

	body := map[string]any{
		"DisplayName": req.DisplayName,
	}

	if req.Email != "" {
		body["PrimaryEmailAddr"] = map[string]string{"Address": req.Email}
	}

	if req.Phone != "" {
		body["PrimaryPhone"] = map[string]string{"FreeFormNumber": req.Phone}
	}

	path := s.client.companyPath("customer") + "?minorversion=" + minorVersion

	var envelope customerEnvelope
	if err := s.client.Post(ctx, path, body, &envelope); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return &envelope.Customer, nil
}

// --- Vendors Service ---

// VendorsService handles vendor operations.
type VendorsService struct {
	client *Client
}

// List queries vendors.
func (s *VendorsService) List(ctx context.Context) (*QueryResponse, error) {
	query := "SELECT * FROM Vendor"
	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list vendors: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// --- Items Service ---

// ItemsService handles item/service operations.
type ItemsService struct {
	client *Client
}

// List queries items.
func (s *ItemsService) List(ctx context.Context) (*QueryResponse, error) {
	query := "SELECT * FROM Item"
	path := s.client.queryPath() + "?query=" + url.QueryEscape(query) + "&minorversion=" + minorVersion

	var envelope queryEnvelope
	if err := s.client.Get(ctx, path, &envelope); err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	return &envelope.QueryResponse, nil
}

// --- Reports Service ---

// ReportsService handles report operations.
type ReportsService struct {
	client *Client
}

// ProfitAndLoss retrieves a P&L report.
func (s *ReportsService) ProfitAndLoss(ctx context.Context, from, to, method string) (*Report, error) {
	params := url.Values{"minorversion": {minorVersion}}

	if from != "" {
		params.Set("start_date", from)
	}

	if to != "" {
		params.Set("end_date", to)
	}

	if method != "" {
		params.Set("accounting_method", method)
	}

	path := s.client.companyPath("reports/ProfitAndLoss") + "?" + params.Encode()

	var report Report
	if err := s.client.Get(ctx, path, &report); err != nil {
		return nil, fmt.Errorf("get profit and loss report: %w", err)
	}

	return &report, nil
}

// BalanceSheet retrieves a balance sheet report.
func (s *ReportsService) BalanceSheet(ctx context.Context, date string) (*Report, error) {
	params := url.Values{"minorversion": {minorVersion}}

	if date != "" {
		params.Set("end_date", date)
	}

	path := s.client.companyPath("reports/BalanceSheet") + "?" + params.Encode()

	var report Report
	if err := s.client.Get(ctx, path, &report); err != nil {
		return nil, fmt.Errorf("get balance sheet report: %w", err)
	}

	return &report, nil
}

// GeneralLedger retrieves the general ledger report.
func (s *ReportsService) GeneralLedger(ctx context.Context, from, to, method string) (*Report, error) {
	params := url.Values{"minorversion": {minorVersion}}

	if from != "" {
		params.Set("start_date", from)
	}

	if to != "" {
		params.Set("end_date", to)
	}

	if method != "" {
		params.Set("accounting_method", method)
	}

	path := s.client.companyPath("reports/GeneralLedger") + "?" + params.Encode()

	var report Report
	if err := s.client.Get(ctx, path, &report); err != nil {
		return nil, fmt.Errorf("get general ledger report: %w", err)
	}

	return &report, nil
}

// TransactionList retrieves the transaction list report.
func (s *ReportsService) TransactionList(ctx context.Context, from, to string) (*Report, error) {
	params := url.Values{"minorversion": {minorVersion}}

	if from != "" {
		params.Set("start_date", from)
	}

	if to != "" {
		params.Set("end_date", to)
	}

	path := s.client.companyPath("reports/TransactionList") + "?" + params.Encode()

	var report Report
	if err := s.client.Get(ctx, path, &report); err != nil {
		return nil, fmt.Errorf("get transaction list report: %w", err)
	}

	return &report, nil
}
