package main

import (
	"context"
	"fmt"
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
	mu         sync.RWMutex
}

// LogEntry 日志条目
type LogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, success, warning, error, debug
	Message   string    `json:"message"`
	Platform  string    `json:"platform,omitempty"`
	Count     int       `json:"count,omitempty"`
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
	count, err := cs.app.coreApp.CrawlWithProgress(ctx, task.Platform, query, func(idx int, total int, p *models.Paper) {
		if p == nil {
			return
		}
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

// addLog 添加日志
func (cs *CrawlService) addLog(task *CrawlTask, level, message, platform string, count ...int) {
	logEntry := LogEntry{
		ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Platform:  platform,
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
