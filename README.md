# Saldo CLI

Agent-friendly command line client for Saldo. The CLI connects to the Saldo
backend over GraphQL and owns all session/token handling internally.

## Install for development

```bash
go mod tidy
go build -o saldo ./cmd/saldo
```

Or use the Makefile:

```bash
make build
```

## Production build

Generate a stripped static binary from the repo:

```bash
make build-prod
```

The compiled CLI is written to `dist/saldo`.

## Configure

For humans, the CLI stores its private session at `~/.config/saldo/session.json`.
For agents, give each agent a unique session path:

```bash
export SALDO_SESSION=/tmp/saldo-agent-a/session.json
export SALDO_API_URL=https://saldo.example.com/graphql/
```

The agent may set `SALDO_SESSION`, but should never read or edit the session
file. The CLI reads/writes tokens itself.

You can also persist the API URL:

```bash
saldo config set api-url https://saldo.example.com/graphql/
saldo config get --json
```

## Auth

```bash
saldo auth login --email user@example.com
saldo auth whoami --json
saldo auth logout --json
```

For non-interactive login:

```bash
SALDO_PASSWORD='secret' saldo auth login --email user@example.com --json
```

## Agent Core Commands

```bash
saldo accounts list --json
saldo accounts get 1 --json

saldo categories list --query food --type EXPENSE --json
saldo tags list --query online --json

saldo transactions list --account-id 1 --from 2026-05-01T00:00:00Z --to 2026-05-31T23:59:59Z --json
saldo transactions create --account-id 1 --amount 25.50 --kind EXPENSE --currency PEN --date 2026-05-03T12:00:00Z --description "Lunch" --json

saldo snapshot ai --from 2026-05-01 --to 2026-05-31 --section transactions --json
```

## Receipt Draft Flow

The agent does OCR/vision and sends structured JSON to the CLI:

```json
{
  "account": "AccountA",
  "merchant": "Metro",
  "date": "2026-05-03",
  "items": [
    {"name": "Pan", "amount": 4.5},
    {"name": "Leche", "amount": 6.2}
  ],
  "total": 10.7,
  "currency": "PEN",
  "category": "Alimentación",
  "tags": ["Efectivo"]
}
```

Then:

```bash
saldo transactions draft --file receipt.json --json
```

The CLI returns a normalized preview. After the user confirms or edits it, the
agent calls `saldo transactions create ... --json` with the final values.
