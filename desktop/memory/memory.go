package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Event 表示一次用户相关事件（如推荐展示/导出）
type Event struct {
	TS       time.Time `json:"ts"`
	Type     string    `json:"type"` // e.g. recommend_show
	Source   string    `json:"source"`
	SourceID string    `json:"source_id"`
	Title    string    `json:"title,omitempty"`
}

type Service struct {
	dir             string
	ttlDays         int
	shortWindowDays int
	cachePath       string
}

func New(dir string, ttlDays, shortWindowDays int) (*Service, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户目录失败: %w", err)
		}
		dir = filepath.Join(home, ".quicksearch", "memory")
	}
	if ttlDays <= 0 {
		ttlDays = 30
	}
	if shortWindowDays <= 0 {
		shortWindowDays = 7
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建记忆目录失败: %w", err)
	}
	return &Service{
		dir:             dir,
		ttlDays:         ttlDays,
		shortWindowDays: shortWindowDays,
		cachePath:       filepath.Join(dir, "profile-cache.json"),
	}, nil
}

// RecordRecommended 记录被推荐给用户的论文（用于短期去重）
func (s *Service) RecordRecommended(papers []Event) error {
	if len(papers) == 0 {
		return nil
	}
	filename := filepath.Join(s.dir, fmt.Sprintf("events-%s.jsonl", time.Now().Format("20060102")))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("打开事件文件失败: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, ev := range papers {
		if ev.TS.IsZero() {
			ev.TS = time.Now().UTC()
		}
		if err := enc.Encode(ev); err != nil {
			return fmt.Errorf("写入事件失败: %w", err)
		}
	}
	return nil
}

// LoadRecentPaperKeys 返回短期窗口内出现过的论文 key（source:source_id），用于避免重复推荐
func (s *Service) LoadRecentPaperKeys() (map[string]struct{}, error) {
	keys := make(map[string]struct{})
	cutoff := time.Now().AddDate(0, 0, -s.shortWindowDays)

	files, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("读取记忆目录失败: %w", err)
	}
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if !strings.HasPrefix(name, "events-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "events-"), ".jsonl")
		t, err := time.Parse("20060102", datePart)
		if err != nil || t.Before(cutoff) {
			continue
		}
		if err := s.readKeys(filepath.Join(s.dir, name), cutoff, keys); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func (s *Service) readKeys(path string, cutoff time.Time, keys map[string]struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开事件文件失败: %w", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.TS.IsZero() || ev.TS.Before(cutoff) {
			continue
		}
		if ev.Source == "" || ev.SourceID == "" {
			continue
		}
		key := fmt.Sprintf("%s:%s", ev.Source, ev.SourceID)
		keys[key] = struct{}{}
	}
	return scanner.Err()
}

type ProfileCache struct {
	UpdatedAt          time.Time          `json:"updated_at"`
	TopKeywords        []string           `json:"top_keywords"`
	PlatformPreference map[string]float64 `json:"platform_pref"`
	VectorModel        string             `json:"vector_model"`
	Vector             []float64          `json:"vector"`
}

func (s *Service) BuildProfile(events []Event, topN int, embedFunc func(texts []string) ([]float64, error), vectorModel string) *ProfileCache {
	if len(events) == 0 {
		return nil
	}
	if topN <= 0 {
		topN = 10
	}

	kwFreq := make(map[string]int)
	platformFreq := make(map[string]int)
	var texts []string

	for _, ev := range events {
		if ev.Title != "" {
			for _, token := range strings.Fields(strings.ToLower(ev.Title)) {
				token = strings.Trim(token, " ,.;:()[]{}\"'`")
				if token == "" {
					continue
				}
				kwFreq[token]++
			}
			texts = append(texts, ev.Title)
		}
		if ev.Source != "" {
			platformFreq[ev.Source]++
		}
	}

	topKeywords := topKFromMap(kwFreq, topN)
	platformPref := normIntMap(platformFreq)

	var vec []float64
	if embedFunc != nil && len(texts) > 0 {
		if v, err := embedFunc(texts); err == nil {
			vec = v
		}
	}

	return &ProfileCache{
		UpdatedAt:          time.Now(),
		TopKeywords:        topKeywords,
		PlatformPreference: platformPref,
		VectorModel:        vectorModel,
		Vector:             vec,
	}
}

func topKFromMap(freq map[string]int, k int) []string {
	type kv struct {
		Key string
		Val int
	}
	arr := make([]kv, 0, len(freq))
	for k0, v := range freq {
		arr = append(arr, kv{Key: k0, Val: v})
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].Val > arr[j].Val
	})
	if k > len(arr) {
		k = len(arr)
	}
	out := make([]string, 0, k)
	for i := 0; i < k; i++ {
		out = append(out, arr[i].Key)
	}
	return out
}

func normIntMap(m map[string]int) map[string]float64 {
	sum := 0
	for _, v := range m {
		sum += v
	}
	if sum == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = float64(v) / float64(sum)
	}
	return out
}

// LoadEvents 加载最近 windowDays 内的事件
func (s *Service) LoadEvents(windowDays int) ([]Event, error) {
	cutoff := time.Now().AddDate(0, 0, -windowDays)
	var events []Event
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("读取记忆目录失败: %w", err)
	}
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if !strings.HasPrefix(name, "events-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "events-"), ".jsonl")
		t, err := time.Parse("20060102", datePart)
		if err != nil || t.Before(cutoff) {
			continue
		}
		path := filepath.Join(s.dir, name)
		evs, err := s.readEvents(path, cutoff)
		if err != nil {
			return nil, err
		}
		events = append(events, evs...)
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].TS.Before(events[j].TS)
	})
	return events, nil
}

func (s *Service) readEvents(path string, cutoff time.Time) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开事件文件失败: %w", err)
	}
	defer f.Close()
	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.TS.IsZero() || ev.TS.Before(cutoff) {
			continue
		}
		events = append(events, ev)
	}
	return events, scanner.Err()
}

// LoadProfileCache 读取画像缓存
func (s *Service) LoadProfileCache() (*ProfileCache, error) {
	f, err := os.Open(s.cachePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var pc ProfileCache
	if err := json.NewDecoder(f).Decode(&pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

// SaveProfileCache 写入画像缓存
func (s *Service) SaveProfileCache(pc *ProfileCache) error {
	if pc == nil {
		return fmt.Errorf("profile cache is nil")
	}
	f, err := os.OpenFile(s.cachePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(pc)
}

func (s *Service) ClearCache() {
	_ = os.Remove(s.cachePath)
}

func (s *Service) Cleanup() {
	cutoff := time.Now().AddDate(0, 0, -s.ttlDays)
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if !strings.HasPrefix(name, "events-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "events-"), ".jsonl")
		t, err := time.Parse("20060102", datePart)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			_ = os.Remove(filepath.Join(s.dir, name))
		}
	}
}
