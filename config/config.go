package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"

	"PaperHunter/internal/core"
	emb "PaperHunter/internal/embedding"
	"PaperHunter/internal/platform/acl"
	"PaperHunter/internal/platform/arxiv"
	"PaperHunter/internal/platform/openreview"
	"PaperHunter/internal/platform/ssrn"
	"PaperHunter/pkg/logger"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `mapstructure:"path" yaml:"path"` // 数据库文件路径
}

// LLMConfig LLM 配置（用于 Agent）
type LLMConfig struct {
	BaseURL   string `mapstructure:"base_url" yaml:"base_url"` // API 地址，支持 OpenAI 兼容的 API
	ModelName string `mapstructure:"model" yaml:"model"`       // 模型名称
	APIKey    string `mapstructure:"api_key" yaml:"api_key"`   // API Key
}

// AppConfig 应用总配置(全局 + 平台)
type AppConfig struct {
	Env        string             `mapstructure:"env" yaml:"env"`               // 运行环境:dev/prod
	Embedder   emb.EmbedderConfig `mapstructure:"embedder" yaml:"embedder"`     // Embedder 配置
	Database   DatabaseConfig     `mapstructure:"database" yaml:"database"`     // 数据库配置
	Zotero     core.ZoteroConfig  `mapstructure:"zotero" yaml:"zotero"`         // Zotero 配置
	FeiShu     core.FeiShuConfig  `mapstructure:"feishu" yaml:"feishu"`         // 飞书配置
	Arxiv      arxiv.Config       `mapstructure:"arxiv" yaml:"arxiv"`           // arXiv 平台配置
	OpenReview openreview.Config  `mapstructure:"openreview" yaml:"openreview"` // OpenReview 平台配置
	ACL        acl.Config         `mapstructure:"acl" yaml:"acl"`               // ACL Anthology 平台配置
	SSRN       ssrn.Config        `mapstructure:"ssrn" yaml:"ssrn"`             // SSRN 平台配置
	LLM        LLMConfig          `mapstructure:"agent" yaml:"agent"`           // LLM 配置（用于 Agent，兼容 yaml 中的 agent 键）
}

var (
	global     *AppConfig
	once       sync.Once
	globalErr  error
	configPath string // 存储当前使用的配置文件路径
)

func setDefaults(v *viper.Viper) {

	homedir, _ := os.UserHomeDir()
	dataBasePath := filepath.Join(homedir, ".quicksearch", "data", "quicksearch.db")
	v.SetDefault("env", "prod")
	v.SetDefault("database.path", dataBasePath)

	v.SetDefault("arxiv.use_api", false)
	v.SetDefault("arxiv.proxy", "")
	v.SetDefault("arxiv.step", 50)
	v.SetDefault("arxiv.timeout", 30)
	v.SetDefault("arxiv.api_base", "https://export.arxiv.org/api/query")
	v.SetDefault("arxiv.web_base", "https://arxiv.org/search/advanced")

	v.SetDefault("openreview.api_base", "https://api2.openreview.net")
	v.SetDefault("openreview.proxy", "")
	v.SetDefault("openreview.timeout", 30)

	v.SetDefault("acl.base_url", "https://aclanthology.org")
	v.SetDefault("acl.timeout", "30s")
	v.SetDefault("acl.proxy", "")
	v.SetDefault("acl.step", 100)
	v.SetDefault("acl.use_rss", true)
	v.SetDefault("acl.use_bibtex", false)

	// SSRN 默认值
	v.SetDefault("ssrn.base_url", "https://papers.ssrn.com")
	v.SetDefault("ssrn.timeout", "30s")
	v.SetDefault("ssrn.proxy", "")
	v.SetDefault("ssrn.page_size", 20)
	v.SetDefault("ssrn.max_pages", 3)
	v.SetDefault("ssrn.rate_limit_per_second", 1.0)
	v.SetDefault("ssrn.sort", "AB_Date_D")
	// Embedder 默认值
	v.SetDefault("embedder.baseurl", "")
	v.SetDefault("embedder.apikey", "")
	v.SetDefault("embedder.model", "Qwen/Qwen3-Embedding-4B")
	v.SetDefault("embedder.dim", 2560)

	// Zotero 默认值
	v.SetDefault("zotero.user_id", "")
	v.SetDefault("zotero.api_key", "")

	// 飞书默认值
	v.SetDefault("feishu.app_id", "")
	v.SetDefault("feishu.app_secret", "")

	// LLM 默认值（使用 agent 作为键名以兼容现有配置）
	v.SetDefault("agent.base_url", "https://openrouter.ai/api/v1")
	v.SetDefault("agent.model", "deepseek/deepseek-v3")
	v.SetDefault("agent.api_key", "")
}

// 可额外传入目录或具体文件路径
func Init(configPaths ...string) (*AppConfig, error) {
	once.Do(func() {
		v := viper.New()
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		homedir, _ := os.UserHomeDir()
		configDir := filepath.Join(homedir, ".quicksearch", "config")
		os.MkdirAll(configDir, 0755)

		v.AddConfigPath("./config")
		v.AddConfigPath("../config")
		v.AddConfigPath(".")
		v.AddConfigPath("~/.quciksearch/config")
		v.AddConfigPath("config")
		v.AddConfigPath(configDir)

		for _, p := range configPaths {
			if p == "" {
				continue
			}
			if strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml") {
				v.SetConfigFile(p)
			} else {
				v.AddConfigPath(p)
			}
		}

		v.SetEnvPrefix("QSP")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		setDefaults(v)

		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				globalErr = fmt.Errorf("读取配置文件失败: %w", err)
				return
			}
			// 配置文件不存在，创建示例配置文件
			if err := CreateExampleConfig(); err != nil {
				globalErr = fmt.Errorf("创建示例配置文件失败: %w", err)
				return
			}
		} else {
			configPath = v.ConfigFileUsed()
		}

		cfg := &AppConfig{}
		if err := v.Unmarshal(&cfg); err != nil {
			globalErr = fmt.Errorf("配置解析失败: %w", err)
			return
		}

		// 验证 arxiv 配置
		if err := cfg.Arxiv.Validate(); err != nil {
			globalErr = fmt.Errorf("arxiv 配置不合法: %w", err)
			return
		}

		// 验证 openreview 配置
		if err := cfg.OpenReview.Validate(); err != nil {
			globalErr = fmt.Errorf("openreview 配置不合法: %w", err)
			return
		}

		// 验证 acl 配置
		if err := cfg.ACL.Validate(); err != nil {
			globalErr = fmt.Errorf("acl 配置不合法: %w", err)
			return
		}

		global = cfg
	})
	return global, globalErr
}

func MustInit(configPaths ...string) *AppConfig {
	cfg, err := Init(configPaths...)
	if err != nil {
		panic(err)
	}
	return cfg
}

func Get() *AppConfig {
	if global == nil {
		_, _ = Init()
	}
	return global
}

func GetConfigPath() string {
	if configPath == "" {

		_, _ = Init()
	}
	return configPath
}

type ConfigManager struct {
	configPath string
}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configPath: GetConfigPath(),
	}
}

func (cm *ConfigManager) GetConfigPath() string {
	if cm.configPath == "" {
		cm.configPath = GetConfigPath()
		if cm.configPath == "" {
			homeDir, _ := os.UserHomeDir()
			cm.configPath = filepath.Join(homeDir, ".quicksearch", "config", "config.yaml")
		}
	}
	return cm.configPath
}

func (cm *ConfigManager) GetConfigDir() string {
	return filepath.Dir(cm.GetConfigPath())
}

func CreateExampleConfig() error {
	homedir, _ := os.UserHomeDir()
	configDir := filepath.Join(homedir, ".quicksearch", "config")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，创建配置文件
			exampleContent := `# QuickSearchPaper 配置文件
# 请根据你的需求修改以下配置

# Embedding 服务配置（用于语义搜索）
embedder:
  baseurl: "https://api.siliconflow.cn/v1"  # 或使用 OpenAI API: "https://api.openai.com/v1"
  apikey: "your-api-key-here"               # 请替换为你的 API Key
  model: "Qwen/Qwen3-Embedding-4B"          # 或使用 OpenAI: "text-embedding-3-small"
  dim: 2560                                 # 向量维度

# 数据库配置
database:
  path: ""  #可以配置后重新初始化，从而指定你的数据库保存位置

# Zotero 配置（可选）
zotero:
  user_id: ""     # 你的 Zotero 用户 ID
  api_key: ""     # 你的 Zotero API Key

# 飞书配置（可选）
feishu:
  app_id: ""      # 飞书应用 ID
  app_secret: ""  # 飞书应用密钥

# arXiv 平台配置
arxiv:
  use_api: false  # 是否使用官方 API（推荐）
  proxy: ""       # 代理设置，如: "http://127.0.0.1:7890"
  step: 50
  timeout: 30

# OpenReview 平台配置
openreview:
  proxy: ""       # 代理设置
  timeout: 30

# ACL Anthology 平台配置
acl:
  proxy: ""       # 代理设置
  timeout: 600

# LLM 配置（用于 Agent）
agent:
  base_url: "https://openrouter.ai/api/v1"  # API 地址，支持 OpenAI 兼容的 API
  model: "deepseek/deepseek-v3"            # 模型名称
  api_key: ""                               # API Key（如果留空，将尝试使用 embedder 的 api_key）
`

			if err := os.WriteFile(configFile, []byte(exampleContent), 0644); err != nil {
				return fmt.Errorf("写入配置文件失败: %w", err)
			}
			logger.Info("已在 %s 中创建配置文件", configFile)
			fmt.Printf("已创建示例配置文件: %s\n", configFile)
			fmt.Println("请编辑配置文件，设置你的 API Key 和其他配置")
			return nil
		} else {
			// 其他错误（权限问题、路径问题等）
			return fmt.Errorf("检查配置文件时出错: %w", err)
		}
	} else {
		// 文件存在
		logger.Warn("home 目录下已存在配置文件，请前往编辑即可")
		return nil
	}
}
