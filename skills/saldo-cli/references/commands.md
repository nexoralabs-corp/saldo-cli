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
saldo auth profiles --json
saldo auth logout --json
```

Non-interactive login:

```bash
SALDO_PASSWORD='secret' saldo auth login --email user@example.com --json
```

Multiple login profiles can share the same session file. The first saved profile
is the default; select another by email:

```bash
saldo --profile second@example.com auth whoami --json
saldo --account second@example.com accounts list --json
saldo --profile second@example.com auth logout --json
saldo auth logout --all --json
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
saldo accounts create --name "Interbank" --type BANK --currency PEN --json
saldo accounts update 1 --name "Interbank Ahorros" --json
saldo accounts delete 1 --json
```

## Lookup

```bash
saldo categories list --query food --type EXPENSE --json
saldo categories list --query salary --type INCOME --json
saldo tags list --query online --json
saldo categories create --name "Servicios" --type EXPENSE --json
saldo categories create --name "Internet" --type EXPENSE --parent-id 5 --json
saldo tags create --name "UTP" --json
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

## Cards, loans, and subscriptions

```bash
saldo credit-cards create --name "CMR - Falabella" --issuer FALABELLA --currency PEN --credit-limit 0 --closing-day 0 --due-day 0 --json
saldo credit-cards payment --card-id 3 --from-account-id 1 --amount 100 --idempotency-key cmr-2026-07 --json
saldo loans create --name "MAF – Agya" --lender MAF --currency PEN --outstanding-balance 761.80 --json
saldo loans payment --loan-id 1 --from-account-id 1 --amount 100 --date 2026-07-12 --idempotency-key maf-2026-07 --json
saldo subscriptions create --name "Movistar Internet" --amount 110 --currency PEN --frequency MONTHLY --due-day 15 --category-id 5 --json
saldo subscriptions upcoming --days 30 --json
saldo budgets create --category-id 5 --monthly-limit 500 --currency PEN --json
saldo budgets list --json
```

## Transfers and bulk import

```bash
saldo transactions transfer --from-account-id 1 --to-account-id 2 --amount 100 --idempotency-key transfer-2026-07-12 --json
saldo import registrations --file gastos.json --dry-run --json
saldo import registrations --file gastos.json --json
```

Every imported registration requires its own `idempotencyKey`. Treat a dry-run response with `valid: false` as a hard stop.

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
