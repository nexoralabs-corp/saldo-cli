package cli

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newCreditCardBalancesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "balances", Short: "Adjust and audit card balances"}
	cmd.AddCommand(newCreditCardBalanceAdjustCommand(state), newCreditCardBalanceHistoryCommand(state))
	return cmd
}

func newCreditCardBalanceAdjustCommand(state *appState) *cobra.Command {
	var cardID, currency, side, date, reason, note, key string
	var amount float64
	var allowInactive bool
	cmd := &cobra.Command{Use: "adjust", Short: "Set a card currency balance with an audit trail", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" || currency == "" || amount < 0 || strings.TrimSpace(key) == "" {
			return fmt.Errorf("--card-id, --currency, non-negative --target-amount, and --idempotency-key are required")
		}
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("--reason is required")
		}
		side = strings.ToUpper(strings.TrimSpace(side))
		if side != "DEBT" && side != "CREDIT" {
			return fmt.Errorf("--balance-side must be debt or credit")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			AdjustCreditCardBalance creditCardBalanceAdjustment `json:"adjustCreditCardBalance"`
		}
		input := map[string]any{"cardId": cardID, "currency": strings.ToUpper(currency), "targetAmount": amount, "balanceSide": side, "effectiveDate": date, "reason": reason, "note": note, "idempotencyKey": key, "allowInactive": allowInactive}
		if err = client.Do(context.Background(), adjustCreditCardBalanceMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.AdjustCreditCardBalance)
		}
		return writeHuman("Adjusted %s balance to %.2f (%s)\n", strings.ToUpper(currency), amount, strings.ToLower(side))
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	cmd.Flags().StringVar(&currency, "currency", "", "card currency")
	cmd.Flags().Float64Var(&amount, "target-amount", 0, "visible target balance")
	cmd.Flags().StringVar(&side, "balance-side", "debt", "debt or credit")
	cmd.Flags().StringVar(&date, "date", "", "effective date (YYYY-MM-DD; defaults to today)")
	cmd.Flags().StringVar(&reason, "reason", "", "audit reason")
	cmd.Flags().StringVar(&note, "note", "", "optional audit note")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key")
	cmd.Flags().BoolVar(&allowInactive, "allow-inactive", false, "explicitly allow an archived/cancelled card")
	return cmd
}

func newCreditCardBalanceHistoryCommand(state *appState) *cobra.Command {
	var currency string
	cmd := &cobra.Command{Use: "history <card-id>", Short: "List card balance adjustments", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCardBalanceAdjustments []creditCardBalanceAdjustment `json:"creditCardBalanceAdjustments"`
		}
		if err = client.Do(context.Background(), creditCardBalanceAdjustmentsQuery, map[string]any{"cardId": args[0], "currency": nullableString(currency)}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCardBalanceAdjustments)
		}
		for _, item := range data.CreditCardBalanceAdjustments {
			if err := writeHuman("%s\t%s\t%.2f\t%s\n", item.ID, item.EffectiveDate, item.TargetAmount, item.Reason); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&currency, "currency", "", "optional card currency")
	return cmd
}

func newCreditCardLimitsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "limits", Short: "Manage shared credit lines"}
	cmd.AddCommand(newCreditCardLimitGetCommand(state), newCreditCardLimitSharedCommand(state), newCreditCardLimitPerCurrencyCommand(state))
	return cmd
}

func newCreditCardLimitGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "get <card-id>", Short: "Get a card credit line summary", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCard creditCard `json:"creditCard"`
		}
		if err = client.Do(context.Background(), creditCardQuery, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCard.LimitSummary)
		}
		if data.CreditCard.LimitSummary == nil {
			return writeHuman("No credit line summary\n")
		}
		summary := data.CreditCard.LimitSummary
		return writeHuman("%s\t%s\tused=%v\tavailable=%v\n", summary.Mode, valueOrEmpty(summary.Currency), summary.Used, summary.Available)
	}}
}

func newCreditCardLimitSharedCommand(state *appState) *cobra.Command {
	var limit float64
	var currency string
	var rates []string
	cmd := &cobra.Command{Use: "set-shared <card-id>", Short: "Set one shared credit line across currencies", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if limit <= 0 || currency == "" {
			return fmt.Errorf("--limit and --currency are required")
		}
		parsed, err := parseLimitRates(rates)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateCreditCardLimitConfiguration creditCard `json:"updateCreditCardLimitConfiguration"`
		}
		input := map[string]any{"mode": "SHARED", "sharedCreditLimit": limit, "sharedLimitCurrency": strings.ToUpper(currency), "rates": parsed}
		if err = client.Do(context.Background(), updateCreditCardLimitMutation, map[string]any{"cardId": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateCreditCardLimitConfiguration)
		}
		return writeHuman("Configured shared %s %.2f line for card %s\n", strings.ToUpper(currency), limit, args[0])
	}}
	cmd.Flags().Float64Var(&limit, "limit", 0, "shared credit limit")
	cmd.Flags().StringVar(&currency, "currency", "", "shared line currency")
	cmd.Flags().StringSliceVar(&rates, "rate", nil, "rate as CURRENCY=base-units, repeatable")
	return cmd
}

func newCreditCardLimitPerCurrencyCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "set-per-currency <card-id>", Short: "Use independent currency limits", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateCreditCardLimitConfiguration creditCard `json:"updateCreditCardLimitConfiguration"`
		}
		if err = client.Do(context.Background(), updateCreditCardLimitMutation, map[string]any{"cardId": args[0], "input": map[string]any{"mode": "PER_CURRENCY"}}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateCreditCardLimitConfiguration)
		}
		return writeHuman("Configured independent currency limits for card %s\n", args[0])
	}}
}

func parseLimitRates(values []string) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("--rate must use CURRENCY=RATE")
		}
		rate, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || rate <= 0 {
			return nil, fmt.Errorf("invalid rate %q", value)
		}
		result = append(result, map[string]any{"currency": strings.ToUpper(strings.TrimSpace(parts[0])), "rateToBase": rate})
	}
	return result, nil
}

func newCreditCardStatementsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "statements", Short: "Import, reconcile, and inspect card statements"}
	cmd.AddCommand(newCreditCardStatementImportCommand(state), newCreditCardStatementConfirmCommand(state), newCreditCardStatementListCommand(state), newCreditCardStatementGetCommand(state), newCreditCardStatementCreateCommand(state))
	return cmd
}

func newCreditCardStatementImportCommand(state *appState) *cobra.Command {
	var cardID, currency, file, closingDate, dueDate, pdfPassword string
	var opening, balance, minimum, total float64
	var hasMinimum, hasTotal bool
	var dryRun bool
	cmd := &cobra.Command{Use: "import", Short: "Parse a PDF or CSV statement without changing balances", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" || currency == "" || file == "" || closingDate == "" {
			return fmt.Errorf("--card-id, --currency, --file, and --closing-date are required")
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read statement: %w", err)
		}
		mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(file)))
		if mimeType == "" {
			mimeType = "text/csv"
		}
		if pdfPassword == "" {
			pdfPassword = os.Getenv("SALDO_PDF_PASSWORD")
		}
		input := map[string]any{"cardId": cardID, "currency": strings.ToUpper(currency), "filename": filepath.Base(file), "mimeType": mimeType, "contentBase64": base64.StdEncoding.EncodeToString(raw), "closingDate": closingDate, "openingBalance": opening, "statementBalance": balance, "dueDate": nullableString(dueDate)}
		if pdfPassword != "" {
			input["pdfPassword"] = pdfPassword
		}
		if hasMinimum {
			input["minimumPayment"] = minimum
		}
		if hasTotal {
			input["totalPayment"] = total
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			PreviewCreditCardStatementImport creditCardStatementImport `json:"previewCreditCardStatementImport"`
		}
		if err = client.Do(context.Background(), previewCreditCardStatementImportMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(map[string]any{"dryRun": dryRun, "import": data.PreviewCreditCardStatementImport})
		}
		return writeHuman("Parsed statement import %s; confirm it to apply financial entries.\n", data.PreviewCreditCardStatementImport.ID)
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	cmd.Flags().StringVar(&currency, "currency", "", "statement currency")
	cmd.Flags().StringVar(&file, "file", "", "PDF or CSV statement")
	cmd.Flags().StringVar(&pdfPassword, "pdf-password", "", "PDF password; prefer SALDO_PDF_PASSWORD")
	cmd.Flags().StringVar(&closingDate, "closing-date", "", "statement closing date")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "optional due date")
	cmd.Flags().Float64Var(&opening, "opening-balance", 0, "opening statement debt")
	cmd.Flags().Float64Var(&balance, "statement-balance", 0, "closing statement debt")
	cmd.Flags().Float64Var(&minimum, "minimum-payment", 0, "minimum payment")
	cmd.Flags().Float64Var(&total, "total-payment", 0, "full statement payment")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "explicitly signal parse/review only; this command never applies entries")
	cmd.Flags().Lookup("minimum-payment").NoOptDefVal = "0"
	cmd.Flags().Lookup("total-payment").NoOptDefVal = "0"
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		hasMinimum = cmd.Flags().Changed("minimum-payment")
		hasTotal = cmd.Flags().Changed("total-payment")
		return nil
	}
	return cmd
}

func newCreditCardStatementConfirmCommand(state *appState) *cobra.Command {
	var importID, key string
	var ignored []string
	cmd := &cobra.Command{Use: "confirm", Short: "Apply a reviewed statement import", RunE: func(cmd *cobra.Command, args []string) error {
		if importID == "" || key == "" {
			return fmt.Errorf("--import-id and --idempotency-key are required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			ConfirmCreditCardStatementImport creditCardStatementImport `json:"confirmCreditCardStatementImport"`
		}
		input := map[string]any{"importId": importID, "ignoredEntryIds": ignored, "idempotencyKey": key}
		if err = client.Do(context.Background(), confirmCreditCardStatementImportMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.ConfirmCreditCardStatementImport)
		}
		return writeHuman("Applied statement import %s\n", importID)
	}}
	cmd.Flags().StringVar(&importID, "import-id", "", "statement import ID")
	cmd.Flags().StringSliceVar(&ignored, "ignore-entry-id", nil, "entry ID to ignore, repeatable")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key")
	return cmd
}

func newCreditCardStatementListCommand(state *appState) *cobra.Command {
	var cardID, currency string
	cmd := &cobra.Command{Use: "list", Short: "List card statement history", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" {
			return fmt.Errorf("--card-id is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCardStatements []creditCardStatement `json:"creditCardStatements"`
		}
		if err = client.Do(context.Background(), creditCardStatementsQuery, map[string]any{"cardId": cardID, "currency": nullableString(currency)}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCardStatements)
		}
		for _, statement := range data.CreditCardStatements {
			if err := writeHuman("%s\t%s\t%.2f\t%s\n", statement.ID, statement.ClosingDate, statement.StatementBalance, statement.Status); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	cmd.Flags().StringVar(&currency, "currency", "", "optional card currency")
	return cmd
}

func newCreditCardStatementGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get a statement and its entries", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCardStatement creditCardStatement `json:"creditCardStatement"`
		}
		if err = client.Do(context.Background(), creditCardStatementQuery, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCardStatement)
		}
		return writeHuman("%s\t%s\t%.2f\n", data.CreditCardStatement.ID, data.CreditCardStatement.ClosingDate, data.CreditCardStatement.StatementBalance)
	}}
}

func newCreditCardStatementCreateCommand(state *appState) *cobra.Command {
	var cardID, currency, closingDate, dueDate, notes string
	var opening, balance, minimum, total float64
	cmd := &cobra.Command{Use: "create", Short: "Create a manual statement history entry", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" || currency == "" || closingDate == "" {
			return fmt.Errorf("--card-id, --currency, and --closing-date are required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateCreditCardStatement creditCardStatement `json:"createCreditCardStatement"`
		}
		input := map[string]any{"cardId": cardID, "currency": strings.ToUpper(currency), "closingDate": closingDate, "dueDate": nullableString(dueDate), "openingBalance": opening, "statementBalance": balance, "minimumPayment": minimum, "totalPayment": total, "notes": notes}
		if err = client.Do(context.Background(), createCreditCardStatementMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateCreditCardStatement)
		}
		return writeHuman("Created statement %s\n", data.CreateCreditCardStatement.ID)
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	cmd.Flags().StringVar(&currency, "currency", "", "statement currency")
	cmd.Flags().StringVar(&closingDate, "closing-date", "", "closing date")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "due date")
	cmd.Flags().Float64Var(&opening, "opening-balance", 0, "opening balance")
	cmd.Flags().Float64Var(&balance, "statement-balance", 0, "statement balance")
	cmd.Flags().Float64Var(&minimum, "minimum-payment", 0, "minimum payment")
	cmd.Flags().Float64Var(&total, "total-payment", 0, "total payment")
	cmd.Flags().StringVar(&notes, "notes", "", "notes")
	return cmd
}

func newCreditCardChargesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "charges", Short: "Manage card memberships and insurance"}
	cmd.AddCommand(newCreditCardChargeCreateCommand(state), newCreditCardChargeListCommand(state), newCreditCardChargeProjectCommand(state), newCreditCardChargeWaiveCommand(state), newCreditCardChargeRecordCommand(state), newCreditCardChargeHistoryCommand(state))
	return cmd
}

func newCreditCardChargeCreateCommand(state *appState) *cobra.Command {
	var cardID, currency, name, chargeType, nextDate, calculation, waiver, conditions string
	var fixed, percentage, minimum, maximum, threshold float64
	cmd := &cobra.Command{Use: "create", Short: "Create a membership or insurance rule", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" || currency == "" || name == "" || nextDate == "" {
			return fmt.Errorf("--card-id, --currency, --name, and --next-charge-date are required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateCreditCardChargeRule creditCardChargeRule `json:"createCreditCardChargeRule"`
		}
		input := map[string]any{"cardId": cardID, "currency": strings.ToUpper(currency), "name": name, "chargeType": strings.ToUpper(chargeType), "nextChargeDate": nextDate, "calculation": strings.ToUpper(calculation), "waiverPolicy": strings.ToUpper(waiver), "conditions": conditions}
		if cmd.Flags().Changed("fixed-amount") {
			input["fixedAmount"] = fixed
		}
		if cmd.Flags().Changed("percentage") {
			input["percentage"] = percentage
		}
		if cmd.Flags().Changed("minimum-amount") {
			input["minimumAmount"] = minimum
		}
		if cmd.Flags().Changed("maximum-amount") {
			input["maximumAmount"] = maximum
		}
		if cmd.Flags().Changed("waiver-threshold") {
			input["waiverThreshold"] = threshold
		}
		if err = client.Do(context.Background(), createCreditCardChargeRuleMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateCreditCardChargeRule)
		}
		return writeHuman("Created card charge rule %s\n", data.CreateCreditCardChargeRule.ID)
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	cmd.Flags().StringVar(&currency, "currency", "", "card currency")
	cmd.Flags().StringVar(&name, "name", "", "rule name")
	cmd.Flags().StringVar(&chargeType, "type", "INSURANCE", "INSURANCE or MEMBERSHIP")
	cmd.Flags().StringVar(&nextDate, "next-charge-date", "", "next charge date")
	cmd.Flags().StringVar(&calculation, "calculation", "FIXED", "FIXED, PERCENT_OF_STATEMENT_BALANCE, or MANUAL")
	cmd.Flags().Float64Var(&fixed, "fixed-amount", 0, "fixed amount")
	cmd.Flags().Float64Var(&percentage, "percentage", 0, "percentage")
	cmd.Flags().Float64Var(&minimum, "minimum-amount", 0, "minimum amount")
	cmd.Flags().Float64Var(&maximum, "maximum-amount", 0, "maximum amount")
	cmd.Flags().StringVar(&waiver, "waiver-policy", "NONE", "NONE, MANUAL, MIN_PURCHASE_COUNT, or MIN_PURCHASE_AMOUNT")
	cmd.Flags().Float64Var(&threshold, "waiver-threshold", 0, "waiver threshold")
	cmd.Flags().StringVar(&conditions, "conditions", "", "bank conditions")
	return cmd
}

func newCreditCardChargeListCommand(state *appState) *cobra.Command {
	var cardID string
	cmd := &cobra.Command{Use: "list", Short: "List card charge rules", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" {
			return fmt.Errorf("--card-id is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCardChargeRules []creditCardChargeRule `json:"creditCardChargeRules"`
		}
		if err = client.Do(context.Background(), creditCardChargeRulesQuery, map[string]any{"cardId": cardID}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCardChargeRules)
		}
		for _, rule := range data.CreditCardChargeRules {
			if err := writeHuman("%s\t%s\t%s\n", rule.ID, rule.Name, rule.NextChargeDate); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	return cmd
}

func newCreditCardChargeProjectCommand(state *appState) *cobra.Command {
	var statementID string
	cmd := &cobra.Command{Use: "project <rule-id>", Short: "Project a charge against a statement", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			ProjectCreditCardCharge creditCardChargeOccurrence `json:"projectCreditCardCharge"`
		}
		if err = client.Do(context.Background(), projectCreditCardChargeMutation, map[string]any{"ruleId": args[0], "statementId": nullableString(statementID)}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.ProjectCreditCardCharge)
		}
		return writeHuman("Projected charge %s\n", data.ProjectCreditCardCharge.ID)
	}}
	cmd.Flags().StringVar(&statementID, "statement-id", "", "optional statement ID")
	return cmd
}

func newCreditCardChargeWaiveCommand(state *appState) *cobra.Command {
	var reason string
	cmd := &cobra.Command{Use: "waive <occurrence-id>", Short: "Waive a projected card charge", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			WaiveCreditCardCharge creditCardChargeOccurrence `json:"waiveCreditCardCharge"`
		}
		if err = client.Do(context.Background(), waiveCreditCardChargeMutation, map[string]any{"occurrenceId": args[0], "reason": reason}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.WaiveCreditCardCharge)
		}
		return writeHuman("Waived card charge %s\n", args[0])
	}}
	cmd.Flags().StringVar(&reason, "reason", "", "waiver reason")
	return cmd
}

func newCreditCardChargeRecordCommand(state *appState) *cobra.Command {
	var amount float64
	var key string
	cmd := &cobra.Command{Use: "record <occurrence-id>", Short: "Record a confirmed card charge", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if key == "" {
			return fmt.Errorf("--idempotency-key is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			RecordCreditCardCharge creditCardChargeOccurrence `json:"recordCreditCardCharge"`
		}
		variables := map[string]any{"occurrenceId": args[0], "idempotencyKey": key}
		if cmd.Flags().Changed("amount") {
			variables["amount"] = amount
		}
		if err = client.Do(context.Background(), recordCreditCardChargeMutation, variables, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.RecordCreditCardCharge)
		}
		return writeHuman("Recorded card charge %s\n", args[0])
	}}
	cmd.Flags().Float64Var(&amount, "amount", 0, "actual charge amount")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key")
	return cmd
}

func newCreditCardChargeHistoryCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "history <rule-id>", Short: "List projected and charged occurrences", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCardChargeOccurrences []creditCardChargeOccurrence `json:"creditCardChargeOccurrences"`
		}
		if err = client.Do(context.Background(), creditCardChargeOccurrencesQuery, map[string]any{"ruleId": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCardChargeOccurrences)
		}
		for _, occurrence := range data.CreditCardChargeOccurrences {
			if err := writeHuman("%s\t%s\t%.2f\t%s\n", occurrence.ID, occurrence.ScheduledFor, occurrence.FinalAmount, occurrence.Status); err != nil {
				return err
			}
		}
		return nil
	}}
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

const adjustmentFields = `id cardAccountId previousBalance targetAmount balanceSide delta effectiveDate reason note source transactionId idempotencyKey`
const statementEntryFields = `id operationDate description amount currency isCredit entryType status transactionId`
const statementFields = `id cardAccountId closingDate dueDate openingBalance statementBalance minimumPayment totalPayment status source entries { ` + statementEntryFields + ` }`
const statementImportFields = `id filename mimeType parserName status summary statements { ` + statementFields + ` }`
const chargeRuleFields = `id cardAccountId name chargeType nextChargeDate calculation fixedAmount percentage waiverPolicy isActive`
const chargeOccurrenceFields = `id ruleId scheduledFor baseBalance calculatedAmount waivedAmount finalAmount status transactionId`
const adjustCreditCardBalanceMutation = `mutation AdjustCreditCardBalance($input: AdjustCreditCardBalanceInput!) { adjustCreditCardBalance(input: $input) { ` + adjustmentFields + ` } }`
const creditCardBalanceAdjustmentsQuery = `query CreditCardBalanceAdjustments($cardId: ID!, $currency: String) { creditCardBalanceAdjustments(cardId: $cardId, currency: $currency) { ` + adjustmentFields + ` } }`
const updateCreditCardLimitMutation = `mutation UpdateCreditCardLimit($cardId: ID!, $input: UpdateCreditCardLimitInput!) { updateCreditCardLimitConfiguration(cardId: $cardId, input: $input) { ` + creditCardFields + ` } }`
const previewCreditCardStatementImportMutation = `mutation PreviewCreditCardStatementImport($input: PreviewCreditCardStatementImportInput!) { previewCreditCardStatementImport(input: $input) { ` + statementImportFields + ` } }`
const confirmCreditCardStatementImportMutation = `mutation ConfirmCreditCardStatementImport($input: ConfirmCreditCardStatementImportInput!) { confirmCreditCardStatementImport(input: $input) { ` + statementImportFields + ` } }`
const creditCardStatementsQuery = `query CreditCardStatements($cardId: ID!, $currency: String) { creditCardStatements(cardId: $cardId, currency: $currency) { ` + statementFields + ` } }`
const creditCardStatementQuery = `query CreditCardStatement($id: ID!) { creditCardStatement(id: $id) { ` + statementFields + ` } }`
const createCreditCardStatementMutation = `mutation CreateCreditCardStatement($input: CreateCreditCardStatementInput!) { createCreditCardStatement(input: $input) { ` + statementFields + ` } }`
const createCreditCardChargeRuleMutation = `mutation CreateCreditCardChargeRule($input: CreateCreditCardChargeRuleInput!) { createCreditCardChargeRule(input: $input) { ` + chargeRuleFields + ` } }`
const creditCardChargeRulesQuery = `query CreditCardChargeRules($cardId: ID!) { creditCardChargeRules(cardId: $cardId) { ` + chargeRuleFields + ` } }`
const projectCreditCardChargeMutation = `mutation ProjectCreditCardCharge($ruleId: ID!, $statementId: ID) { projectCreditCardCharge(ruleId: $ruleId, statementId: $statementId) { ` + chargeOccurrenceFields + ` } }`
const waiveCreditCardChargeMutation = `mutation WaiveCreditCardCharge($occurrenceId: ID!, $reason: String!) { waiveCreditCardCharge(occurrenceId: $occurrenceId, reason: $reason) { ` + chargeOccurrenceFields + ` } }`
const recordCreditCardChargeMutation = `mutation RecordCreditCardCharge($occurrenceId: ID!, $amount: Float, $idempotencyKey: String!) { recordCreditCardCharge(occurrenceId: $occurrenceId, amount: $amount, idempotencyKey: $idempotencyKey) { ` + chargeOccurrenceFields + ` } }`
const creditCardChargeOccurrencesQuery = `query CreditCardChargeOccurrences($ruleId: ID!) { creditCardChargeOccurrences(ruleId: $ruleId) { ` + chargeOccurrenceFields + ` } }`
