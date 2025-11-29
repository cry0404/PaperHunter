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
