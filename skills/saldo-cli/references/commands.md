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
saldo credit-cards create --name "CMR - Falabella" --issuer FALABELLA --currencies-file currencies.json --json
saldo credit-cards list --status active --json
saldo credit-cards get 3 --json
saldo credit-cards update 3 --status cancelled --json
saldo credit-cards currencies add 3 --currency USD --credit-limit 1500 --default-payment-account-id 2 --json
saldo credit-cards currencies update 3 --currency PEN --minimum-payment 250 --json
saldo credit-cards currencies set-default 3 --currency PEN --account-id 1 --json
saldo credit-cards archive 3 --json
saldo credit-cards reactivate 3 --json
saldo credit-cards delete 3 --json
saldo credit-cards payment --card-id 3 --currency PEN --from-account-id 2 --debit-amount 10 --applied-amount 37 --exchange-rate 3.7 --idempotency-key cmr-2026-07-usd-pen --json
saldo credit-cards balances adjust --card-id 3 --currency PEN --target-amount 1250 --balance-side debt --reason "Estado de cuenta" --idempotency-key cmr-2026-07-balance --json
saldo credit-cards balances history 3 --currency PEN --json
saldo credit-cards limits get 3 --json
saldo credit-cards limits set-shared 3 --limit 12000 --currency PEN --rate USD=3.7 --json
saldo credit-cards limits set-per-currency 3 --json
saldo credit-cards statements import --card-id 3 --currency PEN --file julio.csv --closing-date 2026-07-18 --opening-balance 900 --statement-balance 1250 --dry-run --json
saldo credit-cards statements confirm --import-id 44 --idempotency-key cmr-2026-07-statement --json
saldo credit-cards statements list --card-id 3 --currency PEN --json
saldo credit-cards statements get 55 --json
saldo credit-cards statements create --card-id 3 --currency PEN --closing-date 2026-07-18 --opening-balance 900 --statement-balance 1250 --minimum-payment 150 --total-payment 1250 --json
saldo credit-cards charges create --card-id 3 --currency PEN --name "Seguro" --type INSURANCE --next-charge-date 2026-08-18 --calculation PERCENT_OF_STATEMENT_BALANCE --percentage 0.5 --json
saldo credit-cards charges list --card-id 3 --json
saldo credit-cards charges project 8 --statement-id 55 --json
saldo credit-cards charges waive 19 --reason "Consumo mínimo cumplido" --json
saldo credit-cards charges record 19 --idempotency-key cmr-insurance-2026-08 --json
saldo credit-cards charges history 8 --json
saldo loans create --name "MAF – Agya" --lender MAF --currency PEN --outstanding-balance 761.80 --json
saldo loans list --status active --json
saldo loans get 1 --json
saldo loans update 1 --default-payment-account-id 4 --json
saldo loans archive 1 --json
saldo loans reactivate 1 --json
saldo loans delete 1 --json
saldo loans schedule get 1 --json
saldo loans schedule update 1 --file schedule.json --json
saldo loans propose-allocation --loan-id 1 --applied-amount 100 --json
saldo loans payment --loan-id 1 --from-account-id 1 --amount 100 --date 2026-07-12 --idempotency-key maf-2026-07 --json
saldo loans payment --loan-id 1 --from-account-id 2 --amount 100 --source-amount 370 --applied-amount 100 --exchange-rate 3.7 --allocations-file allocations.json --idempotency-key maf-2026-07-fx --json
saldo loans correct-payment --payment-id 12 --source-amount 375 --applied-amount 100 --exchange-rate 3.75 --json
saldo loans create --name ExtraCash --principal 5000 --currency PEN --installments 24 --collection-mode CREDIT_CARD_STATEMENT --credit-card-id 3 --credit-card-account-id 9 --external-reference extracash-2026 --json
saldo loans card-installment post 42 --idempotency-key extracash-42 --json
saldo loans card-installment reverse 17 --idempotency-key extracash-42-reverse --json
saldo subscriptions create --name "Movistar Internet" --amount 110 --currency PEN --billing-cycle MONTHLY --amount-type VARIABLE --charge-mode MANUAL --next-charge-date 2026-08-15T00:00:00Z --due-date 2026-08-20T00:00:00Z --next-charge-amount 120 --due-day 20 --category-id 5 --json
saldo subscriptions list --status active --json
saldo subscriptions list --status archived --json
saldo subscriptions get 1 --json
saldo subscriptions update 1 --charge-mode AUTOMATIC --next-charge-amount 125 --json
saldo subscriptions archive 1 --json
saldo subscriptions reactivate 1 --json
saldo subscriptions delete 1 --json
saldo subscriptions charge 1 --actual-amount 120 --idempotency-key movistar-2026-08 --json
saldo subscriptions correct-charge 44 --actual-amount 118.50 --json
saldo subscriptions history 1 --json
saldo subscriptions upcoming --days 30 --json
saldo budgets create --category-id 5 --monthly-limit 500 --currency PEN --json
saldo budgets list --json
```

Loan lifecycle status is `active`, `archived`, or `all`. Physical deletion is safe-only: the backend rejects a loan with payment or transaction history, so archive it instead. `--default-payment-account-id` must reference an active account; clear it with `--clear-default-payment-account`.

`loans schedule update` accepts either a raw JSON array or `{ "installments": [...] }`. Each row requires `number`, `dueDate`, `principal`, `interest`, `fee`, `insurance`, and `lateFee`; include `id` when replacing existing rows.

```json
{"installments":[{"id":"1","number":1,"dueDate":"2026-08-01","principal":90,"interest":10,"fee":0,"insurance":0,"lateFee":0}]}
```

For allocation overrides, `--allocations-file` accepts a raw array or `{ "allocations": [...] }` with `installmentId`, `principal`, `interest`, `fee`, `insurance`, and `lateFee`. Use `propose-allocation` first. Every `loans payment` requires `--idempotency-key`. Cross-currency payments must supply all three of `--source-amount`, `--applied-amount`, and the bank's `--exchange-rate`; same-currency payments omit them.

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
