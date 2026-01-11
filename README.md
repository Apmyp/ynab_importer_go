# YNAB Importer Go

Import bank transaction SMS from macOS Messages to [YNAB](https://www.ynab.com/).

Supports Moldovan banks: MAIB and Eximbank.

## Requirements

- macOS (reads from Messages app database)
- YNAB account with API key

## Granting Access to Messages Database

The Messages database (`chat.db`) is a protected file in macOS. You need to grant access before the app can read your SMS messages.

### Option 1: Full Disk Access (recommended for automated sync)

Grant your terminal full disk access:

1. Open **System Preferences** → **Security & Privacy**
2. Select the **Privacy** tab
3. Select **Full Disk Access** from the left menu
4. Click the lock icon and authenticate
5. Click **+** and add your terminal app (Terminal, iTerm, etc.)
6. Restart your terminal

After this, the app can read `chat.db` from its default location.

### Option 2: Copy the Database (more secure, manual only)

Copy the database file to an unprotected location:

1. Open Finder
2. Navigate to `~/Library/Messages`
3. Right-click `chat.db` → **Copy "chat.db"**
4. Paste it to your home folder or another location
5. Update `db_path` in your config to point to the copy

Note: This copy won't include new messages. You need to repeat this process each time.

## Installation

```bash
go build -o ynab_importer_go .
```

## Configuration

Create `config.json` in the same directory:

```json
{
  "senders": ["102", "EXIMBANK"],
  "db_path": "~/Library/Messages/chat.db",
  "default_currency": "MDL",
  "ynab": {
    "start_date": "2025-01-01"
  }
}
```

| Field | Description |
|-------|-------------|
| `senders` | SMS sender IDs to track |
| `db_path` | Path to macOS Messages database |
| `default_currency` | Target currency for conversion (default: MDL) |
| `ynab.budget_id` | Your YNAB budget UUID (auto-fetched if not set) |
| `ynab.start_date` | Only sync transactions after this date |
| `ynab.accounts` | Map card last 4 digits to YNAB account IDs (auto-created) |

Set your YNAB API key:

```bash
export YNAB_API_KEY="your-api-key"
```

## Commands

### Default Command (Sync to YNAB)

Running without arguments syncs transactions to YNAB:

```bash
./ynab_importer_go
```

Parses all SMS messages, converts currencies using BNM exchange rates, and syncs to YNAB.

Features:
- Auto-fetches budget ID from YNAB API if not configured
- Auto-creates YNAB accounts for new cards
- Skips already synced transactions (deduplication via import ID)
- Skips declined transactions
- Converts foreign currency to MDL using National Bank of Moldova rates

### Find Missing Templates

```bash
./ynab_importer_go missing_templates
```

Shows SMS messages that don't match any parsing template. Useful for debugging or adding new bank formats.

### Install System Service

```bash
./ynab_importer_go system_install
```

Installs a macOS launchd service that syncs every hour.

Logs are saved to:
- `ynab_sync.log` - standard output
- `ynab_sync_error.log` - errors

### Uninstall System Service

```bash
./ynab_importer_go system_uninstall
```

Removes the hourly sync service.

## Options

| Option | Description |
|--------|-------------|
| `--config <path>` | Use custom config file (default: `config.json`) |
| `--data-file <path>` | Use custom data file for cache and sync records |

Example:

```bash
./ynab_importer_go --config ~/my-config.json --data-file ~/my-data.json
```

## How It Works

1. Reads SMS messages from macOS Messages database
2. Parses transactions using regex templates for MAIB and Eximbank formats
3. Fetches exchange rates from National Bank of Moldova (cached locally)
4. Maps card numbers to YNAB accounts
5. Creates transactions in YNAB with unique import IDs

## Supported Message Types

- MAIB transaction notifications (sender "102")
- Eximbank account transfers
- Card debits (Debitare)
- Successful transactions (Tranzactie reusita)
- Card top-ups (Suplinire)

Non-transaction messages (OTP codes, marketing, etc.) are ignored.

## Data Storage

Exchange rates and sync records are cached in `ynab_importer_go_data.json` (or custom path via `--data-file`).
