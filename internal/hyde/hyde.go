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


type Service interface {

	GenerateHypotheticalPaper(ctx context.Context, userQuery string) (*HypotheticalPaper, error)
}


type hydeService struct {
	model *openai.ChatModel
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

	return &hydeService{model: model}, nil
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

	resp, err := s.model.Generate(ctx, messages)
	if err != nil {
		logger.Error("LLM 生成失败: %v", err)
		return nil, fmt.Errorf("LLM 生成失败: %w", err)
	}

	if resp == nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 返回空响应")
	}


	paper, err := parseHyDEResponse(resp.Content)
	if err != nil {
		logger.Warn("解析 LLM 响应失败，尝试提取文本: %v", err)

		return nil, nil
	}

	return paper, nil
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




