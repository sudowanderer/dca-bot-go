package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/shopspring/decimal"
	"github.com/sudowanderer/dca-bot-go/env"
	"github.com/sudowanderer/dca-bot-go/internal/config"
	"github.com/sudowanderer/dca-bot-go/internal/exchange"
)

func main() {
	if env.IsLambdaEnvironment() {
		// normal Lambda entrypoint
		lambda.Start(handleRequest)
		return
	}

	// --- local testing mode ---
	log.Println("🌱 Running in local mode, reading local_event.json …")

	data, err := os.ReadFile("local_event.json")
	if err != nil {
		log.Fatalf("failed to read event file: %v", err)
	}

	if err := handleRequest(context.Background(), data); err != nil {
		log.Fatalf("error in handleRequest: %v", err)
	}
}

func handleRequest(ctx context.Context, event json.RawMessage) error {
	// Parse the new DCA payload format
	payload, err := config.ParseDCAPayload(event)
	if err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	log.Printf("📊 Parsed DCA configuration:")
	log.Printf("   Exchange: %s", payload.Exchange.Name)
	log.Printf("   Symbol: %s", payload.Strategy.Symbol)
	log.Printf("   Quote Amount: %s", payload.Strategy.QuoteAmount)
	log.Printf("   Balance Threshold: %s", payload.Strategy.BalanceThreshold)
	log.Printf("   Order Type: %s", payload.Strategy.OrderType)
	log.Printf("   Dry Run: %v", payload.Flags.DryRun)
	log.Printf("   Credential Type: %s", payload.Exchange.Credentials.Type)

	if payload.Notifications.Telegram != nil {
		log.Printf("   Telegram Notification: %s", payload.Notifications.Telegram.Type)
	}

	// Convert to unified format for backward compatibility if needed
	unified, err := payload.ToUnified()
	if err != nil {
		return fmt.Errorf("failed to convert to unified format: %w", err)
	}

	log.Printf("🚀 DCA Bot processing %s on %s (DryRun: %v)",
		unified.Symbol, unified.Exchange, unified.DryRun)

	// Create exchange instance
	exchange, err := exchange.NewExchange(payload)
	if err != nil {
		return fmt.Errorf("failed to create exchange: %w", err)
	}

	// Run DCA strategy
	if err := runDCAStrategy(ctx, payload, exchange); err != nil {
		return fmt.Errorf("DCA strategy failed: %w", err)
	}

	return nil
}

// runDCAStrategy executes the DCA trading strategy
func runDCAStrategy(ctx context.Context, payload *config.DCAPayload, exc exchange.Exchange) error {
	log.Printf("🔍 Starting DCA strategy execution...")

	// Parse quote amount
	quoteAmount, err := decimal.NewFromString(payload.Strategy.QuoteAmount)
	if err != nil {
		return fmt.Errorf("invalid quote amount: %w", err)
	}

	// Step 1: Place market buy order
	if payload.Flags.DryRun {
		log.Printf("🧪 DRY RUN: Simulating market buy order for %s %s", quoteAmount.String(), payload.Strategy.Symbol)
	} else {
		log.Printf("📈 Placing market buy order: %s %s", quoteAmount.String(), payload.Strategy.Symbol)
	}

	order, err := exc.PlaceMarketBuyOrder(ctx, payload.Strategy.Symbol, quoteAmount)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	log.Printf("✅ Order executed successfully:")
	log.Printf("   Order ID: %s", order.ID)
	log.Printf("   Symbol: %s", order.Symbol)
	log.Printf("   Quantity: %s", order.Quantity.String())
	log.Printf("   Price: %s", order.Price.String())
	log.Printf("   Status: %s", order.Status)

	// Step 2: Check remaining balance and send notification if low
	if payload.Strategy.BalanceThreshold != "" {
		if err := checkBalanceAndNotify(ctx, payload, exc); err != nil {
			log.Printf("⚠️ Balance check failed: %v", err)
			// Don't return error - order was successful (or would be in dry run)
		}
	}

	// TODO: Send success notification

	return nil
}

// checkBalanceAndNotify checks remaining balance and sends notification if below threshold
func checkBalanceAndNotify(ctx context.Context, payload *config.DCAPayload, exc exchange.Exchange) error {
	// Extract quote currency from symbol (e.g., "BTC-USDT" -> "USDT")
	quoteCurrency, err := extractQuoteCurrency(payload.Strategy.Symbol)
	if err != nil {
		return fmt.Errorf("failed to extract quote currency: %w", err)
	}

	// Get current balance
	balance, err := exc.GetBalance(ctx, quoteCurrency)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	log.Printf("💰 Current %s balance after order: %s", quoteCurrency, balance.String())

	// Parse balance threshold
	threshold, err := decimal.NewFromString(payload.Strategy.BalanceThreshold)
	if err != nil {
		return fmt.Errorf("invalid balance threshold: %w", err)
	}

	// Check if balance is below threshold
	if balance.LessThan(threshold) {
		log.Printf("⚠️ Balance is below threshold: %s < %s", balance.String(), threshold.String())
		// TODO: Send low balance notification via Telegram
		return sendLowBalanceNotification(payload, quoteCurrency, balance, threshold)
	}

	log.Printf("✅ Balance is sufficient: %s >= %s (threshold)", balance.String(), threshold.String())
	return nil
}

// sendLowBalanceNotification sends a notification about low balance
func sendLowBalanceNotification(payload *config.DCAPayload, currency string, balance, threshold decimal.Decimal) error {
	// TODO: Implement Telegram notification
	log.Printf("📢 Would send low balance notification:")
	log.Printf("   Currency: %s", currency)
	log.Printf("   Current Balance: %s", balance.String())
	log.Printf("   Threshold: %s", threshold.String())
	log.Printf("   Symbol: %s", payload.Strategy.Symbol)
	
	if payload.Notifications.Telegram != nil {
		log.Printf("   Telegram notification configured: %s", payload.Notifications.Telegram.Type)
	}
	
	return nil
}

// extractQuoteCurrency extracts the quote currency from a trading pair symbol
func extractQuoteCurrency(symbol string) (string, error) {
	// Handle different symbol formats: "BTC-USDT", "BTCUSDT", etc.
	if strings.Contains(symbol, "-") {
		parts := strings.Split(symbol, "-")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid symbol format: %s", symbol)
		}
		return parts[1], nil
	}
	
	// For symbols like "BTCUSDT", assume common quote currencies
	commonQuotes := []string{"USDT", "USDC", "BUSD", "USD", "BTC", "ETH", "FDUSD"}
	for _, quote := range commonQuotes {
		if strings.HasSuffix(symbol, quote) {
			return quote, nil
		}
	}
	
	return "", fmt.Errorf("unable to extract quote currency from symbol: %s", symbol)
}
