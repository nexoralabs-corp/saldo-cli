# Saldo CLI Command Reference

Use `--json` on all commands intended for agent parsing.

## Environment

```bash
export SALDO_SESSION=/tmp/saldo-agent/session.json
export SALDO_API_URL=https://saldo.example.com/graphql/
```

`SALDO_SESSION` isolates agents. Do not read or edit its contents.

## Auth

```bash
saldo auth login --email user@example.com --json
saldo auth whoami --json
saldo auth logout --json
```

Non-interactive login:

```bash
SALDO_PASSWORD='secret' saldo auth login --email user@example.com --json
```

## Config

```bash
saldo config set api-url https://saldo.example.com/graphql/ --json
saldo config get --json
```

## Accounts

```bash
saldo accounts list --json
saldo accounts get 1 --json
```

## Lookup

```bash
saldo categories list --query food --type EXPENSE --json
saldo categories list --query salary --type INCOME --json
saldo tags list --query online --json
```

Category type values are `INCOME`, `EXPENSE`, or `BOTH`.

## Transactions

```bash
saldo transactions list --account-id 1 --from 2026-05-01T00:00:00Z --to 2026-05-31T23:59:59Z --json
```

```bash
saldo transactions create \
  --account-id 1 \
  --amount 25.50 \
  --kind EXPENSE \
  --currency PEN \
  --date 2026-05-03T12:00:00Z \
  --category-id 5 \
  --description "Lunch" \
  --tag "Online" \
  --json
```

`--kind` must be `INCOME` or `EXPENSE`.

## Drafts

From a file:

```bash
saldo transactions draft --file receipt.json --json
```

From stdin:

```bash
printf '%s' "$DRAFT_JSON" | saldo transactions draft --file - --json
```

Commit by calling `saldo transactions create` with the user-confirmed final values. The draft command only previews; it does not write to the backend.

## AI Snapshot

```bash
saldo snapshot ai --from 2026-05-01 --to 2026-05-31 --section transactions --json
```

Valid repeated `--section` values include `net-worth`, `transactions`, `subscriptions`, `loans`, `credit-cards`, and `budgets`.

