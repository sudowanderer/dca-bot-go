package config

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseDCAPayload_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DCAPayload
	}{
		{
			name: "binance_ssm_credentials",
			input: `{
				"version": "v2",
				"exchange": {
					"name": "binance",
					"credentials": {
						"type": "ssm",
						"config": {
							"apiKeyPath": "/myapp/binance/apiKey",
							"apiSecretPath": "/myapp/binance/apiSecret"
						}
					}
				},
				"strategy": {
					"symbol": "BTC-USDT",
					"quoteAmount": "10.00",
					"balanceThreshold": "5000.00",
					"orderType": "market"
				},
				"notifications": {
					"telegram": {
						"type": "ssm",
						"config": {
							"botTokenPath": "/myapp/telegram/token",
							"chatId": "123456789"
						}
					}
				},
				"flags": {
					"dryRun": true
				}
			}`,
			expected: DCAPayload{
				Version: "v2",
				Exchange: ExchangeConfig{
					Name: "binance",
					Credentials: CredentialSource{
						Type: "ssm",
						Config: map[string]interface{}{
							"apiKeyPath":    "/myapp/binance/apiKey",
							"apiSecretPath": "/myapp/binance/apiSecret",
						},
					},
				},
				Strategy: DCAStrategy{
					Symbol:           "BTC-USDT",
					QuoteAmount:      "10.00",
					BalanceThreshold: "5000.00",
					OrderType:        "market",
				},
				Notifications: NotificationConfig{
					Telegram: &TelegramConfig{
						Type: "ssm",
						Config: map[string]interface{}{
							"botTokenPath": "/myapp/telegram/token",
							"chatId":       "123456789",
						},
					},
				},
				Flags: RuntimeFlags{
					DryRun: true,
				},
			},
		},
		{
			name: "okx_inline_credentials",
			input: `{
				"version": "v2",
				"exchange": {
					"name": "okx",
					"credentials": {
						"type": "inline",
						"config": {
							"apiKey": "test_key",
							"apiSecret": "test_secret",
							"passphrase": "test_passphrase"
						}
					}
				},
				"strategy": {
					"symbol": "ETH-USDT",
					"quoteAmount": "20.50",
					"balanceThreshold": "1000.00"
				},
				"flags": {
					"dryRun": false
				}
			}`,
			expected: DCAPayload{
				Version: "v2",
				Exchange: ExchangeConfig{
					Name: "okx",
					Credentials: CredentialSource{
						Type: "inline",
						Config: map[string]interface{}{
							"apiKey":     "test_key",
							"apiSecret":  "test_secret",
							"passphrase": "test_passphrase",
						},
					},
				},
				Strategy: DCAStrategy{
					Symbol:           "ETH-USDT",
					QuoteAmount:      "20.50",
					BalanceThreshold: "1000.00",
					OrderType:        "market", // default value
				},
				Flags: RuntimeFlags{
					DryRun: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := ParseDCAPayload([]byte(tt.input))
			if err != nil {
				t.Fatalf("ParseDCAPayload() error = %v", err)
			}

			// Compare basic fields
			if payload.Version != tt.expected.Version {
				t.Errorf("Version = %v, want %v", payload.Version, tt.expected.Version)
			}
			if payload.Exchange.Name != tt.expected.Exchange.Name {
				t.Errorf("Exchange.Name = %v, want %v", payload.Exchange.Name, tt.expected.Exchange.Name)
			}
			if payload.Strategy.Symbol != tt.expected.Strategy.Symbol {
				t.Errorf("Strategy.Symbol = %v, want %v", payload.Strategy.Symbol, tt.expected.Strategy.Symbol)
			}
			if payload.Strategy.OrderType != tt.expected.Strategy.OrderType {
				t.Errorf("Strategy.OrderType = %v, want %v", payload.Strategy.OrderType, tt.expected.Strategy.OrderType)
			}
			if payload.Flags.DryRun != tt.expected.Flags.DryRun {
				t.Errorf("Flags.DryRun = %v, want %v", payload.Flags.DryRun, tt.expected.Flags.DryRun)
			}
		})
	}
}

func TestParseDCAPayload_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "invalid_json",
			input:       `{"version": "v2", "exchange":`,
			expectedErr: "invalid JSON",
		},
		{
			name: "missing_version",
			input: `{
				"exchange": {"name": "binance"},
				"strategy": {"symbol": "BTC-USDT", "quoteAmount": "10"}
			}`,
			expectedErr: "version must be",
		},
		{
			name: "wrong_version",
			input: `{
				"version": "v1",
				"exchange": {"name": "binance"},
				"strategy": {"symbol": "BTC-USDT", "quoteAmount": "10"}
			}`,
			expectedErr: "version must be",
		},
		{
			name: "missing_exchange_name",
			input: `{
				"version": "v2",
				"exchange": {"credentials": {"type": "ssm"}},
				"strategy": {"symbol": "BTC-USDT", "quoteAmount": "10"}
			}`,
			expectedErr: "exchange name is required",
		},
		{
			name: "missing_symbol",
			input: `{
				"version": "v2",
				"exchange": {"name": "binance"},
				"strategy": {"quoteAmount": "10"}
			}`,
			expectedErr: "strategy symbol is required",
		},
		{
			name: "missing_quote_amount",
			input: `{
				"version": "v2",
				"exchange": {"name": "binance"},
				"strategy": {"symbol": "BTC-USDT"}
			}`,
			expectedErr: "strategy quoteAmount is required",
		},
		{
			name: "invalid_quote_amount",
			input: `{
				"version": "v2",
				"exchange": {"name": "binance"},
				"strategy": {"symbol": "BTC-USDT", "quoteAmount": "invalid"}
			}`,
			expectedErr: "invalid quoteAmount",
		},
		{
			name: "invalid_balance_threshold",
			input: `{
				"version": "v2",
				"exchange": {"name": "binance"},
				"strategy": {"symbol": "BTC-USDT", "quoteAmount": "10", "balanceThreshold": "invalid"}
			}`,
			expectedErr: "invalid balanceThreshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDCAPayload([]byte(tt.input))
			if err == nil {
				t.Fatal("ParseDCAPayload() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("ParseDCAPayload() error = %v, want to contain %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDCAPayload_ToUnified(t *testing.T) {
	tests := []struct {
		name     string
		payload  DCAPayload
		expected Unified
	}{
		{
			name: "binance_to_unified",
			payload: DCAPayload{
				Version: "v2",
				Exchange: ExchangeConfig{
					Name: "binance",
					Credentials: CredentialSource{
						Type: "ssm",
						Config: map[string]interface{}{
							"apiKeyPath":    "/test/key",
							"apiSecretPath": "/test/secret",
						},
					},
				},
				Strategy: DCAStrategy{
					Symbol:           "BTC-USDT",
					QuoteAmount:      "10.00",
					BalanceThreshold: "5000.00",
					OrderType:        "market",
				},
				Flags: RuntimeFlags{
					DryRun: true,
				},
			},
			expected: Unified{
				Exchange:         "binance",
				Symbol:           "BTC-USDT",
				QuoteAmount:      decimal.RequireFromString("10.00"),
				BalanceThreshold: decimal.RequireFromString("5000.00"),
				DryRun:           true,
				Binance: &struct {
					APIKeyPath, APISecretPath string
				}{
					APIKeyPath:    "/test/key",
					APISecretPath: "/test/secret",
				},
			},
		},
		{
			name: "okx_inline_to_unified",
			payload: DCAPayload{
				Version: "v2",
				Exchange: ExchangeConfig{
					Name: "okx",
					Credentials: CredentialSource{
						Type: "inline",
						Config: map[string]interface{}{
							"apiKey":     "test_key",
							"apiSecret":  "test_secret",
							"passphrase": "test_passphrase",
						},
					},
				},
				Strategy: DCAStrategy{
					Symbol:      "ETH-USDT",
					QuoteAmount: "20.50",
					OrderType:   "market",
				},
				Flags: RuntimeFlags{
					DryRun: false,
				},
			},
			expected: Unified{
				Exchange:         "okx",
				Symbol:           "ETH-USDT",
				QuoteAmount:      decimal.RequireFromString("20.50"),
				BalanceThreshold: decimal.Zero,
				DryRun:           false,
				OKX: &struct {
					APIKeyPath, APISecretPath, PassphrasePath string
				}{},
				OKXInline: &struct {
					APIKey, APISecret, Passphrase string
				}{
					APIKey:     "test_key",
					APISecret:  "test_secret",
					Passphrase: "test_passphrase",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unified, err := tt.payload.ToUnified()
			if err != nil {
				t.Fatalf("ToUnified() error = %v", err)
			}

			if unified.Exchange != tt.expected.Exchange {
				t.Errorf("Exchange = %v, want %v", unified.Exchange, tt.expected.Exchange)
			}
			if unified.Symbol != tt.expected.Symbol {
				t.Errorf("Symbol = %v, want %v", unified.Symbol, tt.expected.Symbol)
			}
			if !unified.QuoteAmount.Equal(tt.expected.QuoteAmount) {
				t.Errorf("QuoteAmount = %v, want %v", unified.QuoteAmount, tt.expected.QuoteAmount)
			}
			if !unified.BalanceThreshold.Equal(tt.expected.BalanceThreshold) {
				t.Errorf("BalanceThreshold = %v, want %v", unified.BalanceThreshold, tt.expected.BalanceThreshold)
			}
			if unified.DryRun != tt.expected.DryRun {
				t.Errorf("DryRun = %v, want %v", unified.DryRun, tt.expected.DryRun)
			}

			// Test exchange-specific fields
			if tt.expected.Binance != nil {
				if unified.Binance == nil {
					t.Error("Expected Binance config, got nil")
				} else {
					if unified.Binance.APIKeyPath != tt.expected.Binance.APIKeyPath {
						t.Errorf("Binance.APIKeyPath = %v, want %v", unified.Binance.APIKeyPath, tt.expected.Binance.APIKeyPath)
					}
				}
			}

			if tt.expected.OKXInline != nil {
				if unified.OKXInline == nil {
					t.Error("Expected OKXInline config, got nil")
				} else {
					if unified.OKXInline.APIKey != tt.expected.OKXInline.APIKey {
						t.Errorf("OKXInline.APIKey = %v, want %v", unified.OKXInline.APIKey, tt.expected.OKXInline.APIKey)
					}
				}
			}
		})
	}
}

func TestDCAPayload_ToUnified_Errors(t *testing.T) {
	tests := []struct {
		name        string
		payload     DCAPayload
		expectedErr string
	}{
		{
			name: "invalid_quote_amount",
			payload: DCAPayload{
				Strategy: DCAStrategy{
					QuoteAmount: "invalid",
				},
			},
			expectedErr: "invalid quoteAmount",
		},
		{
			name: "invalid_balance_threshold",
			payload: DCAPayload{
				Strategy: DCAStrategy{
					QuoteAmount:      "10.00",
					BalanceThreshold: "invalid",
				},
			},
			expectedErr: "invalid balanceThreshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.payload.ToUnified()
			if err == nil {
				t.Fatal("ToUnified() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("ToUnified() error = %v, want to contain %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDefaultOrderType(t *testing.T) {
	input := `{
		"version": "v2",
		"exchange": {
			"name": "binance",
			"credentials": {"type": "ssm", "config": {}}
		},
		"strategy": {
			"symbol": "BTC-USDT",
			"quoteAmount": "10.00"
		},
		"flags": {"dryRun": true}
	}`

	payload, err := ParseDCAPayload([]byte(input))
	if err != nil {
		t.Fatalf("ParseDCAPayload() error = %v", err)
	}

	if payload.Strategy.OrderType != "market" {
		t.Errorf("OrderType = %v, want market", payload.Strategy.OrderType)
	}
}