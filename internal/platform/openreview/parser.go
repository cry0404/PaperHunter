package openreview

import (
	"encoding/json"
	"fmt"
	"time"

	"PaperHunter/internal/models"
)

type APIResponse struct {
	Notes []struct {
		ID      string `json:"id"`
		Number  int    `json:"number"`
		Content struct {
			Title struct {
				Value string `json:"value"`
			} `json:"title"`
			Authors struct {
				Value []string `json:"value"`
			} `json:"authors"`
			Abstract struct {
				Value string `json:"value"`
			} `json:"abstract"`
			Keywords struct {
				Value []string `json:"value"`
			} `json:"keywords"`
			PrimaryArea struct {
				Value string `json:"value"`
			} `json:"primary_area"`
		} `json:"content"`
	} `json:"notes"`
}

func parseResponse(body string) (*struct{ Notes []*models.Paper }, error) {
	var raw APIResponse
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	papers := make([]*models.Paper, 0, len(raw.Notes))
	for _, note := range raw.Notes {
		paper := &models.Paper{
			Source:           "openreview",
			SourceID:         note.ID,
			URL:              fmt.Sprintf("https://openreview.net/forum?id=%s", note.ID),
			Title:            note.Content.Title.Value,
			Authors:          note.Content.Authors.Value,
			Abstract:         note.Content.Abstract.Value,
			Categories:       append(note.Content.Keywords.Value, note.Content.PrimaryArea.Value),
			FirstSubmittedAt: time.Now(), // OpenReview 未提供，用当前时间
			FirstAnnouncedAt: time.Now(),
			UpdatedAt:        time.Now(),
		}
		papers = append(papers, paper)
	}

	return &struct{ Notes []*models.Paper }{Notes: papers}, nil
}
