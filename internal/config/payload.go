package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// New unified payload structure
type DCAPayload struct {
	Version       string              `json:"version"`
	Exchange      ExchangeConfig      `json:"exchange"`
	Strategy      DCAStrategy         `json:"strategy"`
	Notifications NotificationConfig  `json:"notifications"`
	Flags         RuntimeFlags        `json:"flags"`
}

type ExchangeConfig struct {
	Name        string          `json:"name"`        // "binance", "okx"
	Credentials CredentialSource `json:"credentials"` // unified credential source
	Region      string          `json:"region,omitempty"` // optional, for different regions
}

type DCAStrategy struct {
	Symbol           string `json:"symbol"`           // "BTC-USDT"
	QuoteAmount      string `json:"quoteAmount"`      // "10.00"
	BalanceThreshold string `json:"balanceThreshold"` // "5000.00"
	OrderType        string `json:"orderType"`        // "market", "limit"
}

type CredentialSource struct {
	Type   string                 `json:"type"`   // "inline", "env", "ssm"
	Config map[string]interface{} `json:"config"` // flexible configuration
}

type NotificationConfig struct {
	Telegram *TelegramConfig `json:"telegram,omitempty"`
}

type TelegramConfig struct {
	Type   string                 `json:"type"`   // "inline", "env", "ssm"
	Config map[string]interface{} `json:"config"` // flexible configuration
}

type RuntimeFlags struct {
	DryRun bool `json:"dryRun"`
}

// Legacy PayloadV2 struct (keep for backward compatibility)
type PayloadV2 struct {
	Version  string `json:"version"`
	Exchange string `json:"exchange"`
	DCA      struct {
		Symbol           string `json:"symbol"`
		TargetAsset      string `json:"targetAsset"`
		OrderCurrency    string `json:"orderCurrency"`
		QuoteAmount      string `json:"quoteAmount"`
		BalanceThreshold string `json:"balanceThreshold"`
	} `json:"dca"`
	Credentials struct {
		OKX *struct {
			APIKeyPath     string `json:"apiKeyPath"`
			APISecretPath  string `json:"apiSecretPath"`
			PassphrasePath string `json:"passphrasePath"`
			Inline         *struct {
				APIKey     string `json:"apiKey"`
				APISecret  string `json:"apiSecret"`
				Passphrase string `json:"passphrase"`
			} `json:"inline"`
			Env *struct {
				APIKeyEnv     string `json:"apiKeyEnv"`
				APISecretEnv  string `json:"apiSecretEnv"`
				PassphraseEnv string `json:"passphraseEnv"`
			} `json:"env"`
		} `json:"okx"`
		Binance *struct {
			APIKeyPath    string `json:"apiKeyPath"`
			APISecretPath string `json:"apiSecretPath"`
			Inline        *struct {
				APIKey    string `json:"apiKey"`
				APISecret string `json:"apiSecret"`
			} `json:"inline"`
			Env *struct {
				APIKeyEnv    string `json:"apiKeyEnv"`
				APISecretEnv string `json:"apiSecretEnv"`
			} `json:"env"`
		} `json:"binance"`
	} `json:"credentials"`
	Notifications struct {
		Telegram *struct {
			BotTokenPath string `json:"botTokenPath"`
			ChatID       string `json:"chatID"`
			Sink         string `json:"sink"` // "stdout"|default telegram
			Inline       *struct {
				BotToken string `json:"botToken"`
			} `json:"inline"`
			Env *struct {
				BotTokenEnv string `json:"botTokenEnv"`
			} `json:"env"`
		} `json:"telegram"`
	} `json:"notifications"`
	Flags struct {
		DryRun bool `json:"dryRun"`
	} `json:"flags"`
}

type Unified struct {
	Exchange         string
	Symbol           string
	QuoteAmount      decimal.Decimal
	BalanceThreshold decimal.Decimal
	DryRun           bool

	OKX *struct {
		APIKeyPath, APISecretPath, PassphrasePath string
	}
	OKXInline *struct {
		APIKey, APISecret, Passphrase string
	}
	Binance *struct {
		APIKeyPath, APISecretPath string
	}
	Telegram *struct {
		BotTokenPath, ChatID, Sink string
	}
}

// Parse new DCAPayload format
func ParseDCAPayload(raw []byte) (*DCAPayload, error) {
	var payload DCAPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	
	if strings.ToLower(payload.Version) != "v2" {
		return nil, fmt.Errorf(`version must be "v2"`)
	}
	
	// Validate exchange name
	if payload.Exchange.Name == "" {
		return nil, fmt.Errorf("exchange name is required")
	}
	
	// Validate strategy
	if payload.Strategy.Symbol == "" {
		return nil, fmt.Errorf("strategy symbol is required")
	}
	
	if payload.Strategy.QuoteAmount == "" {
		return nil, fmt.Errorf("strategy quoteAmount is required")
	}
	
	// Validate quote amount is a valid decimal
	if _, err := decimal.NewFromString(payload.Strategy.QuoteAmount); err != nil {
		return nil, fmt.Errorf("invalid quoteAmount: %w", err)
	}
	
	// Validate balance threshold if provided
	if payload.Strategy.BalanceThreshold != "" {
		if _, err := decimal.NewFromString(payload.Strategy.BalanceThreshold); err != nil {
			return nil, fmt.Errorf("invalid balanceThreshold: %w", err)
		}
	}
	
	// Set default order type
	if payload.Strategy.OrderType == "" {
		payload.Strategy.OrderType = "market"
	}
	
	return &payload, nil
}

// Convert DCAPayload to Unified for backward compatibility
func (p *DCAPayload) ToUnified() (Unified, error) {
	qa, err := decimal.NewFromString(p.Strategy.QuoteAmount)
	if err != nil {
		return Unified{}, fmt.Errorf("invalid quoteAmount: %w", err)
	}
	
	bt := decimal.Zero
	if p.Strategy.BalanceThreshold != "" {
		bt, err = decimal.NewFromString(p.Strategy.BalanceThreshold)
		if err != nil {
			return Unified{}, fmt.Errorf("invalid balanceThreshold: %w", err)
		}
	}
	
	unified := Unified{
		Exchange:         strings.ToLower(p.Exchange.Name),
		Symbol:           strings.ToUpper(p.Strategy.Symbol),
		QuoteAmount:      qa,
		BalanceThreshold: bt,
		DryRun:           p.Flags.DryRun,
	}
	
	// Handle credentials based on exchange type and credential source
	if err := p.populateUnifiedCredentials(&unified); err != nil {
		return Unified{}, err
	}
	
	// Handle telegram notifications
	if p.Notifications.Telegram != nil {
		unified.Telegram = &struct {
			BotTokenPath, ChatID, Sink string
		}{}
		
		if chatID, ok := p.Notifications.Telegram.Config["chatId"].(string); ok {
			unified.Telegram.ChatID = chatID
		}
		
		switch p.Notifications.Telegram.Type {
		case "ssm":
			if path, ok := p.Notifications.Telegram.Config["botTokenPath"].(string); ok {
				unified.Telegram.BotTokenPath = path
			}
		case "inline":
			// For inline, we'll need to handle this differently in the future
		case "env":
			// For env, we'll need to handle this differently in the future
		}
	}
	
	return unified, nil
}

func (p *DCAPayload) populateUnifiedCredentials(unified *Unified) error {
	switch strings.ToLower(p.Exchange.Name) {
	case "binance":
		unified.Binance = &struct {
			APIKeyPath, APISecretPath string
		}{}
		
		switch p.Exchange.Credentials.Type {
		case "ssm":
			if keyPath, ok := p.Exchange.Credentials.Config["apiKeyPath"].(string); ok {
				unified.Binance.APIKeyPath = keyPath
			}
			if secretPath, ok := p.Exchange.Credentials.Config["apiSecretPath"].(string); ok {
				unified.Binance.APISecretPath = secretPath
			}
		}
		
	case "okx":
		unified.OKX = &struct {
			APIKeyPath, APISecretPath, PassphrasePath string
		}{}
		
		switch p.Exchange.Credentials.Type {
		case "ssm":
			if keyPath, ok := p.Exchange.Credentials.Config["apiKeyPath"].(string); ok {
				unified.OKX.APIKeyPath = keyPath
			}
			if secretPath, ok := p.Exchange.Credentials.Config["apiSecretPath"].(string); ok {
				unified.OKX.APISecretPath = secretPath
			}
			if passphrasePath, ok := p.Exchange.Credentials.Config["passphrasePath"].(string); ok {
				unified.OKX.PassphrasePath = passphrasePath
			}
		}
		
		if p.Exchange.Credentials.Type == "inline" {
			unified.OKXInline = &struct {
				APIKey, APISecret, Passphrase string
			}{}
			
			if key, ok := p.Exchange.Credentials.Config["apiKey"].(string); ok {
				unified.OKXInline.APIKey = key
			}
			if secret, ok := p.Exchange.Credentials.Config["apiSecret"].(string); ok {
				unified.OKXInline.APISecret = secret
			}
			if passphrase, ok := p.Exchange.Credentials.Config["passphrase"].(string); ok {
				unified.OKXInline.Passphrase = passphrase
			}
		}
	}
	
	return nil
}

func ParseUnifiedV2(raw []byte) (Unified, error) {
	var v2 PayloadV2
	if err := json.Unmarshal(raw, &v2); err != nil {
		return Unified{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if strings.ToLower(v2.Version) != "v2" {
		return Unified{}, fmt.Errorf(`version must be "v2"`)
	}
	ex := strings.ToLower(strings.TrimSpace(v2.Exchange))
	if ex == "" {
		return Unified{}, fmt.Errorf("exchange is required")
	}

	symbol := strings.ToUpper(strings.TrimSpace(v2.DCA.Symbol))
	if symbol == "" {
		if v2.DCA.TargetAsset == "" || v2.DCA.OrderCurrency == "" {
			return Unified{}, fmt.Errorf("dca.symbol or (dca.targetAsset+orderCurrency) required")
		}
		symbol = strings.ToUpper(v2.DCA.TargetAsset) + "-" + strings.ToUpper(v2.DCA.OrderCurrency)
	}

	qa, err := decimal.NewFromString(v2.DCA.QuoteAmount)
	if err != nil || !qa.IsPositive() {
		return Unified{}, fmt.Errorf("dca.quoteAmount invalid: %q", v2.DCA.QuoteAmount)
	}
	bt := decimal.Zero
	if s := strings.TrimSpace(v2.DCA.BalanceThreshold); s != "" {
		bt, err = decimal.NewFromString(s)
		if err != nil {
			return Unified{}, fmt.Errorf("dca.balanceThreshold invalid: %q", s)
		}
	}

	u := Unified{
		Exchange:         ex,
		Symbol:           symbol,
		QuoteAmount:      qa,
		BalanceThreshold: bt,
		DryRun:           v2.Flags.DryRun,
	}
	if v2.Credentials.OKX != nil {
		u.OKX = &struct {
			APIKeyPath, APISecretPath, PassphrasePath string
		}{
			v2.Credentials.OKX.APIKeyPath,
			v2.Credentials.OKX.APISecretPath,
			v2.Credentials.OKX.PassphrasePath,
		}
		if v2.Credentials.OKX.Inline != nil {
			u.OKXInline = &struct {
				APIKey, APISecret, Passphrase string
			}{
				v2.Credentials.OKX.Inline.APIKey,
				v2.Credentials.OKX.Inline.APISecret,
				v2.Credentials.OKX.Inline.Passphrase,
			}
		}
	}
	if v2.Credentials.Binance != nil {
		u.Binance = &struct {
			APIKeyPath, APISecretPath string
		}{
			v2.Credentials.Binance.APIKeyPath,
			v2.Credentials.Binance.APISecretPath,
		}
	}
	if v2.Notifications.Telegram != nil {
		u.Telegram = &struct {
			BotTokenPath, ChatID, Sink string
		}{
			v2.Notifications.Telegram.BotTokenPath,
			v2.Notifications.Telegram.ChatID,
			strings.ToLower(strings.TrimSpace(v2.Notifications.Telegram.Sink)),
		}
	}
	return u, nil
}
