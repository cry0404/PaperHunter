package hyde

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"PaperHunter/config"
	"PaperHunter/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// HypotheticalPaper 虚拟论文结构
type HypotheticalPaper struct {
	Title    string `json:"title"`
	Abstract string `json:"abstract"`
}

// Service HyDE 服务接口
type Service interface {
	// GenerateHypotheticalPaper 基于用户查询生成虚拟论文
	GenerateHypotheticalPaper(ctx context.Context, userQuery string) (*HypotheticalPaper, error)
}

// hydeService HyDE 服务实现
type hydeService struct {
	model *openai.ChatModel
}

// New 创建 HyDE 服务
func New(cfg config.LLMConfig) (Service, error) {
	if cfg.APIKey == "" {
		logger.Warn("LLM API Key 未配置，使用简单的 HyDE 回退方案")
		return &fallbackService{}, nil
	}

	ctx := context.Background()
	temp := float32(0.3) // 稍高的温度以获得更多样的输出

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      cfg.APIKey,
		Model:       cfg.ModelName,
		BaseURL:     cfg.BaseURL,
		Temperature: &temp,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	return &hydeService{model: model}, nil
}

// GenerateHypotheticalPaper 使用 LLM 生成虚拟论文
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

	resp, err := s.model.Generate(ctx, messages)
	if err != nil {
		logger.Error("LLM 生成失败: %v", err)
		return nil, fmt.Errorf("LLM 生成失败: %w", err)
	}

	if resp == nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 返回空响应")
	}

	// 解析 JSON 响应
	paper, err := parseHyDEResponse(resp.Content)
	if err != nil {
		logger.Warn("解析 LLM 响应失败，尝试提取文本: %v", err)
		// 尝试从纯文本中提取
		paper = extractFromText(resp.Content, userQuery)
	}

	logger.Info("HyDE 生成成功 - 标题: %s", paper.Title)
	logger.Debug("HyDE 生成的摘要: %s", paper.Abstract)

	return paper, nil
}

// getSystemPrompt 返回系统提示词
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

// buildHyDEPrompt 构建 HyDE 提示词
func buildHyDEPrompt(userQuery string) string {
	return fmt.Sprintf(`Generate a hypothetical academic paper about: "%s"

IMPORTANT: 
- The title MUST include "%s" or very similar wording
- The abstract should be about "%s" specifically, not a narrow sub-topic

Output ONLY a JSON object with "title" and "abstract" fields.`, userQuery, userQuery, userQuery)
}

// parseHyDEResponse 解析 LLM 响应
func parseHyDEResponse(content string) (*HypotheticalPaper, error) {
	// 清理响应，移除可能的 markdown 代码块标记
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// 尝试找到 JSON 对象的起始和结束位置
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

// extractFromText 从纯文本中提取标题和摘要（回退方案）
func extractFromText(content string, userQuery string) *HypotheticalPaper {
	lines := strings.Split(content, "\n")

	var title, abstract string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "title:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
			title = strings.TrimSpace(strings.TrimPrefix(title, "title:"))
		} else if strings.HasPrefix(strings.ToLower(line), "abstract:") {
			// 收集摘要（可能跨多行）
			abstract = strings.TrimSpace(strings.TrimPrefix(line, "Abstract:"))
			abstract = strings.TrimSpace(strings.TrimPrefix(abstract, "abstract:"))
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" || strings.HasPrefix(strings.ToLower(nextLine), "title:") {
					break
				}
				abstract += " " + nextLine
			}
		}
	}

	// 如果仍然没有提取到，使用整个内容作为摘要
	if title == "" {
		title = generateFallbackTitle(userQuery)
	}
	if abstract == "" {
		abstract = content
		if len(abstract) > 500 {
			abstract = abstract[:500] + "..."
		}
	}

	return &HypotheticalPaper{
		Title:    title,
		Abstract: abstract,
	}
}

// fallbackService 回退服务（当 LLM 不可用时）
type fallbackService struct{}

// GenerateHypotheticalPaper 简单的回退实现
func (s *fallbackService) GenerateHypotheticalPaper(ctx context.Context, userQuery string) (*HypotheticalPaper, error) {
	if strings.TrimSpace(userQuery) == "" {
		return &HypotheticalPaper{
			Title:    "Recent Advances in Computer Science Research",
			Abstract: "This paper surveys recent developments in computer science, covering emerging trends and methodologies. We analyze state-of-the-art approaches and discuss future research directions.",
		}, nil
	}

	// 提取关键词用于生成标题
	keywords := extractKeywords(userQuery)
	title := generateFallbackTitle(userQuery)
	abstract := generateFallbackAbstract(userQuery, keywords)

	return &HypotheticalPaper{
		Title:    title,
		Abstract: abstract,
	}, nil
}

// extractKeywords 从查询中提取关键词
func extractKeywords(query string) []string {
	// 简单的关键词提取：去除常见词
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "and": true, "or": true, "with": true, "by": true,
		"about": true, "research": true, "paper": true, "papers": true,
		"related": true, "want": true, "find": true, "search": true,
		"looking": true, "interested": true, "i": true, "me": true,
		"my": true, "recent": true, "new": true, "latest": true,
	}

	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:'\"")
		if len(w) > 2 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}

	return keywords
}

// generateFallbackTitle 生成回退标题
func generateFallbackTitle(query string) string {
	keywords := extractKeywords(query)
	if len(keywords) == 0 {
		return "Recent Advances in Machine Learning and Artificial Intelligence"
	}

	// 取前3个关键词
	if len(keywords) > 3 {
		keywords = keywords[:3]
	}

	// 首字母大写
	for i, kw := range keywords {
		if len(kw) > 0 {
			keywords[i] = strings.ToUpper(kw[:1]) + kw[1:]
		}
	}

	return fmt.Sprintf("Advances in %s: Methods, Applications and Future Directions", strings.Join(keywords, ", "))
}

// generateFallbackAbstract 生成回退摘要
func generateFallbackAbstract(query string, keywords []string) string {
	if len(keywords) == 0 {
		keywords = []string{"machine learning", "deep learning"}
	}

	keywordStr := strings.Join(keywords, ", ")

	return fmt.Sprintf(`This paper presents a comprehensive study on %s, addressing key challenges and proposing novel solutions in the field. We first analyze the current state of research and identify critical gaps in existing approaches. Our work introduces innovative methodologies that significantly improve upon baseline methods, demonstrating strong performance across multiple benchmarks. Through extensive experiments, we validate our approach and show substantial improvements in both efficiency and effectiveness. The proposed techniques offer practical benefits for real-world applications while maintaining theoretical rigor. We also discuss limitations and outline promising directions for future research in this rapidly evolving area.`, keywordStr)
}
