package embedding

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/embedding/openai"

	"PaperHunter/internal/models"
)

type EmbedderConfig struct {
	BaseURL   string `mapstructure:"baseurl" yaml:"baseurl"`
	APIKey    string `mapstructure:"apikey" yaml:"apikey"`
	ModelName string `mapstructure:"model" yaml:"model"`
	Dim       int    `mapstructure:"dim" yaml:"dim"`
}

type Service interface {
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	ModelName() string
	Dim() int
}

type openaiAdapter struct {
	cfg   EmbedderConfig
	inner *openai.Embedder
}

func New(cfg EmbedderConfig) (Service, error) {
	if cfg.ModelName == "" {
		cfg.ModelName = "text-embedding-3-small"
	}
	if cfg.Dim == 0 {
		cfg.Dim = 1536
	}

	if cfg.APIKey == "" {
		return &noopService{cfg: cfg}, nil
	}

	inner, err := openai.NewEmbedder(context.Background(), &openai.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.ModelName,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("创建向量服务失败: %w", err)
	}
	return &openaiAdapter{cfg: cfg, inner: inner}, nil
}

func (a *openaiAdapter) ModelName() string { return a.cfg.ModelName }
func (a *openaiAdapter) Dim() int          { return a.cfg.Dim }

func (a *openaiAdapter) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("query text is empty")
	}
	vecs, err := a.inner.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}
	return toFloat32(vecs[0]), nil
}

func (a *openaiAdapter) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	filtered := make([]string, 0, len(texts))
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t != "" {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}
	vecs64, err := a.inner.EmbedStrings(ctx, filtered)
	if err != nil {
		return nil, err
	}
	vecs32 := make([][]float32, len(vecs64))
	for i, v := range vecs64 {
		vecs32[i] = toFloat32(v)
	}
	return vecs32, nil
}

// noopService 空实现，用于没有配置 APIKey 时
type noopService struct {
	cfg EmbedderConfig
}

func (n *noopService) ModelName() string { return n.cfg.ModelName }
func (n *noopService) Dim() int          { return n.cfg.Dim }
func (n *noopService) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embedder not configured (missing APIKey)")
}
func (n *noopService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("embedder not configured (missing APIKey)")
}

// toFloat32 转换 float64 到 float32（SQLite BLOB 存储用）
func toFloat32(v []float64) []float32 {
	ret := make([]float32, len(v))
	for i, val := range v {
		ret[i] = float32(val)
	}
	return ret
}

// BuildEmbeddingText 生成用于向量化的文本（标题 + 摘要）
func BuildEmbeddingText(p *models.Paper) string {
	title := strings.TrimSpace(p.Title)
	abs := strings.TrimSpace(p.Abstract)
	if abs == "" {
		return title
	}
	return fmt.Sprintf("%s\n\n%s", title, abs)
}
