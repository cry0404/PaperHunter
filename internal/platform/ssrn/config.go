package ssrn

import (
	"fmt"
	"time"

	"PaperHunter/internal/platform"
)

// Config 定义 SSRN 平台的基础配置
type Config struct {
	// HTTP 行为
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`
	Proxy   string        `mapstructure:"proxy" yaml:"proxy"`

	// 站点与抓取参数
	BaseURL            string  `mapstructure:"base_url" yaml:"base_url"`
	PageSize           int     `mapstructure:"page_size" yaml:"page_size"`
	MaxPages           int     `mapstructure:"max_pages" yaml:"max_pages"`
	RateLimitPerSecond float64 `mapstructure:"rate_limit_per_second" yaml:"rate_limit_per_second"`

	// 排序: AB_Date_D(按时间降序) / AB_Date_A / relevance 等
	Sort string `mapstructure:"sort" yaml:"sort"`
}

// DefaultConfig 返回 SSRN 的默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:            30 * time.Second,
		BaseURL:            "https://papers.ssrn.com",
		PageSize:           20,
		MaxPages:           3,
		RateLimitPerSecond: 0.2,
		Sort:               "AB_Date_D",
	}
}

// Validate 校验配置
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("invalid timeout: %v", c.Timeout)
	}
	if c.PageSize <= 0 || c.PageSize > 100 {
		return fmt.Errorf("invalid page_size: %d", c.PageSize)
	}
	if c.MaxPages < 0 || c.MaxPages > 50 {
		return fmt.Errorf("invalid max_pages: %d", c.MaxPages)
	}
	if c.RateLimitPerSecond <= 0 {
		return fmt.Errorf("invalid rate_limit_per_second: %v", c.RateLimitPerSecond)
	}
	if c.BaseURL == "" {
		return fmt.Errorf("base_url required")
	}
	return nil
}

// 确保实现 platform.Config 接口
var _ platform.Config = (*Config)(nil)
