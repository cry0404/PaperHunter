package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"PaperHunter/config"
	"PaperHunter/internal/core"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"

	"gopkg.in/yaml.v2"
)

func (a *App) GetConfig() (*config.AppConfig, error) {
	if a.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}
	return a.config, nil
}

func (a *App) UpdateConfig(cfg *config.AppConfig) error {
	oldConfig := a.config

	if err := a.validateConfig(cfg); err != nil {
		return fmt.Errorf("验证配置失败: %w", err)
	}

	if err := a.copyDatabaseIfPathChanged(oldConfig, cfg); err != nil {
		return fmt.Errorf("复制数据库失败: %w", err)
	}

	if err := a.saveConfig(cfg); err != nil {

		return fmt.Errorf("保存配置失败: %w", err)
	}

	if err := a.reloadCoreApp(cfg); err != nil {

		logger.Error("重载失败，尝试恢复旧配置: %v", err)
		if oldConfig != nil {
			if rollbackErr := a.reloadCoreApp(oldConfig); rollbackErr != nil {
				logger.Error("恢复旧配置也失败: %v", rollbackErr)
			}
		}
		return fmt.Errorf("重载 app 失败: %w", err)
	}

	// 更新内存配置并重新初始化 HyDE（确保 LLM 配置生效）
	a.config = cfg
	a.initHyDE()

	logger.Info("配置更新并重载成功")
	return nil
}

func (a *App) validateConfig(cfg *config.AppConfig) error {
	if cfg.Embedder.APIKey == "" {
		return fmt.Errorf("请配置对应的 apikey")
	}
	if cfg.Embedder.ModelName == "" {
		return fmt.Errorf("请配置对应的模型名字")
	}
	if cfg.Embedder.Dim == 0 {
		logger.Error("请配置对应 embedding 模型的纬度")
		return fmt.Errorf("请配置对应 embedding 模型的纬度")
	}

	if cfg.OpenReview.Timeout <= 20 {
		cfg.OpenReview.Timeout = 20
	}
	if cfg.Arxiv.Timeout <= 20 {
		cfg.Arxiv.Timeout = 20
	}

	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}
	logger.Debug("配置验证成功")
	return nil
}

func (a *App) saveConfig(cfg *config.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	configManager := config.NewConfigManager()
	configPath := configManager.GetConfigPath()
	configDir := configManager.GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	logger.Debug("Configuration saved to file: %s", configPath)
	return nil
}

func (a *App) reloadCoreApp(cfg *config.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	coreApp, err := core.NewApp(cfg.Database.Path, cfg.Embedder,
		map[string]platform.Config{
			"arxiv":      &cfg.Arxiv,
			"openreview": &cfg.OpenReview,
			"acl":        &cfg.ACL,
			"ssrn":       &cfg.SSRN,
		}, cfg.Zotero, cfg.FeiShu)

	if err != nil {
		return fmt.Errorf("重新初始化核心模块失败: %w", err)
	}

	if a.coreApp != nil {
		_ = a.coreApp.Close()
		logger.Debug("Closing old core application instance")
	}

	a.coreApp = coreApp
	logger.Debug("Core application reloaded with new config")
	return nil
}

func (a *App) ReloadConfig() error {

	cfg, err := config.Init("")
	if err != nil {
		return fmt.Errorf("重新加载配置文件失败: %w", err)
	}

	return a.UpdateConfig(cfg)
}

func defaultDBPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".quicksearch", "data", "quicksearch.db")
}

func (a *App) copyDatabaseIfPathChanged(oldCfg, newCfg *config.AppConfig) error {
	if newCfg == nil {
		return fmt.Errorf("新配置不能为空")
	}

	oldPath := defaultDBPath()
	if oldCfg != nil && strings.TrimSpace(oldCfg.Database.Path) != "" {
		oldPath = strings.TrimSpace(oldCfg.Database.Path)
	}

	newPath := strings.TrimSpace(newCfg.Database.Path)
	if newPath == "" || filepath.Clean(newPath) == filepath.Clean(oldPath) {
		return nil
	}

	if a.coreApp != nil {
		_ = a.coreApp.Close()
		a.coreApp = nil
		logger.Info("已关闭旧数据库连接，准备复制到新路径")
	}

	if _, err := os.Stat(oldPath); err != nil {
		return fmt.Errorf("原数据库不存在或不可访问: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	if err := copyFileForce(oldPath, newPath); err != nil {
		return fmt.Errorf("复制数据库文件失败: %w", err)
	}

	for _, suf := range []string{"-wal", "-shm"} {
		src := oldPath + suf
		if _, err := os.Stat(src); err == nil {
			_ = copyFileForce(src, newPath+suf)
		}
	}

	logger.Info("数据库已复制到新路径: %s", newPath)
	return nil
}

func copyFileForce(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
