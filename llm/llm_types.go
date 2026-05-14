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
	APIKey         string  `toml:"api_key" json:"api_key" mapstructure:"api_key"`
	Temperature    float64 `toml:"temperature" json:"temperature" mapstructure:"temperature"`
	MaxConsumption float64 `toml:"max_consumption" json:"max_consumption" mapstructure:"max_consumption"`
	LastBalance    float64 `toml:"last_balance" json:"last_balance" mapstructure:"last_balance"`
	Timeout        int     `toml:"timeout" json:"timeout" mapstructure:"timeout"`
}

type InputForLLM string
