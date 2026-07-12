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

saldo credit-cards create --name "CMR - Falabella" --issuer FALABELLA --currency PEN --credit-limit 0 --closing-day 0 --due-day 0 --json
saldo credit-cards list --json
saldo credit-cards payment --card-id 3 --from-account-id 1 --amount 100 --idempotency-key cmr-2026-07 --json

saldo loans create --name "MAF – Agya" --lender MAF --currency PEN --outstanding-balance 761.80 --json
saldo loans list --json
saldo loans payment --loan-id 1 --from-account-id 1 --amount 100 --date 2026-07-12 --idempotency-key maf-2026-07 --json

saldo subscriptions create --name "Movistar Internet" --amount 110 --currency PEN --frequency MONTHLY --due-day 15 --category-id 5 --json
saldo subscriptions list --json
saldo subscriptions upcoming --days 30 --json

saldo budgets create --category-id 5 --monthly-limit 500 --currency PEN --json
saldo budgets list --json

saldo snapshot ai --from 2026-05-01 --to 2026-05-31 --section transactions --json
```

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
