---
name: saldo-cli
description: "Use when Codex needs to manage Saldo through the local `saldo` CLI: accounts, taxonomy, transactions, transfers, cards, loans, subscriptions, safe bulk imports, snapshots, or receipt drafts. Prefer this skill over direct GraphQL for routine Saldo operations."
---

# Saldo CLI

Use the `saldo` executable as the only interface for Saldo user operations. Do not read or edit the CLI session file; the CLI owns tokens, refresh, and authenticated headers internally.

## Core Rules

- Use `--json` for agent workflows unless the user explicitly wants human text.
- Use a stable `--idempotency-key` for payments, transfers, and retryable writes.
- Always run bulk registrations with `--dry-run` first and stop if `valid` is false.
- Set `SALDO_SESSION` to a unique path for this agent/thread when isolation matters.
- Set `SALDO_API_URL` or run `saldo config set api-url <url>` before login.
- Run `saldo auth whoami --json` before write operations when identity matters.
- Use `--profile <email>` when the session file has multiple saved logins and a non-default login is required. `--account <email>` is accepted as an alias.
- Do not inspect `SALDO_SESSION` contents. Treat the file as private implementation detail.
- Do not bypass the CLI with GraphQL calls unless the CLI lacks the needed capability.

## Setup Check

1. Confirm the CLI exists:

```bash
saldo --help
```

2. If unavailable but working inside the `saldo-cli` repo, build it:

```bash
go build -o saldo ./cmd/saldo
```

3. Configure backend URL:

```bash
saldo config set api-url https://saldo.example.com/graphql/ --json
```

4. Authenticate:

```bash
saldo auth login --email user@example.com --json
saldo auth whoami --json
```

If non-interactive login is needed, set `SALDO_PASSWORD` for the command invocation rather than placing the password in files.

The CLI can keep multiple active login profiles in one `SALDO_SESSION` file. The first saved profile is the default when no selector is passed:

```bash
saldo auth login --email first@example.com --json
saldo auth login --email second@example.com --json
saldo auth profiles --json
saldo --profile second@example.com auth whoami --json
```

## Common Workflows

For account/category/tag/transaction command examples, read [references/commands.md](references/commands.md).

### Multi-currency Credit Cards

Use `saldo credit-cards` for a card contract and its independent currency
ledgers. Filter lifecycle views with `list --status active|archived|all`; use
`update --status active|cancelled` for contractual state, which is independent
of archive/reactivate.

Create complex card ledgers from a JSON file, then manage each currency with
the `currencies` subcommands:

```bash
saldo credit-cards create --name "Interbank Visa" --issuer Interbank --currencies-file card.json --json
saldo credit-cards currencies set-default 3 --currency PEN --account-id 1 --json
```

Currency payments require a stable idempotency key. Use equal debit/applied
amounts with no FX for the same currency. For a cross-currency payment, record
the bank's actual debit, applied amount, and exchange rate; do not invent a
rate:

```bash
saldo credit-cards payment --card-id 3 --currency PEN --from-account-id 2 \
  --debit-amount 10 --applied-amount 37 --exchange-rate 3.7 \
  --idempotency-key visa-2026-07-usd-pen --json
```

Use `delete` only when a card has no financial history; archive it otherwise.

### Loans and Installments

Use `saldo loans list --status active|archived|all` for lifecycle views and
`get` for the persisted schedule and payment allocation history. Archive or
reactivate without changing whether the loan is paid; `delete` is permitted
only before financial history exists. Set an active source account at create or
update time with `--default-payment-account-id`.

For editable schedules and allocation overrides, use JSON files. First request
the deterministic oldest-first proposal, then pass an override only when it is
intentional:

```bash
saldo loans schedule update 1 --file schedule.json --json
saldo loans propose-allocation --loan-id 1 --applied-amount 100 --json
saldo loans payment --loan-id 1 --from-account-id 2 --amount 100 \
  --allocations-file allocations.json --idempotency-key loan-2026-08 --json
```

Every payment requires `--idempotency-key`. A same-currency explicit payment
may use equal `--source-amount` and `--applied-amount` without FX. If those
amounts differ, provide the bank's `--exchange-rate`; corrections use
`correct-payment` with the same amount and allocation fields.

### Services and Subscriptions

Use `saldo subscriptions` for recurring services. `list --status active|archived|all` controls lifecycle visibility; use `archive`, `reactivate`, and `delete` explicitly. Permanent deletion is only valid when the service has no recorded charges.

Set independent `--next-charge-date` and `--due-date`. A service can use `--amount-type FIXED|VARIABLE` and `--charge-mode MANUAL|AUTOMATIC`; automatic processing remains a backend deployment task and is not a user CLI command.

Manual charges require a stable retry key:

```bash
saldo subscriptions charge 1 --actual-amount 120 --idempotency-key movistar-2026-08 --json
saldo subscriptions correct-charge 44 --actual-amount 118.50 --json
saldo subscriptions history 1 --json
```

### Create a Transaction

1. Resolve the target account:

```bash
saldo accounts list --json
```

2. Resolve category or tags if needed:

```bash
saldo categories list --query food --type EXPENSE --json
saldo tags list --query online --json
```

3. Create the transaction:

```bash
saldo transactions create --account-id 1 --amount 25.50 --kind EXPENSE --currency PEN --date 2026-05-03T12:00:00Z --category-id 5 --description "Lunch" --json
```

### Receipt/OCR Draft

The agent performs OCR/vision and produces structured JSON. Send that JSON to the CLI for deterministic resolution and validation:

```bash
saldo transactions draft --file receipt.json --json
```

Show the preview to the user. After they confirm or edit account, amount, category, description, date, or tags, call `saldo transactions create ... --json` with the final values.

Draft JSON shape:

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

## Error Handling

- Missing API URL: set `SALDO_API_URL` or run `saldo config set api-url`.
- Missing login: run `saldo auth login --email <email>`. If multiple profiles are saved, pass `--profile <email>` to select the intended login.
- Auth/permission errors: run `saldo auth whoami --json`; if it fails, log in again.
- Draft warning: ask the user before committing, especially for total mismatches or unresolved account/category.
