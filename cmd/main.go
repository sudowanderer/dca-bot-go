package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sudowanderer/dca-bot-go/env"
	"github.com/sudowanderer/dca-bot-go/internal/config"
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

	// TODO: Implement actual DCA logic here

	return nil
}
