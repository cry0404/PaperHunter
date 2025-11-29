package acl

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"PaperHunter/internal/models"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"
)

type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// BibTeX 相关结构
type BibTeXEntry struct {
	ID           string
	Title        string
	Authors      []string
	Abstract     string
	Venue        string
	Year         string
	Pages        string
	Publisher    string
	URL          string
	DOI          string
	Keywords     []string
	Month        string
	Volume       string
	Number       string
	Address      string
	Booktitle    string
	Journal      string
	Series       string
	Editor       []string
	Note         string
	Annote       string
	Crossref     string
	Howpublished string
	Institution  string
	Organization string
	School       string
	Type         string
}

// RSS 解析方法
func (a *Adapter) searchViaRSS(ctx context.Context, q platform.Query) (platform.Result, error) {
	rssURL := a.config.BaseURL + "/papers/index.xml"
	logger.Debug("[ACL] 请求 RSS 源: %s", rssURL)

	content, err := a.request(ctx, rssURL)
	if err != nil {
		return platform.Result{}, fmt.Errorf("RSS request failed: %w", err)
	}

	// 解析 RSS
	var feed RSSFeed
	if err := xml.Unmarshal([]byte(content), &feed); err != nil {
		return platform.Result{}, fmt.Errorf("failed to parse RSS: %w", err)
	}

	logger.Info("[ACL] RSS 解析完成，共 %d 篇论文", len(feed.Channel.Items))

	var papers []*models.Paper
	for _, item := range feed.Channel.Items {
		paper := a.convertRSSItemToPaper(item)
		if paper != nil && a.matchesQuery(paper, q) {
			papers = append(papers, paper)
		}
	}

	if q.Limit > 0 && len(papers) > q.Limit {
		papers = papers[:q.Limit]
	}

	logger.Info("[ACL] RSS 模式完成，返回 %d 篇论文", len(papers))
	return platform.Result{
		Total:  len(papers),
		Papers: papers,
	}, nil
}

func (a *Adapter) convertRSSItemToPaper(item RSSItem) *models.Paper {
	paper := &models.Paper{
		Source:    "acl",
		URL:       item.Link,
		Title:     cleanText(item.Title),
		Abstract:  cleanText(item.Description),
		UpdatedAt: time.Now(),
	}

	// 生成 SourceID - 使用标题哈希确保唯一性
	if paper.Title != "" {
		// 使用标题生成哈希值
		titleHash := a.generateTitleHash(paper.Title)
		paper.SourceID = fmt.Sprintf("acl_rss_%s", titleHash)
	} else {
		// 如果标题为空，使用时间戳
		paper.SourceID = fmt.Sprintf("acl_rss_%d", time.Now().UnixNano())
	}

	// 如果 URL 为空，生成唯一的 URL
	if paper.URL == "" {
		paper.URL = fmt.Sprintf("https://aclanthology.org/%s", paper.SourceID)
	}

	// 解析作者（从描述中提取）
	if strings.Contains(item.Description, " in ") {
		parts := strings.Split(item.Description, " in ")
		if len(parts) > 0 {
			authorsStr := strings.TrimSpace(parts[0])
			paper.Authors = parseAuthors(authorsStr)
		}
	}

	// 解析会议信息
	if strings.Contains(item.Description, " in ") {
		parts := strings.Split(item.Description, " in ")
		if len(parts) > 1 {
			venueStr := strings.TrimSpace(parts[1])
			paper.Categories = []string{venueStr}
		}
	}

	// 解析日期
	if item.PubDate != "" {
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			paper.FirstSubmittedAt = t
			paper.FirstAnnouncedAt = t
		}
	}

	return paper
}

// BibTeX 解析方法
func (a *Adapter) searchViaBibTeX(ctx context.Context, q platform.Query) (platform.Result, error) {
	bibURL := a.config.BaseURL + "/anthology+abstracts.bib.gz"
	logger.Debug("[ACL] 请求 BibTeX 文件: %s", bibURL)

	// 直接使用 HTTP 客户端下载 gzip 文件
	req, err := http.NewRequestWithContext(ctx, "GET", bibURL, nil)
	if err != nil {
		return platform.Result{}, fmt.Errorf("failed to create request: %w", err)
	}
	//不重要，这类论文平台有 user-agent 头就可以了
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/gzip,application/octet-stream")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return platform.Result{}, fmt.Errorf("BibTeX request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return platform.Result{}, fmt.Errorf("HTTP error fetching BibTeX: %d", resp.StatusCode)
	}

	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return platform.Result{}, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		return platform.Result{}, fmt.Errorf("failed to read BibTeX response: %w", err)
	}

	papers, err := a.parseBibTeX(string(body))
	if err != nil {
		return platform.Result{}, fmt.Errorf("failed to parse BibTeX: %w", err)
	}

	logger.Info("[ACL] BibTeX 解析完成，共 %d 篇论文", len(papers))

	var filteredPapers []*models.Paper
	for _, paper := range papers {
		if a.matchesQuery(paper, q) {
			filteredPapers = append(filteredPapers, paper)
		}
	}

	if q.Limit > 0 && len(filteredPapers) > q.Limit {
		filteredPapers = filteredPapers[:q.Limit]
	}

	logger.Info("[ACL] BibTeX 模式完成，返回 %d 篇论文", len(filteredPapers))
	return platform.Result{
		Total:  len(filteredPapers),
		Papers: filteredPapers,
	}, nil
}

func (a *Adapter) parseBibTeX(content string) ([]*models.Paper, error) {
	var papers []*models.Paper

	// 按 @ 分割条目
	entries := strings.Split(content, "@")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// 优先处理包含摘要的条目类型
		if strings.HasPrefix(entry, "inproceedings") ||
			strings.HasPrefix(entry, "article") ||
			strings.HasPrefix(entry, "incollection") ||
			strings.HasPrefix(entry, "inbook") {
			paper := a.parseBibTeXEntry(entry)
			if paper != nil {
				papers = append(papers, paper)
			}
		}
	}

	return papers, nil
}

func (a *Adapter) parseBibTeXEntry(entry string) *models.Paper {
	paper := &models.Paper{
		Source:    "acl",
		UpdatedAt: time.Now(),
	}

	paper.Title = a.extractBibTeXField(entry, "title")
	paper.Abstract = a.extractBibTeXField(entry, "abstract")
	paper.URL = a.extractBibTeXField(entry, "url")

	// 解析作者
	authorsStr := a.extractBibTeXField(entry, "author")
	if authorsStr != "" {
		paper.Authors = parseAuthors(authorsStr)
	}

	// 解析年份
	yearStr := a.extractBibTeXField(entry, "year")
	if yearStr != "" {
		if year, err := time.Parse("2006", yearStr); err == nil {
			paper.FirstSubmittedAt = year
			paper.FirstAnnouncedAt = year
		}
	}

	// 解析 DOI
	doi := a.extractBibTeXField(entry, "doi")
	if doi != "" {
		paper.Comments = "DOI: " + doi
	}

	// 解析会议/期刊
	venue := a.extractBibTeXField(entry, "booktitle")
	if venue == "" {
		venue = a.extractBibTeXField(entry, "journal")
	}
	if venue != "" {
		paper.Categories = []string{venue}
	}

	// 生成 SourceID - 使用标题哈希确保唯一性
	if paper.Title != "" {
		// 使用标题生成哈希值
		titleHash := a.generateTitleHash(paper.Title)
		paper.SourceID = fmt.Sprintf("acl_%s", titleHash)
	} else {
		// 如果标题为空，使用时间戳
		paper.SourceID = fmt.Sprintf("acl_%d", time.Now().UnixNano())
	}

	// 如果 URL 为空，生成唯一的 URL
	if paper.URL == "" {
		paper.URL = fmt.Sprintf("https://aclanthology.org/%s", paper.SourceID)
	}

	// 调试日志 - 只在 SourceID 异常时输出
	if paper.SourceID == "" || len(paper.SourceID) < 3 {
		logger.Debug("[ACL] 异常 SourceID: title=%s, url=%s, source_id=%s", paper.Title, paper.URL, paper.SourceID)
	}

	return paper
}

func (a *Adapter) extractBibTeXField(entry, fieldName string) string {
	// 查找字段开始位置 - 支持双引号和大括号两种格式
	fieldPattern := fmt.Sprintf(`(?i)%s\s*=\s*["\{]`, fieldName)
	re := regexp.MustCompile(fieldPattern)
	lines := strings.Split(entry, "\n")

	for i, line := range lines {
		if re.MatchString(line) {
			// 找到字段，提取值（可能跨多行）
			value := a.extractMultiLineBibTeXValue(lines, i)
			cleaned := cleanText(value)
			return cleaned
		}
	}
	// logger.Debug("[ACL] 未找到字段 %s", fieldName)
	return ""
}

func (a *Adapter) extractMultiLineBibTeXValue(lines []string, startLine int) string {
	// 从指定行开始提取多行 BibTeX 值
	var value strings.Builder
	braceCount := 0
	quoteCount := 0
	inValue := false
	inQuotes := false

	for i := startLine; i < len(lines); i++ {
		line := lines[i]

		// 查找第一个 { 或 " 开始位置
		if !inValue {
			braceIdx := strings.Index(line, "{")
			quoteIdx := strings.Index(line, "\"")

			if braceIdx == -1 && quoteIdx == -1 {
				continue
			}

			if quoteIdx != -1 && (braceIdx == -1 || quoteIdx < braceIdx) {

				line = line[quoteIdx+1:]
				inValue = true
				inQuotes = true
				quoteCount = 1
			} else {

				line = line[braceIdx+1:]
				inValue = true
				braceCount = 1
			}
		}

		if inQuotes {
			// 处理双引号格式
			quoteCount += strings.Count(line, "\"")
			if quoteCount%2 == 0 {
				// 找到结束引号
				value.WriteString(strings.TrimSuffix(line, "\""))
				break
			} else {
				value.WriteString(line)
			}
		} else {
			// 处理大括号格式
			braceCount += strings.Count(line, "{")
			braceCount -= strings.Count(line, "}")

			if braceCount <= 0 {
				// 值结束
				value.WriteString(strings.TrimSuffix(line, "}"))
				break
			} else {
				value.WriteString(line)
			}
		}
	}

	return value.String()
}

// generateTitleHash 生成标题的哈希值
func (a *Adapter) generateTitleHash(title string) string {
	// 清理标题，移除特殊字符
	cleanTitle := strings.ToLower(title)
	cleanTitle = strings.ReplaceAll(cleanTitle, " ", "_")
	cleanTitle = strings.ReplaceAll(cleanTitle, ",", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, ".", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, ":", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, ";", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "?", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "!", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "(", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, ")", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "[", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "]", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "{", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "}", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "\"", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "'", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "/", "_")
	cleanTitle = strings.ReplaceAll(cleanTitle, "\\", "_")

	// 限制长度并添加时间戳确保唯一性
	if len(cleanTitle) > 30 {
		cleanTitle = cleanTitle[:30]
	}

	// 添加时间戳确保唯一性
	timestamp := time.Now().UnixNano() % 1000000 // 取后6位
	return fmt.Sprintf("%s_%d", cleanTitle, timestamp)
}

// 查询匹配
func (a *Adapter) matchesQuery(paper *models.Paper, q platform.Query) bool {
	// 关键词匹配
	if len(q.Keywords) > 0 {
		text := strings.ToLower(paper.Title + " " + paper.Abstract)
		for _, kw := range q.Keywords {
			if !strings.Contains(text, strings.ToLower(kw)) {
				return false
			}
		}
	}

	// 日期范围匹配
	if q.DateFrom != "" {
		if fromDate, err := time.Parse("2006-01-02", q.DateFrom); err == nil {
			if paper.FirstSubmittedAt.Before(fromDate) {
				return false
			}
		}
	}

	if q.DateTo != "" {
		if toDate, err := time.Parse("2006-01-02", q.DateTo); err == nil {
			if paper.FirstSubmittedAt.After(toDate) {
				return false
			}
		}
	}

	return true
}

// 辅助函数
func cleanText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	// 处理 LaTeX 格式的特殊字符
	// 移除 LaTeX 强调标记 {text} -> text
	latexEmphasis := regexp.MustCompile(`\{([^}]+)\}`)
	text = latexEmphasis.ReplaceAllString(text, "$1")

	// 移除其他 LaTeX 命令
	latexCommands := regexp.MustCompile(`\\[a-zA-Z]+\{[^}]*\}`)
	text = latexCommands.ReplaceAllString(text, "")

	// 移除单独的 LaTeX 命令（无参数）
	latexSimpleCommands := regexp.MustCompile(`\\[a-zA-Z]+`)
	text = latexSimpleCommands.ReplaceAllString(text, "")

	// 处理特殊字符
	text = strings.ReplaceAll(text, "\\&", "&")
	text = strings.ReplaceAll(text, "\\%", "%")
	text = strings.ReplaceAll(text, "\\$", "$")
	text = strings.ReplaceAll(text, "\\#", "#")
	text = strings.ReplaceAll(text, "\\_", "_")
	text = strings.ReplaceAll(text, "\\{", "{")
	text = strings.ReplaceAll(text, "\\}", "}")

	// 压缩多个空格
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return text
}

func parseAuthors(authorsStr string) []string {
	if authorsStr == "" {
		return nil
	}

	// 处理 " and " 分隔符
	authors := strings.Split(authorsStr, " and ")
	var result []string
	for _, author := range authors {
		author = strings.TrimSpace(author)
		if author != "" {
			result = append(result, author)
		}
	}
	return result
}
