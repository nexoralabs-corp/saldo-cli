---
name: saldo-cli
description: "Use when Codex needs to interact with a Saldo personal-finance backend through the local `saldo` CLI: authenticate, inspect accounts, search categories/tags, list or create transactions, generate AI snapshots, or turn agent-extracted receipt/OCR data into a validated transaction draft. Prefer this skill over direct GraphQL or backend code for routine Saldo operations."
---

# Saldo CLI

Use the `saldo` executable as the only interface for Saldo user operations. Do not read or edit the CLI session file; the CLI owns tokens, refresh, and authenticated headers internally.

## Core Rules

- Use `--json` for agent workflows unless the user explicitly wants human text.
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
