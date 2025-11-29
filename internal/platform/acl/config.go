package acl

import (
	"fmt"
	"time"
)

type Config struct {
	BaseURL   string        `mapstructure:"base_url" yaml:"base_url"`
	Timeout   time.Duration `mapstructure:"timeout" yaml:"timeout"`
	Proxy     string        `mapstructure:"proxy" yaml:"proxy"`
	Step      int           `mapstructure:"step" yaml:"step"`
	UseRSS    bool          `mapstructure:"use_rss" yaml:"use_rss"`       // true: 使用 RSS 获取最新 1000 篇, false: 使用 BibTeX 全量
	UseBibTeX bool          `mapstructure:"use_bibtex" yaml:"use_bibtex"` // 是否使用带摘要的 BibTeX 文件
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:   "https://aclanthology.org",
		Timeout:   30 * time.Second,
		Step:      100,
		UseRSS:    true,  // 默认使用 RSS 模式
		UseBibTeX: false, // 默认不使用 BibTeX 全量模式
	}
}

func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.Step <= 0 {
		return fmt.Errorf("step must be positive")
	}
	return nil
}
