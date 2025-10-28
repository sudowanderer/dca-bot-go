package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sudowanderer/dca-bot-go/env"
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
	fmt.Println("Hello, DCA Bot!")
	return nil
}
