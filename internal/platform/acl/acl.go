package acl

import (
	"PaperHunter/internal/core"
	"PaperHunter/internal/platform"
)

func New(config *Config) (platform.Platform, error) {
	return NewAdapter(config)
}

func init() {
	core.MustRegister(core.Provider{
		Name: "acl",
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
