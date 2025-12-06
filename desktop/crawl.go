package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"PaperHunter/internal/models"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CrawlTask 爬取任务
type CrawlTask struct {
	ID         string                 `json:"id"`
	Platform   string                 `json:"platform"`
	Params     map[string]interface{} `json:"params"`
	Status     string                 `json:"status"` // pending, running, completed, failed
	Progress   int                    `json:"progress"`
	TotalCount int                    `json:"total_count"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Logs       []LogEntry             `json:"logs"`
	Inserted   []PaperRef             `json:"inserted,omitempty"`
	mu         sync.RWMutex
}

// PaperRef 记录本次任务成功入库的论文引用，便于前端一键导出
type PaperRef struct {
	Source   string `json:"source"`
	SourceID string `json:"source_id"`
	URL      string `json:"url"`
	PaperID  int64  `json:"paper_id"`
}

// LogEntry 日志条目
type LogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, success, warning, error, debug
	Message   string    `json:"message"`
	Platform  string    `json:"platform,omitempty"`
	Count     int       `json:"count,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
}

// CrawlHistory 任务历史记录（持久化）
type CrawlHistory struct {
	TaskID    string                 `json:"task_id"`
	Platform  string                 `json:"platform"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Total     int                    `json:"total"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
}

// CrawlService 爬取服务
type CrawlService struct {
	tasks map[string]*CrawlTask
	mu    sync.RWMutex
	app   *App
}

// NewCrawlService 创建爬取服务
func NewCrawlService(app *App) *CrawlService {
	return &CrawlService{
		tasks: make(map[string]*CrawlTask),
		app:   app,
	}
}

// StartCrawl 开始爬取任务
func (cs *CrawlService) StartCrawl(platform string, params map[string]interface{}) (string, error) {
	taskID := fmt.Sprintf("crawl_%d", time.Now().UnixNano())

	task := &CrawlTask{
		ID:        taskID,
		Platform:  platform,
		Params:    params,
		Status:    "pending",
		Progress:  0,
		StartTime: time.Now(),
		Logs:      make([]LogEntry, 0),
	}

	cs.mu.Lock()
	cs.tasks[taskID] = task
	cs.mu.Unlock()

	// 异步执行爬取任务
	go cs.executeCrawlTask(task)

	return taskID, nil
}

// GetTask 获取任务状态
func (cs *CrawlService) GetTask(taskID string) (*CrawlTask, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	task, exists := cs.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetTaskLogs 获取任务日志
func (cs *CrawlService) GetTaskLogs(taskID string) ([]LogEntry, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	task, exists := cs.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	return task.Logs, nil
}

// executeCrawlTask 执行爬取任务
func (cs *CrawlService) executeCrawlTask(task *CrawlTask) {
	task.mu.Lock()
	task.Status = "running"
	task.mu.Unlock()

	cs.addLog(task, "info", fmt.Sprintf("开始从 %s 爬取论文...", task.Platform), task.Platform)

	// 构建查询参数
	query := cs.buildQuery(task.Platform, task.Params)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cs.addLog(task, "debug", "抓取进行中...", task.Platform)
			case <-done:
				return
			}
		}
	}()

	// 执行爬取
	ctx := context.Background()
	// 带进度回调，逐条记录 URL
	count, err := cs.app.coreApp.CrawlWithProgress(ctx, task.Platform, query, func(idx int, total int, p *models.Paper, paperID int64) {
		if p == nil {
			return
		}
		// 记录入库成功的论文引用，便于一键导出
		task.mu.Lock()
		task.Inserted = append(task.Inserted, PaperRef{
			Source:   p.Source,
			SourceID: p.SourceID,
			URL:      p.URL,
			PaperID:  paperID,
		})
		task.mu.Unlock()

		cs.addLog(task, "debug", fmt.Sprintf("[%d/%d] %s", idx+1, total, p.URL), task.Platform)
	})

	task.mu.Lock()
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		now := time.Now()
		task.EndTime = &now
	} else {
		task.Status = "completed"
		task.TotalCount = count
		now := time.Now()
		task.EndTime = &now
	}
	task.mu.Unlock()

	// 停止心跳
	close(done)

	if err != nil {
		cs.addLog(task, "error", fmt.Sprintf("爬取失败: %v", err), task.Platform)
	} else {
		cs.addLog(task, "success", fmt.Sprintf("爬取完成！共获取 %d 篇论文", count), task.Platform, count)
		cs.saveTaskHistory(task)
	}
}

// buildQuery 构建查询参数
func (cs *CrawlService) buildQuery(platformName string, params map[string]interface{}) platform.Query {
	query := platform.Query{}

	// 通用参数
	if keywords, ok := params["keywords"].([]interface{}); ok {
		for _, k := range keywords {
			if keyword, ok := k.(string); ok {
				query.Keywords = append(query.Keywords, keyword)
			}
		}
	}

	if categories, ok := params["categories"].([]interface{}); ok {
		for _, c := range categories {
			if category, ok := c.(string); ok {
				query.Categories = append(query.Categories, category)
			}
		}
	}

	if dateFrom, ok := params["dateFrom"].(string); ok {
		query.DateFrom = dateFrom
	}

	if dateTo, ok := params["dateTo"].(string); ok {
		query.DateTo = dateTo
	}

	if limit, ok := params["limit"].(float64); ok {
		query.Limit = int(limit)
	}

	// 平台特定参数
	if platformName == "openreview" {
		if venueId, ok := params["venueId"].(string); ok {
			// OpenReview 使用 venueId 作为 categories
			query.Categories = []string{venueId}
		}
	}

	return query
}

// historyPath 获取历史文件路径（与数据库同目录）
func (cs *CrawlService) historyPath() string {
	if cs.app != nil && cs.app.config != nil && cs.app.config.Database.Path != "" {
		return filepath.Join(filepath.Dir(cs.app.config.Database.Path), "crawl_history.jsonl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".quicksearch", "data", "crawl_history.jsonl")
}

// saveTaskHistory 将完成的任务写入本地 jsonl
func (cs *CrawlService) saveTaskHistory(task *CrawlTask) {
	if task == nil || task.Status != "completed" || task.EndTime == nil {
		return
	}
	entry := CrawlHistory{
		TaskID:    task.ID,
		Platform:  task.Platform,
		Params:    task.Params,
		Total:     task.TotalCount,
		StartTime: task.StartTime,
		EndTime:   *task.EndTime,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		logger.Warn("写入历史失败(序列化): %v", err)
		return
	}
	path := cs.historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		logger.Warn("创建历史目录失败: %v", err)
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Warn("打开历史文件失败: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		logger.Warn("写入历史文件失败: %v", err)
	}

	cs.truncateHistoryFile(10)
}

// loadHistory 读取历史记录，limit=0 表示全部
func (cs *CrawlService) loadHistory(limit int) ([]CrawlHistory, error) {
	path := cs.historyPath()
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []CrawlHistory{}, nil
		}
		return nil, err
	}
	lines := bytes.Split(bytes.TrimSpace(content), []byte{'\n'})
	history := make([]CrawlHistory, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- { // 逆序：最新在前
		if len(lines[i]) == 0 {
			continue
		}
		var h CrawlHistory
		if err := json.Unmarshal(lines[i], &h); err != nil {
			continue
		}
		history = append(history, h)
		if limit > 0 && len(history) >= limit {
			break
		}
	}
	return history, nil
}

// truncateHistoryFile 仅保留最近 max 条
func (cs *CrawlService) truncateHistoryFile(max int) {
	if max <= 0 {
		return
	}
	path := cs.historyPath()
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := bytes.Split(bytes.TrimSpace(content), []byte{'\n'})
	if len(lines) <= max {
		return
	}
	// 保留最新的 max 条（按写入顺序，文件末尾是最新）
	start := len(lines) - max
	trimmed := bytes.Join(lines[start:], []byte{'\n'})
	if err := os.WriteFile(path, append(trimmed, '\n'), 0644); err != nil {
		logger.Warn("截断历史文件失败: %v", err)
	}
}

// clearHistory 删除历史文件
func (cs *CrawlService) clearHistory() error {
	path := cs.historyPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// addLog 添加日志
func (cs *CrawlService) addLog(task *CrawlTask, level, message, platform string, count ...int) {
	logEntry := LogEntry{
		ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Platform:  platform,
		TaskID:    task.ID,
	}

	if len(count) > 0 {
		logEntry.Count = count[0]
	}

	task.mu.Lock()
	task.Logs = append(task.Logs, logEntry)
	task.mu.Unlock()

	// 记录到系统日志
	switch level {
	case "error":
		logger.Error("[%s] %s", platform, message)
	case "warning":
		logger.Warn("[%s] %s", platform, message)
	case "success":
		logger.Info("[%s] %s", platform, message)
	default:
		logger.Debug("[%s] %s", platform, message)
	}

	// 发送事件到前端
	if cs.app.ctx != nil {
		runtime.EventsEmit(cs.app.ctx, "crawl-log", logEntry)
	}
}

// GetAllTasks 获取所有任务
func (cs *CrawlService) GetAllTasks() []*CrawlTask {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	tasks := make([]*CrawlTask, 0, len(cs.tasks))
	for _, task := range cs.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// CleanupCompletedTasks 清理已完成的任务
func (cs *CrawlService) CleanupCompletedTasks() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour) // 保留24小时内的任务

	for id, task := range cs.tasks {
		if (task.Status == "completed" || task.Status == "failed") &&
			task.EndTime != nil && task.EndTime.Before(cutoff) {
			delete(cs.tasks, id)
		}
	}
}
