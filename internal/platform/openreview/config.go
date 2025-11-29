package openreview

import "fmt"

// Config OpenReview 平台配置
type Config struct {
	APIBase string `mapstructure:"api_base" yaml:"api_base"` // API 地址
	Proxy   string `mapstructure:"proxy" yaml:"proxy"`
	Timeout int    `mapstructure:"timeout" yaml:"timeout"`
}

func DefaultConfig() *Config {
	return &Config{
		APIBase: "https://api2.openreview.net",
		Timeout: 30,
	}
}



func (c *Config) Validate() error {
	if c.APIBase == "" {
		return fmt.Errorf("api_base 不能为空")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout 不能为负")
	}
	return nil
}