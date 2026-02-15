# quickbooks-cli

A CLI tool for [QuickBooks Online](https://quickbooks.intuit.com/) built with Go. Manage invoices, bills, payments, customers, vendors, items, and financial reports from the command line.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap builtbyrobben/tap
brew install quickbooks-cli
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/builtbyrobben/quickbooks-cli/releases).

### Build from Source

```bash
git clone https://github.com/builtbyrobben/quickbooks-cli.git
cd quickbooks-cli
make build
```

## Configuration

quickbooks-cli uses OAuth 2.0 to authenticate with the QuickBooks Online API. You need a QuickBooks app with a client ID and secret.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `QUICKBOOKS_CLIENT_ID` | OAuth 2.0 client ID |
| `QUICKBOOKS_CLIENT_SECRET` | OAuth 2.0 client secret |
| `QUICKBOOKS_REFRESH_TOKEN` | OAuth 2.0 refresh token |
| `QUICKBOOKS_REALM_ID` | QuickBooks company/realm ID |

### Initial Setup

```bash
# 1. Store your OAuth client credentials
quickbooks-cli auth set-credentials

# 2. Set your company/realm ID
quickbooks-cli auth set-realm 1234567890

# 3. Authenticate via OAuth 2.0 browser flow
quickbooks-cli auth login

# Check authentication status
quickbooks-cli auth status

# Remove all stored credentials
quickbooks-cli auth remove
```

## Commands

### auth -- Authentication and credentials

```bash
quickbooks-cli auth login              # OAuth 2.0 login flow
quickbooks-cli auth set-credentials    # Set client ID and secret
quickbooks-cli auth set-realm <id>     # Set company/realm ID
quickbooks-cli auth status             # Show authentication status
quickbooks-cli auth remove             # Remove all credentials
```

### invoices -- Invoice operations

```bash
# List invoices
quickbooks-cli invoices list

# Filter with SQL-like query
quickbooks-cli invoices list --query "SELECT * FROM Invoice WHERE TotalAmt > '100'"

# Paginate results
quickbooks-cli invoices list --page 2 --page-size 25

# Get invoice details
quickbooks-cli invoices get 123

# Create an invoice
quickbooks-cli invoices create --customer 42 --item 1 --qty 10 --rate 150.00 --due-date 2026-03-01

# Send invoice by email
quickbooks-cli invoices send 123

# Send to a specific email
quickbooks-cli invoices send 123 --to billing@example.com

# Void an invoice
quickbooks-cli invoices void 123 --sync-token 0
```

### bills -- Bill operations

```bash
# List bills
quickbooks-cli bills list

# Filter bills
quickbooks-cli bills list --query "SELECT * FROM Bill WHERE TotalAmt > '500'"

# Get bill details
quickbooks-cli bills get 456
```

### payments -- Payment operations

```bash
# List payments
quickbooks-cli payments list

# Filter payments
quickbooks-cli payments list --query "SELECT * FROM Payment"

# Create a payment
quickbooks-cli payments create --customer 42 --amount 500.00 --invoice 123
```

### customers -- Customer operations

```bash
# List customers
quickbooks-cli customers list

# Get customer details
quickbooks-cli customers get 42

# Create a customer
quickbooks-cli customers create --name "Acme Corp" --email billing@acme.com --phone "555-0100"
```

### vendors -- Vendor operations

```bash
# List vendors
quickbooks-cli vendors list
```

### items -- Item/service operations

```bash
# List items and services
quickbooks-cli items list
```

### reports -- Financial reports

```bash
# Profit and Loss report
quickbooks-cli reports profit-and-loss

# With date range
quickbooks-cli reports profit-and-loss --from 2026-01-01 --to 2026-12-31

# With accounting method
quickbooks-cli reports profit-and-loss --method Cash

# Balance Sheet report
quickbooks-cli reports balance-sheet

# Balance Sheet for a specific date
quickbooks-cli reports balance-sheet --date 2026-01-31
```

### version

```bash
quickbooks-cli version
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output JSON to stdout (for scripting) |
| `--plain` | Output stable TSV text (no colors) |
| `--verbose` | Enable verbose logging |
| `--force` | Skip confirmation prompts |
| `--no-input` | Never prompt; fail instead (CI mode) |
| `--color` | Color output: `auto`, `always`, or `never` |

## License

MIT
