package exchange

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sudowanderer/dca-bot-go/internal/config"
)

// Order represents a trading order result
type Order struct {
	ID       string          `json:"id"`
	Symbol   string          `json:"symbol"`
	Side     string          `json:"side"`     // "buy" or "sell"
	Type     string          `json:"type"`     // "market" or "limit"
	Quantity decimal.Decimal `json:"quantity"` // filled quantity
	Price    decimal.Decimal `json:"price"`    // average fill price
	Status   string          `json:"status"`   // "filled", "partial", "rejected"
}

// Exchange defines the interface for cryptocurrency exchange operations
type Exchange interface {
	// GetBalance returns the available balance for a specific asset
	GetBalance(ctx context.Context, asset string) (decimal.Decimal, error)

	// PlaceMarketBuyOrder places a market buy order with the specified quote amount
	// symbol: trading pair (e.g., "BTC-USDT")
	// quoteAmount: amount in quote currency to spend
	PlaceMarketBuyOrder(ctx context.Context, symbol string, quoteAmount decimal.Decimal) (*Order, error)
}

// NewExchange creates an Exchange instance based on the provided configuration
func NewExchange(cfg *config.DCAPayload) (Exchange, error) {
	// Use mock exchange for dry run mode
	if cfg.Flags.DryRun {
		return NewMockExchange(), nil
	}

	switch cfg.Exchange.Name {
	case "binance":
		return NewBinanceExchange(cfg)
	case "okx":
		return NewOKXExchange(cfg)
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", cfg.Exchange.Name)
	}
}

// NewBinanceExchange creates a Binance exchange instance (placeholder)
func NewBinanceExchange(cfg *config.DCAPayload) (Exchange, error) {
	// TODO: Implement Binance exchange
	return nil, fmt.Errorf("Binance exchange not implemented yet")
}

// NewOKXExchange creates an OKX exchange instance (placeholder)
func NewOKXExchange(cfg *config.DCAPayload) (Exchange, error) {
	// TODO: Implement OKX exchange
	return nil, fmt.Errorf("OKX exchange not implemented yet")
}

// MockExchange is a mock implementation for testing and dry run
type MockExchange struct{}

// NewMockExchange creates a new mock exchange instance
func NewMockExchange() Exchange {
	return &MockExchange{}
}

// GetBalance returns a mock balance for testing
func (m *MockExchange) GetBalance(ctx context.Context, asset string) (decimal.Decimal, error) {
	// Return a mock balance that's above typical thresholds for testing
	return decimal.NewFromFloat(10000), nil
}

// PlaceMarketBuyOrder simulates placing a market buy order
func (m *MockExchange) PlaceMarketBuyOrder(ctx context.Context, symbol string, quoteAmount decimal.Decimal) (*Order, error) {
	// Simulate a successful order with mock data
	return &Order{
		ID:       "mock-order-12345",
		Symbol:   symbol,
		Side:     "buy",
		Type:     "market",
		Quantity: quoteAmount.Div(decimal.NewFromFloat(50000)), // Assume BTC price ~50k
		Price:    decimal.NewFromFloat(50000),
		Status:   "filled",
	}, nil
}