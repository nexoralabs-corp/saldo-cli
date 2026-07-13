# Saldo CLI

Agent-friendly command line client for Saldo. The CLI connects to the Saldo
backend over GraphQL and owns all session/token handling internally.

## Install

Download the archive for your operating system and CPU from the
[latest GitHub release](https://github.com/nexoralabs-corp/saldo-cli/releases/latest).
Prebuilt binaries are available for Linux, macOS, and Windows on both x86-64
(`amd64`) and ARM64.

On Linux or macOS, extract the archive and put `saldo` on your `PATH`:

```bash
tar -xzf saldo_<version>_<os>_<arch>.tar.gz
mkdir -p ~/.local/bin
install -m 755 saldo ~/.local/bin/saldo
saldo --version
```

On Windows, extract the `.zip` file and move `saldo.exe` into a directory on
your `PATH`. The release also includes `checksums.txt` for SHA-256 verification.

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

## Publishing a release

Pushing a semantic version tag automatically tests the project and publishes
the prebuilt archives and checksums to GitHub Releases:

```bash
git tag v0.1.0
git push origin v0.1.0
```

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

## Multiple Login Profiles

The CLI can keep multiple logged-in sessions in the same session file. Each
login is stored as a profile keyed by the authenticated email address. When no
profile is selected, the CLI uses the first saved profile as the default.

```bash
saldo auth login --email first@example.com
saldo auth login --email second@example.com

saldo auth profiles
saldo --profile second@example.com auth whoami --json
saldo --profile second@example.com accounts list --json
```

`--account` is also accepted as an alias for `--profile` when selecting a login
profile:

```bash
saldo --account second@example.com transactions list --json
```

To log out of one saved profile, select it. To clear every saved profile, use
`--all`:

```bash
saldo --profile second@example.com auth logout
saldo auth logout --all
```

## Agent Core Commands

```bash
saldo accounts list --json
saldo accounts get 1 --json
saldo accounts create --name "Interbank" --type BANK --currency PEN --json
saldo accounts update 1 --name "Interbank Ahorros" --json
saldo accounts delete 1 --json

saldo categories list --query food --type EXPENSE --json
saldo categories create --name "Telefonía" --type EXPENSE --parent-id 5 --json
saldo tags list --query online --json
saldo tags create --name "UTP" --json

saldo transactions list --account-id 1 --from 2026-05-01T00:00:00Z --to 2026-05-31T23:59:59Z --json
saldo transactions create --account-id 1 --amount 25.50 --kind EXPENSE --currency PEN --date 2026-05-03T12:00:00Z --description "Lunch" --json
saldo transactions transfer --from-account-id 1 --to-account-id 2 --amount 100 --idempotency-key transfer-2026-07-12 --json

saldo credit-cards create --name "CMR - Falabella" --issuer FALABELLA --currencies-file card-currencies.json --json
saldo credit-cards list --status active --json
saldo credit-cards payment --card-id 3 --currency PEN --from-account-id 1 --debit-amount 100 --applied-amount 100 --idempotency-key cmr-2026-07 --json

saldo loans create --name "MAF – Agya" --lender MAF --currency PEN --outstanding-balance 761.80 --json
saldo loans list --status active --json
saldo loans get 1 --json
saldo loans payment --loan-id 1 --from-account-id 1 --amount 100 --date 2026-07-12 --idempotency-key maf-2026-07 --json
saldo loans schedule get 1 --json

saldo subscriptions create --name "Movistar Internet" --amount 110 --currency PEN --billing-cycle MONTHLY --amount-type VARIABLE --charge-mode MANUAL --next-charge-date 2026-08-15T00:00:00Z --due-date 2026-08-20T00:00:00Z --next-charge-amount 120 --due-day 20 --category-id 5 --json
saldo subscriptions list --status active --json
saldo subscriptions get 1 --json
saldo subscriptions charge 1 --actual-amount 120 --idempotency-key movistar-2026-08 --json
saldo subscriptions correct-charge 1 --actual-amount 118.50 --json
saldo subscriptions history 1 --json
saldo subscriptions upcoming --days 30 --json

saldo budgets create --category-id 5 --monthly-limit 500 --currency PEN --json
saldo budgets list --json

saldo snapshot ai --from 2026-05-01 --to 2026-05-31 --section transactions --json
```

## Loans

Loans support `get`, `update`, `archive`, `reactivate`, and safe `delete`, with `list --status active|archived|all`. Configure an active default source through `--default-payment-account-id`; inspect or replace the durable schedule with `loans schedule get|update --file schedule.json`; call `propose-allocation` before passing a custom `--allocations-file` to `payment` or `correct-payment`. Payments require `--idempotency-key`. When a payment crosses currencies, send the actual `--source-amount`, debt `--applied-amount`, and bank `--exchange-rate` together.

## Multi-currency credit cards

Credit cards are grouped by contract; each currency has its own balance and
ledger. Create them from a JSON array so complex, repeatable configurations
remain safe for agents:

```json
[
  {"currency":"PEN","creditLimit":5000,"closingDay":15,"dueDay":5,"defaultPaymentAccountId":"1"},
  {"currency":"USD","creditLimit":1500,"closingDay":15,"dueDay":5,"defaultPaymentAccountId":"2"}
]
```

```bash
saldo credit-cards get 3 --json
saldo credit-cards update 3 --status cancelled --json
saldo credit-cards currencies add 3 --currency USD --credit-limit 1500 --default-payment-account-id 2 --json
saldo credit-cards currencies update 3 --currency PEN --minimum-payment 250 --json
saldo credit-cards currencies set-default 3 --currency PEN --account-id 1 --json
saldo credit-cards archive 3 --json
saldo credit-cards reactivate 3 --json
```

Currency payments require an idempotency key. Same-currency payments must use
equal debit/applied amounts and omit the exchange rate. Cross-currency payments
must record the bank's actual debit, applied amount, and exchange rate:

```bash
saldo credit-cards payment --card-id 3 --currency PEN --from-account-id 2 \
  --debit-amount 10 --applied-amount 37 --exchange-rate 3.7 \
  --idempotency-key card-2026-07-usd-pen --json
```

`delete` is safe only for cards without financial history; archive a card that
has payments or transactions instead.

## Safe Bulk Import

Each registration must include a stable, user-chosen `idempotencyKey`. Previewing validates required fields and rejects duplicate keys or duplicate content inside the file. Reusing the same key in a later execution returns the original backend transaction without applying its balance twice.

```json
{
  "registrations": [
    {
      "accountId": "1",
      "amount": 110,
      "kind": "EXPENSE",
      "currency": "PEN",
      "date": "2026-07-12",
      "categoryId": "5",
      "description": "Movistar Internet",
      "tags": ["Wilber"],
      "idempotencyKey": "movistar-2026-07"
    }
  ]
}
```

```bash
saldo import registrations --file gastos.json --dry-run --json
saldo import registrations --file gastos.json --json
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
