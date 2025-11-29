package arxiv

import (
	"fmt"

)

type Config struct {
	UseAPI  bool   `mapstructure:"use_api" yaml:"use_api"`   // 使用官方 API（true）或网页搜索（false）
	Proxy   string `mapstructure:"proxy" yaml:"proxy"`       // 代理地址，如 "http://127.0.0.1:7890"
	Step    int    `mapstructure:"step" yaml:"step"`         // 每页数量（1-200）
	Timeout int    `mapstructure:"timeout" yaml:"timeout"`   // 超时时间（秒）

	APIBase string `mapstructure:"api_base" yaml:"api_base"` // API 基础 URL
	

	WebBase string `mapstructure:"web_base" yaml:"web_base"` // 网页基础 URL
}


func DefaultConfig() *Config {
	return &Config{
		UseAPI:  false,
		Step:    50,
		Timeout: 30,
		APIBase: "https://export.arxiv.org/api/query",
		WebBase: "https://arxiv.org/search/advanced",
	}
}


func (c *Config) Validate() error {
	if c.Step <= 0 || c.Step > 200 {
		return fmt.Errorf("step must be between 1 and 200, got %d", c.Step)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %d", c.Timeout)
	}
	if c.APIBase == "" {
		return fmt.Errorf("api_base cannot be empty")
	}
	if c.WebBase == "" {
		return fmt.Errorf("web_base cannot be empty")
	}
	return nil
}
