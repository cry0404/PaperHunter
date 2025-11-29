package acl

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"PaperHunter/internal/core"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"
)

type Adapter struct {
	config     *Config
	httpClient *http.Client
}

func NewAdapter(config *Config) (*Adapter, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	client := core.NewHTTPClient(int(config.Timeout.Seconds()), config.Proxy)
	return &Adapter{
		config:     config,
		httpClient: client,
	}, nil
}

func (a *Adapter) Name() string { return "acl" }

func (a *Adapter) GetConfig() platform.Config { return a.config }

func (a *Adapter) Search(ctx context.Context, q platform.Query) (platform.Result, error) {
	if a.config.UseRSS {
		logger.Info("[ACL] 使用 RSS 模式获取最新论文")
		return a.searchViaRSS(ctx, q)
	} else if a.config.UseBibTeX {
		logger.Info("[ACL] 使用 BibTeX 模式获取全量论文")
		return a.searchViaBibTeX(ctx, q)
	}

	return platform.Result{}, fmt.Errorf("错误的配置，请检查 config.yaml 中的配置")
}

func (a *Adapter) request(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/xml,application/rss+xml,text/plain")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}
