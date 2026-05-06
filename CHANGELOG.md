# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Read-only transaction visibility commands:
  - `purchases list|get`
  - `checks list|get`
  - `credit-card-charges list|get`
  - `bill-payments list|get`
  - `journal-entries list|get`
  - `sales-receipts list|get`
- `reports general-ledger` and `reports transaction-list` commands for broader transaction review.

### Changed
- Purchase-family JSON list output now returns the full query response envelope, matching other list commands.
- Plain TSV list/get output for purchase, bill payment, and sales receipt views now keeps `TOTAL` as the final column.

### Fixed
- `checks get` and `credit-card-charges get` now validate the returned purchase `PaymentType` before formatting results.

## [0.1.0] - 2024-01-01

### Added
- Template repository for CLI tools
- Keyring-backed credential storage with file fallback
- Auth commands (set-key, status, remove)
- Output formatting (JSON/plain)
- Cross-platform build support (macOS/Linux/Windows)
- GitHub Actions CI/CD
- GoReleaser configuration
