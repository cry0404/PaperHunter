package arxiv

import (
	"PaperHunter/internal/core"
	"PaperHunter/internal/platform"
)

func New(config *Config) (platform.Platform, error) {
	return NewAdapter(config)
}

func init() {

	core.MustRegister(core.Provider{
		Name: "arxiv",
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

//如果添加下载功能可以使用
/*
func PDFUrl(arxivID string) string {
	return "https://arxiv.org/pdf/" + arxivID
}


func PapersCoolUrl(arxivID string) string {
	return "https://papers.cool/arxiv/" + arxivID
}*/
