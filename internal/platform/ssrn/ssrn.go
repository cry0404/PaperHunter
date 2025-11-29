package ssrn

import (
	"PaperHunter/internal/core"
	"PaperHunter/internal/platform"
)

// New 供外部创建 SSRN 平台实例
func New(config *Config) (platform.Platform, error) {
	return NewAdapter(config)
}

// 在包初始化时注册 Provider
func init() {
	core.MustRegister(core.Provider{
		Name: "ssrn",
		New: func(cfg platform.Config) (platform.Platform, error) {
			c, _ := cfg.(*Config)
			if c == nil {
				c = DefaultConfig()
			}
			return New(c)
		},
		DefaultConfig: func() platform.Config { return DefaultConfig() },
	})
}
