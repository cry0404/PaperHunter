package arxiv

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"

	"PaperHunter/internal/models"

	"github.com/PuerkitoBio/goquery"
)

func ParseSearchHTML(htmlContent string) ([]*models.Paper, int, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse HTML: %w", err)
	}

	total := 0
	candidates := []string{
		"#main-container h1",
		"h1.title",
		"h1",
	}
	for _, sel := range candidates {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			if total > 0 {
				return
			}
			text := cleanText(s.Text())
			if strings.Contains(text, "Sorry") {
				total = 0
				return
			}

			re := regexp.MustCompile(`of\s+([\d,]+)\s+results`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				totalStr := strings.ReplaceAll(matches[1], ",", "")
				fmt.Sscanf(totalStr, "%d", &total)
			}
		})
		if total > 0 {
			break
		}
	}

	var papers []*models.Paper
	doc.Find("li.arxiv-result").Each(func(i int, s *goquery.Selection) {
		paper := parsePaperItem(s)
		if paper != nil {
			papers = append(papers, paper)
		}
	})

	if total == 0 && len(papers) > 0 {
		total = len(papers)
	}

	return papers, total, nil
}

func parsePaperItem(s *goquery.Selection) *models.Paper {
	paper := &models.Paper{
		Source: "arxiv", // 设置平台标识
	}

	if link := s.Find("a").First(); link.Length() > 0 {
		paper.URL, _ = link.Attr("href")
		paper.SourceID = parseArxivIDFromURL(paper.URL)
	}

	if title := s.Find("p.title"); title.Length() > 0 {
		paper.Title = cleanText(title.Text())
	}

	if authors := s.Find("p.authors"); authors.Length() > 0 {
		text := authors.Text()
		text = strings.TrimPrefix(text, "Authors:")
		authorsStr := cleanText(text)
		paper.Authors = parseAuthorsToSlice(authorsStr)
	}

	if abstract := s.Find("span.abstract-full"); abstract.Length() > 0 {
		paper.Abstract = cleanText(parseAbstract(abstract))
	}

	var categories []string
	s.Find("span.tag.tooltip").Each(func(i int, tag *goquery.Selection) {
		cat := strings.TrimSpace(tag.Text())
		if cat != "" {
			categories = append(categories, cat)
		}
	})
	paper.Categories = categories

	if comments := s.Find("p.comments"); comments.Length() > 0 {
		text := comments.Text()
		text = strings.TrimPrefix(text, "Comments:")
		paper.Comments = cleanText(text)
	}

	if dateElem := s.Find("p.is-size-7"); dateElem.Length() > 0 {
		paper.FirstSubmittedAt = parseDate(dateElem.Text())
		paper.FirstAnnouncedAt = paper.FirstSubmittedAt
	}

	paper.UpdatedAt = time.Now()
	return paper
}

func parseAbstract(s *goquery.Selection) string {
	var text string
	s.Contents().Each(func(i int, node *goquery.Selection) {
		if goquery.NodeName(node) == "#text" {
			text += node.Text()
		} else if node.Is("span.search-hit") {
			text += node.Text()
		}
	})
	return text
}

func parseDate(text string) time.Time {
	var dateStr string

	if strings.Contains(text, "v1submitted") || strings.Contains(text, "v1 submitted") {
		re := regexp.MustCompile(`v1\s*submitted\s+(.+?);\s*originally`)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			dateStr = matches[1]
		}
	}

	if dateStr == "" {
		re := regexp.MustCompile(`Submitted\s*(.+?);\s*originally`)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			dateStr = matches[1]
		}
	}

	dateStr = strings.TrimSpace(dateStr)
	t, err := time.Parse("2 January, 2006", dateStr)
	if err != nil {
		t, _ = time.Parse("2 Jan, 2006", dateStr)
	}
	return t
}

func cleanText(text string) string {
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func parseArxivIDFromURL(url string) string {
	// 从 https://arxiv.org/abs/2408.12345 提取 2408.12345
	if len(url) > 0 {
		idx := strings.LastIndex(url, "/")
		if idx > 0 && idx < len(url)-1 {
			return url[idx+1:]
		}
	}
	return ""
}

func parseAuthorsToSlice(authorsStr string) []string {
	if authorsStr == "" {
		return nil
	}
	authors := strings.Split(authorsStr, ",")
	result := make([]string, 0, len(authors))
	for _, author := range authors {
		author = strings.TrimSpace(author)
		if author != "" {
			result = append(result, author)
		}
	}
	return result
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Total   int         `xml:"http://a9.com/-/spec/opensearch/1.1/ totalResults"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Summary    string         `xml:"summary"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Authors    []AtomAuthor   `xml:"author"`
	Links      []AtomLink     `xml:"link"`
	Categories []AtomCategory `xml:"category"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

type AtomCategory struct {
	Term string `xml:"term,attr"`
}

// ParseNewSubmissionsHTML 解析 arXiv New Submissions 页面
// URL 格式: https://arxiv.org/list/cs/new
func ParseNewSubmissionsHTML(htmlContent string) ([]*models.Paper, int, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var papers []*models.Paper

	// New submissions 页面的结构：
	// <dl id="articles"> 或 <dl>
	//   <dt>...</dt>  <- 包含 arXiv ID 和链接
	//   <dd>...</dd>  <- 包含标题、作者、摘要
	// </dl>

	// 查找所有 dt 元素（每个 dt 对应一篇论文的元信息）
	doc.Find("dl dt").Each(func(i int, dt *goquery.Selection) {
		dd := dt.Next() // 对应的 dd 元素
		if dd.Length() == 0 || goquery.NodeName(dd) != "dd" {
			return
		}

		paper := &models.Paper{
			Source: "arxiv",
		}

		// 从 dt 中提取 arXiv ID 和链接
		// 格式: <a href="/abs/2411.xxxxx" title="Abstract">arXiv:2411.xxxxx</a>
		dt.Find("a[href*='/abs/']").Each(func(_ int, link *goquery.Selection) {
			href, exists := link.Attr("href")
			if exists && paper.URL == "" {
				if strings.HasPrefix(href, "/") {
					paper.URL = "https://arxiv.org" + href
				} else {
					paper.URL = href
				}
				paper.SourceID = parseArxivIDFromURL(href)
			}
		})

		// 如果没有找到 abs 链接，尝试其他方式
		if paper.SourceID == "" {
			// 尝试从文本中提取 arXiv ID
			dtText := dt.Text()
			re := regexp.MustCompile(`arXiv:(\d{4}\.\d{4,5})`)
			matches := re.FindStringSubmatch(dtText)
			if len(matches) > 1 {
				paper.SourceID = matches[1]
				paper.URL = "https://arxiv.org/abs/" + matches[1]
			}
		}

		// 从 dd 中提取标题
		// 格式: <div class="list-title mathjax">Title: xxxx</div>
		if title := dd.Find("div.list-title"); title.Length() > 0 {
			titleText := title.Text()
			titleText = strings.TrimPrefix(titleText, "Title:")
			titleText = strings.TrimPrefix(titleText, "Title :")
			paper.Title = cleanText(titleText)
		}

		// 从 dd 中提取作者
		// 格式: <div class="list-authors">Authors: xxx, yyy</div>
		if authors := dd.Find("div.list-authors"); authors.Length() > 0 {
			authorsText := authors.Text()
			authorsText = strings.TrimPrefix(authorsText, "Authors:")
			authorsText = strings.TrimPrefix(authorsText, "Authors :")
			paper.Authors = parseAuthorsToSlice(cleanText(authorsText))
		}

		// 从 dd 中提取摘要
		// 格式: <p class="mathjax">摘要内容</p>
		if abstract := dd.Find("p.mathjax"); abstract.Length() > 0 {
			paper.Abstract = cleanText(abstract.Text())
		}

		// 从 dd 中提取分类
		// 格式: <span class="primary-subject">cs.AI</span>
		var categories []string
		dd.Find("span.primary-subject").Each(func(_ int, span *goquery.Selection) {
			cat := strings.TrimSpace(span.Text())
			if cat != "" {
				categories = append(categories, cat)
			}
		})
		// 也尝试从 list-subjects 中提取
		if subjects := dd.Find("div.list-subjects"); subjects.Length() > 0 {
			subjectsText := subjects.Text()
			subjectsText = strings.TrimPrefix(subjectsText, "Subjects:")
			subjectsText = strings.TrimPrefix(subjectsText, "Subjects :")
			// 分类通常用分号或逗号分隔
			for _, cat := range strings.Split(subjectsText, ";") {
				cat = strings.TrimSpace(cat)
				if cat != "" && !containsString(categories, cat) {
					categories = append(categories, cat)
				}
			}
		}
		paper.Categories = categories

		// 设置今天的日期作为发布日期（New Submissions 页面的论文都是今天公布的）
		paper.FirstSubmittedAt = time.Now()
		paper.FirstAnnouncedAt = time.Now()
		paper.UpdatedAt = time.Now()

		// 只添加有效的论文
		if paper.Title != "" && paper.SourceID != "" {
			papers = append(papers, paper)
		}
	})

	return papers, len(papers), nil
}

// containsString 检查字符串切片是否包含指定字符串
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func ParseAtomFeed(xmlContent string) ([]*models.Paper, int, error) {
	var feed AtomFeed
	if err := xml.Unmarshal([]byte(xmlContent), &feed); err != nil {
		return nil, 0, fmt.Errorf("failed to parse atom: %w", err)
	}

	var papers []*models.Paper
	for _, e := range feed.Entries {
		p := &models.Paper{
			Source: "arxiv", // 设置平台标识
		}

		// e.ID 类似 http://arxiv.org/abs/XXXX
		p.URL = e.ID
		p.SourceID = parseArxivIDFromURL(e.ID)
		p.Title = cleanText(e.Title)
		p.Abstract = cleanText(e.Summary)

		var authorNames []string
		for _, a := range e.Authors {
			name := strings.TrimSpace(a.Name)
			if name != "" {
				authorNames = append(authorNames, name)
			}
		}
		p.Authors = authorNames

		// Categories - 转换为字符串切片
		var cats []string
		for _, c := range e.Categories {
			if c.Term != "" {
				cats = append(cats, c.Term)
			}
		}
		p.Categories = cats

		if t, err := time.Parse(time.RFC3339, e.Published); err == nil {
			p.FirstSubmittedAt = t
			p.FirstAnnouncedAt = t
		}
		p.UpdatedAt = time.Now()

		papers = append(papers, p)
	}

	return papers, feed.Total, nil
}
