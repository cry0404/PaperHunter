package hyde

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"

	"PaperHunter/config"
	"PaperHunter/pkg/logger"

	embopenai "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// HypotheticalPaper 虚拟论文结构
type HypotheticalPaper struct {
	Title    string `json:"title"`
	Abstract string `json:"abstract"`
}

type Service interface {
	GenerateHypotheticalPaper(ctx context.Context, userQuery string) (*HypotheticalPaper, error)
}

type hydeService struct {
	model    *openai.ChatModel
	embedder *embopenai.Embedder
}

func New(cfg config.LLMConfig) (Service, error) {
	if cfg.APIKey == "" {
		logger.Warn("LLM API Key 未配置，使用简单的 HyDE 回退方案")
		return nil, nil
	}

	ctx := context.Background()
	temp := float32(0.3)

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      cfg.APIKey,
		Model:       cfg.ModelName,
		BaseURL:     cfg.BaseURL,
		Temperature: &temp,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	// 尝试创建 embedding 客户端用于候选选优（失败则降级为词重合评分）
	embedModel := cfg.ModelName
	if embedModel == "" || !strings.Contains(embedModel, "embedding") {
		embedModel = "text-embedding-3-small"
	}
	embedder, err := embopenai.NewEmbedder(ctx, &embopenai.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		Model:   embedModel,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		logger.Warn("创建 embedding 客户端失败，选优将使用词重合: %v", err)
	}

	return &hydeService{model: model, embedder: embedder}, nil
}

func (s *hydeService) GenerateHypotheticalPaper(ctx context.Context, userQuery string) (*HypotheticalPaper, error) {
	if strings.TrimSpace(userQuery) == "" {
		return nil, fmt.Errorf("用户查询不能为空")
	}

	prompt := buildHyDEPrompt(userQuery)

	logger.Info("使用 HyDE 生成虚拟论文，用户查询: %s", userQuery)

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: getSystemPrompt(),
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	const candidateAttempts = 3
	candidates := make([]*HypotheticalPaper, 0, candidateAttempts)
	seen := make(map[string]struct{})

	for i := 0; i < candidateAttempts; i++ {
		resp, err := s.model.Generate(ctx, messages)
		if err != nil {
			logger.Warn("LLM 生成失败(第 %d 次): %v", i+1, err)
			continue
		}

		if resp == nil || resp.Content == "" {
			logger.Warn("LLM 返回空响应(第 %d 次)", i+1)
			continue
		}

		paper, err := parseHyDEResponse(resp.Content)
		if err != nil {
			logger.Warn("解析 LLM 响应失败(第 %d 次): %v", i+1, err)
			continue
		}

		key := normalizeKey(paper.Title + "|" + paper.Abstract)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, paper)
	}

	if len(candidates) == 0 {
		logger.Warn("HyDE 未得到有效候选，使用降级结果")
		return fallbackHypotheticalPaper(userQuery), nil
	}

	best := selectBestCandidate(ctx, s.embedder, candidates, userQuery)
	if best == nil {
		logger.Warn("HyDE 候选选优失败，使用降级结果")
		return fallbackHypotheticalPaper(userQuery), nil
	}

	return best, nil
}

func fallbackHypotheticalPaper(userQuery string) *HypotheticalPaper {
	q := strings.TrimSpace(userQuery)
	if q == "" {
		q = "generic research topic"
	}
	return &HypotheticalPaper{
		Title:    q,
		Abstract: q, // 降级时直接使用原始查询，不再模拟生成
	}
}

func getSystemPrompt() string {
	return `You are an expert academic paper generator for information retrieval. Your task is to generate a hypothetical academic paper (title and abstract) that will be used to search for similar papers.

CRITICAL RULES:
1. The title MUST contain the EXACT keywords from the user's query
2. The abstract MUST heavily feature the user's keywords (repeat them naturally 3-5 times)
3. Keep the paper focused on the EXACT topic the user mentioned - do NOT drift to related but different topics

The generated paper should:
1. Title: Include the user's exact keywords/phrases prominently
2. Abstract (150-200 words):
   - Start by mentioning the exact topic from user's query
   - Describe common approaches in this specific area
   - Mention typical challenges and solutions
   - Use terminology that papers in this field would use

Output Format:
You MUST respond with a valid JSON object:
{
  "title": "Your generated paper title here",
  "abstract": "Your generated abstract here..."
}

Important:
- Keep the user's original keywords in the title and abstract
- Do NOT over-specialize or narrow down the topic
- Do NOT include any text outside the JSON object`
}

func buildHyDEPrompt(userQuery string) string {
	return fmt.Sprintf(`Generate a hypothetical academic paper about: "%s"

IMPORTANT: 
- The title MUST include "%s" or very similar wording
- The abstract should be about "%s" specifically, not a narrow sub-topic

Output ONLY a JSON object with "title" and "abstract" fields.`, userQuery, userQuery, userQuery)
}

func parseHyDEResponse(content string) (*HypotheticalPaper, error) {
	// 清理可能获得的 markdown 的格式输出
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	startIdx := strings.Index(content, "{")
	endIdx := strings.LastIndex(content, "}")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		content = content[startIdx : endIdx+1]
	}

	var paper HypotheticalPaper
	if err := json.Unmarshal([]byte(content), &paper); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	if paper.Title == "" || paper.Abstract == "" {
		return nil, fmt.Errorf("生成的论文标题或摘要为空")
	}

	return &paper, nil
}

// 选优：优先使用 embedding 余弦相似度，失败则回退到词重合
func selectBestCandidate(ctx context.Context, embedder *embopenai.Embedder, candidates []*HypotheticalPaper, userQuery string) *HypotheticalPaper {
	if len(candidates) == 0 {
		return nil
	}

	// 尝试 embedding 评分
	if embedder != nil {
		queryVec, err := embedOnce(ctx, embedder, userQuery)
		if err == nil && len(queryVec) > 0 {
			best := candidates[0]
			bestScore := cosine(queryVec, embedOnceOrZero(ctx, embedder, best))

			for _, c := range candidates[1:] {
				score := cosine(queryVec, embedOnceOrZero(ctx, embedder, c))
				if score > bestScore {
					bestScore = score
					best = c
				}
			}
			return best
		}
		logger.Warn("Embedding 选优失败，回退到词重合")
	}

	// 词重合回退
	queryTokens := tokenSet(userQuery)
	best := candidates[0]
	bestScore := tokenOverlapScore(best, queryTokens)

	for _, c := range candidates[1:] {
		score := tokenOverlapScore(c, queryTokens)
		if score > bestScore {
			bestScore = score
			best = c
		}
	}
	return best
}

func embedOnce(ctx context.Context, embedder *embopenai.Embedder, text string) ([]float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text for embedding")
	}
	vecs, err := embedder.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}
	return vecs[0], nil
}

func embedOnceOrZero(ctx context.Context, embedder *embopenai.Embedder, p *HypotheticalPaper) []float64 {
	if p == nil {
		return nil
	}
	text := strings.TrimSpace(p.Title + "\n\n" + p.Abstract)
	vec, err := embedOnce(ctx, embedder, text)
	if err != nil {
		return nil
	}
	return vec
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrt(na) * sqrt(nb))
}

func sqrt(v float64) float64 {
	// 简单调用标准库 math.Sqrt，避免额外依赖
	return math.Sqrt(v)
}

func tokenOverlapScore(p *HypotheticalPaper, queryTokens map[string]struct{}) float64 {
	if p == nil {
		return -1
	}
	if len(queryTokens) == 0 {
		return 0
	}
	titleTokens := tokenSet(p.Title)
	absTokens := tokenSet(p.Abstract)

	combined := make(map[string]struct{}, len(titleTokens)+len(absTokens))
	for t := range titleTokens {
		combined[t] = struct{}{}
	}
	for t := range absTokens {
		combined[t] = struct{}{}
	}

	intersect := 0
	for t := range combined {
		if _, ok := queryTokens[t]; ok {
			intersect++
		}
	}
	union := len(combined) + len(queryTokens) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}

func tokenSet(text string) map[string]struct{} {
	normalized := normalizeKey(text)
	tokens := strings.Fields(normalized)
	set := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		set[t] = struct{}{}
	}
	return set
}

func normalizeKey(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, s)
	return strings.Join(strings.Fields(s), " ")
}
