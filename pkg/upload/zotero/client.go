package zotero

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"
)

type Client struct {
	userID     string
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewClient(userID, apiKey string) *Client {
	return &Client{
		userID:  userID,
		apiKey:  apiKey,
		baseURL: "https://api.zotero.org",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddPaper 添加论文到 Zotero（使用统一的 models.Paper）
func (c *Client) AddPaper(paper *models.Paper, collectionKey string) error {
	item := c.paperToZoteroItem(paper, collectionKey)

	items := []ItemData{item}
	jsonData, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("解析失败: %w", err)
	}

	url := fmt.Sprintf("%s/users/%s/items", c.baseURL, c.userID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Zotero-API-Version", "3")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned error %d: %s", resp.StatusCode, string(body))
	}

	var result CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Failed) > 0 {
		for _, failed := range result.Failed {
			return fmt.Errorf("failed to add paper: %s", failed.Message)
		}
	}

	return nil
}

// AddPapers 批量添加论文
func (c *Client) AddPapers(papers []*models.Paper, collectionKey string) error {
	batchSize := 50
	for i := 0; i < len(papers); i += batchSize {
		end := i + batchSize
		if end > len(papers) {
			end = len(papers)
		}
		batch := papers[i:end]
		if err := c.addPapersBatch(batch, collectionKey); err != nil {
			return fmt.Errorf("failed to add batch %d-%d: %w", i, end, err)
		}
		fmt.Printf("Added papers %d-%d to Zotero\n", i+1, end)
		// 429 避免触发速率限制， 请自行阅读 zotero 文档
		time.Sleep(1 * time.Second)
	}
	return nil
}

// addPapersBatch 批量添加论文（单次请求）
func (c *Client) addPapersBatch(papers []*models.Paper, collectionKey string) error {
	items := make([]ItemData, len(papers))
	for i, paper := range papers {
		items[i] = c.paperToZoteroItem(paper, collectionKey)
	}

	jsonData, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	url := fmt.Sprintf("%s/users/%s/items", c.baseURL, c.userID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Zotero-API-Version", "3")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned error %d: %s", resp.StatusCode, string(body))
	}

	var result CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Failed) > 0 {
		fmt.Printf("Warning: %d papers failed to add\n", len(result.Failed))
		for key, failed := range result.Failed {
			fmt.Printf("  - Paper %s: %s\n", key, failed.Message)
		}
	}

	return nil
}

// paperToZoteroItem 将统一 Paper 转换为 Zotero ItemData
func (c *Client) paperToZoteroItem(paper *models.Paper, collectionKey string) ItemData {
	creators := c.authorsToCreators(paper.Authors)

	// 构建 Extra 字段（包含平台与平台内ID，及可选译文）
	extra := ""
	if paper.Source != "" && paper.SourceID != "" {
		extra = fmt.Sprintf("%s:%s", strings.ToLower(paper.Source), paper.SourceID)
	}
	if paper.TitleTranslated != "" {
		if extra != "" {
			extra += "\n标题：" + paper.TitleTranslated
		} else {
			extra = "标题：" + paper.TitleTranslated
		}
	}
	if paper.AbstractTranslated != "" {
		abbr := paper.AbstractTranslated
		if len(abbr) > 200 {
			abbr = abbr[:200]
		}
		if extra != "" {
			extra += "\n摘要：" + abbr
		} else {
			extra = "摘要：" + abbr
		}
	}

	repo := paper.Source
	if strings.EqualFold(paper.Source, "arxiv") {
		repo = "arXiv"
	}

	item := ItemData{
		ItemType: "preprint",
		Title:    paper.Title,
		Creators: creators,
		Tags:     c.createTags(paper),
	}

	if paper.Abstract != "" {
		item.AbstractNote = &paper.Abstract
	}
	if paper.URL != "" {
		item.URL = &paper.URL
	}
	if repo != "" {
		item.Repository = &repo
	}
	if paper.SourceID != "" {
		item.ArchiveID = &paper.SourceID
	}
	if extra != "" {
		item.Extra = &extra
	}
	if !paper.FirstSubmittedAt.IsZero() {
		date := paper.FirstSubmittedAt.Format("2006-01-02")
		item.Date = &date
	}
	// 如果指定了集合，且是有效的 key 格式
	if collectionKey != "" {
		if c.isValidCollectionKey(collectionKey) {
			item.Collections = []string{collectionKey}
		} else {
			// 如果不是有效的 key，打印警告但不阻止导出
			fmt.Printf("警告: '%s' 不是有效的 Zotero collection key，将添加到默认位置\n", collectionKey)
		}
	}
	return item
}

func (c *Client) isValidCollectionKey(key string) bool {
	if len(key) < 6 || len(key) > 10 {
		return false
	}
	// 检查是否全为字母数字
	for _, r := range key {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}
	return true
}

// authorsToCreators 将作者名转换为 Zotero Creators
func (c *Client) authorsToCreators(authors []string) []Creator {
	creators := make([]Creator, 0, len(authors))
	for _, name := range authors {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		parts := strings.Fields(name)
		if len(parts) >= 2 {
			creators = append(creators, Creator{
				CreatorType: "author",
				FirstName:   strings.Join(parts[:len(parts)-1], " "),
				LastName:    parts[len(parts)-1],
			})
		} else {
			creators = append(creators, Creator{
				CreatorType: "author",
				Name:        name,
			})
		}
	}
	return creators
}

// createTags 创建标签（包含平台与分类）
func (c *Client) createTags(paper *models.Paper) []Tag {
	tags := []Tag{}
	if paper.Source != "" {
		tags = append(tags, Tag{Tag: strings.ToLower(paper.Source), Type: 1})
	}
	for _, cat := range paper.Categories {
		cat = strings.TrimSpace(cat)
		if cat == "" {
			continue
		}
		tags = append(tags, Tag{Tag: cat, Type: 1})
	}
	return tags
}

// CheckPaperExists 检查论文是否已存在（按平台与平台内ID）
func (c *Client) CheckPaperExists(source string, sourceID string) (bool, error) {
	key := fmt.Sprintf("%s:%s", strings.ToLower(source), sourceID)
	url := fmt.Sprintf("%s/users/%s/items?q=%s", c.baseURL, c.userID, sourceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Zotero-API-Version", "3")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned error: %d", resp.StatusCode)
	}

	var items []Item
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return false, err
	}

	// 检查是否有匹配的条目（Extra 或 ArchiveID 中包含 sourceID）
	for _, item := range items {
		if item.Data.Extra != nil && strings.Contains(strings.ToLower(*item.Data.Extra), strings.ToLower(key)) {
			return true, nil
		}
		if item.Data.ArchiveID != nil && strings.EqualFold(*item.Data.ArchiveID, sourceID) {
			return true, nil
		}
	}
	return false, nil
}

// GetCollections 获取用户的 collection 列表
func (c *Client) GetCollections() ([]Collection, error) {
	url := fmt.Sprintf("%s/users/%s/collections", c.baseURL, c.userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Zotero-API-Version", "3")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned error %d: %s", resp.StatusCode, string(body))
	}

	var collections []Collection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return collections, nil
}

// SelectCollection 交互式选择 collection
func (c *Client) SelectCollection() (string, error) {
	collections, err := c.GetCollections()
	if err != nil {
		return "", fmt.Errorf("获取 collection 列表失败: %w", err)
	}

	if len(collections) == 0 {
		fmt.Println("没有找到任何 collection，将使用默认位置")
		return "", nil
	}

	fmt.Println("\n可用的 Zotero Collections:")
	fmt.Println("序号\tKey\t\t\tName")
	fmt.Println("----\t----\t\t\t----")
	for i, col := range collections {
		fmt.Printf("%d\t%s\t\t%s\n", i+1, col.Key, col.Data.Name)
	}

	fmt.Print("\n请选择要使用的 collection（输入序号，或按回车使用默认位置）: ")
	var input string
	fmt.Scanln(&input)

	if input == "" {
		fmt.Println("使用默认位置")
		return "", nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil {
		return "", fmt.Errorf("无效的选择: %s", input)
	}

	if choice < 1 || choice > len(collections) {
		return "", fmt.Errorf("选择超出范围: %d", choice)
	}

	selected := collections[choice-1]
	fmt.Printf("已选择: %s (%s)\n", selected.Data.Name, selected.Key)
	return selected.Key, nil
}

// GetPapers 从 Zotero 获取论文列表
// collectionKey: 可选，指定 collection key，为空则获取所有论文
// limit: 限制返回数量，0 表示不限制
// 如果指定的 collection 不存在（404），会自动降级为获取所有论文
func (c *Client) GetPapers(collectionKey string, limit int) ([]*models.Paper, error) {
	var url string
	useCollection := collectionKey != ""

	if useCollection {
		url = fmt.Sprintf("%s/users/%s/collections/%s/items", c.baseURL, c.userID, collectionKey)
	} else {
		url = fmt.Sprintf("%s/users/%s/items", c.baseURL, c.userID)
	}

	// 添加查询参数
	if limit > 0 {
		url += fmt.Sprintf("?limit=%d", limit)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Zotero-API-Version", "3")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 如果指定了 collection 但返回 404，降级为获取所有论文
	if resp.StatusCode == http.StatusNotFound && useCollection {
		// 记录警告但不返回错误，而是尝试获取所有论文
		logger.Warn("指定的 Zotero collection '%s' 不存在，将获取所有论文", collectionKey)

		// 重新请求所有论文
		url = fmt.Sprintf("%s/users/%s/items", c.baseURL, c.userID)
		if limit > 0 {
			url += fmt.Sprintf("?limit=%d", limit)
		}

		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		req.Header.Set("Zotero-API-Version", "3")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned error %d: %s", resp.StatusCode, string(body))
	}

	var items []Item
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为 models.Paper
	papers := make([]*models.Paper, 0, len(items))
	for _, item := range items {
		// 只处理论文类型的条目（preprint, journalArticle, conferencePaper 等）
		itemType := item.Data.ItemType
		if itemType != "preprint" && itemType != "journalArticle" && itemType != "conferencePaper" {
			continue
		}

		paper := c.zoteroItemToPaper(&item)
		if paper != nil {
			papers = append(papers, paper)
		}
	}

	return papers, nil
}

// zoteroItemToPaper 将 Zotero Item 转换为 models.Paper
func (c *Client) zoteroItemToPaper(item *Item) *models.Paper {
	if item == nil {
		return nil
	}

	paper := &models.Paper{
		Title:   item.Data.Title,
		URL:     "",
		Authors: make([]string, 0),
	}

	// 设置 URL
	if item.Data.URL != nil {
		paper.URL = *item.Data.URL
	}

	// 设置摘要
	if item.Data.AbstractNote != nil {
		paper.Abstract = *item.Data.AbstractNote
	}

	// 转换作者
	for _, creator := range item.Data.Creators {
		if creator.CreatorType == "author" {
			var authorName string
			if creator.FirstName != "" && creator.LastName != "" {
				authorName = creator.FirstName + " " + creator.LastName
			} else if creator.Name != "" {
				authorName = creator.Name
			} else if creator.LastName != "" {
				authorName = creator.LastName
			}
			if authorName != "" {
				paper.Authors = append(paper.Authors, authorName)
			}
		}
	}

	// 从 Extra 字段提取 source 和 source_id
	if item.Data.Extra != nil {
		extra := *item.Data.Extra
		// 格式通常是 "source:source_id" 或 "arXiv:2401.12345"
		parts := strings.Split(extra, "\n")
		if len(parts) > 0 {
			firstLine := strings.TrimSpace(parts[0])
			if strings.Contains(firstLine, ":") {
				parts2 := strings.SplitN(firstLine, ":", 2)
				if len(parts2) == 2 {
					paper.Source = strings.ToLower(strings.TrimSpace(parts2[0]))
					paper.SourceID = strings.TrimSpace(parts2[1])
				}
			}
		}
	}


	if paper.SourceID == "" && item.Data.ArchiveID != nil {
		paper.SourceID = *item.Data.ArchiveID
		if item.Data.Repository != nil {
			paper.Source = strings.ToLower(*item.Data.Repository)
		}
	}

	if len(item.Data.Tags) > 0 {
		paper.Categories = make([]string, 0, len(item.Data.Tags))
		for _, tag := range item.Data.Tags {
			if tag.Tag != "" {
				paper.Categories = append(paper.Categories, tag.Tag)
			}
		}
	}

	// 设置日期
	if item.Data.Date != nil {
		if t, err := time.Parse("2006-01-02", *item.Data.Date); err == nil {
			paper.FirstSubmittedAt = t
		}
	}

	return paper
}
