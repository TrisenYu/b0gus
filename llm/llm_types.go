package llm

import "github.com/tmc/langchaingo/llms"

/// Last modified at 2026/04/14 星期二 15:35:42

type LLMconfig struct {
	model llms.Model
	// ModelName represents the model you select defined in some row of configuration
	ModelName  string `toml:"model_name" json:"model_name" mapstructure:"model_name"`
	RolePrompt string `toml:"role_prompt" json:"role_prompt" mapstructure:"role_prompt"`
	// BaseURL is the llm provider site-url
	BaseURL string `toml:"base_url" json:"base_url" mapstructure:"base_url"`
	// BalanceAPI is the api for querying the balance detail
	BalanceAPI string `toml:"balance_api" json:"balance_api" mapstructure:"balance_api"`
	// APIKey is your access token typically used in headers: `Authorization: Bearer APIKey`
	APIKey      string  `toml:"api_key" json:"api_key" mapstructure:"api_key"`
	Temperature float64 `toml:"temperature" json:"temperature" mapstructure:"temperature"`
	Timeout     int     `toml:"timeout" json:"timeout" mapstructure:"timeout"`

	// as a threshold
	MaxConsumption float64 `toml:"max_consumption" json:"max_consumption" mapstructure:"max_consumption"`
	LastBalance    float64 `toml:"last_balance" json:"last_balance" mapstructure:"last_balance"`
}

type balanceInfo struct {
	Currency        string `toml:"currency" json:"currency" mapstructure:"currency"`
	TotalBalance    string `toml:"total_balance" json:"total_balance" mapstructure:"total_balance"`
	GrantedBalance  string `toml:"granted_balance" json:"granted_balance" mapstructure:"granted_balance"`
	ToppedUpBalance string `toml:"topped_up_balance" json:"topped_up_balance" mapstructure:"topped_up_balance"`
}

// BalanceOfDeepSeek
//
//	{
//		"is_available": true,
//		"balance_infos": [{
//			"currency": "CNY",
//			"total_balance": "110.00",
//			"granted_balance": "10.00",
//			"topped_up_balance": "100.00"
//		}]
//	}
type BalanceOfDeepSeek struct {
	IsAvailable  bool          `toml:"is_available" json:"is_available" mapstructure:"is_available"`
	BalanceInfos []balanceInfo `toml:"balance_infos" json:"balance_infos" mapstructure:"balance_infos"`
}

type BalanceOfGoogle struct {
}
type InputForLLM string
