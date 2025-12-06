import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Separator } from './ui/separator';
import { Save, RefreshCw, Settings, Sparkles, Database as DatabaseIcon, BookOpen, Globe, Palette, Moon, Sun, Monitor } from 'lucide-react';
import { GetConfig, UpdateConfig } from '../../wailsjs/go/main/App';
import * as models from '../../wailsjs/go/models';
import { useToast } from './ui/use-toast';
import { useTheme } from './ThemeProvider';

interface Config {
  embedder: {
    baseurl: string;
    apikey: string;
    model: string;
    dim: number;
  };
  database: {
    path: string;
  };
  zotero: {
    user_id: string;
    api_key: string;
  };
  arxiv: {
    proxy: string;
    step: number;
    timeout: number;
  };
  openreview: {
    proxy: string;
    timeout: number;
  };
}

const SettingsView: React.FC = () => {
  const [config, setConfig] = useState<models.config.AppConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();
  const { theme, setTheme } = useTheme();

  const loadConfig = async () => {
    setLoading(true);
    try {
      const cfg = await GetConfig();
      setConfig(cfg);
      
      toast({
        title: "配置加载成功",
        description: "已获取最新配置信息",
        duration: 2000, // 3秒后消失
      });
    } catch (error) {
      console.error('Failed to load config:', error);
      toast({
        title: "配置加载失败",
        description: "无法获取配置信息，请重试",
        variant: "destructive",
        duration: 3000, // 错误信息显示5秒
      });
    } finally {
      setLoading(false);
    }
  };

  const saveConfig = async () => {
    if (!config) return;
    
    setSaving(true);
    try {
      await UpdateConfig(config);
      console.log('Saving config:', config);
      
      toast({
        title: "配置保存成功",
        description: "应用已成功重载，新配置已生效",
        duration: 4000, // 成功信息显示4秒
      });
    } catch (error) {
      console.error('Failed to save config:', error);
      toast({
        title: "配置保存失败",
        description: "请检查配置是否正确，然后重试",
        variant: "destructive",
        duration: 6000, // 错误信息显示6秒，让用户有足够时间阅读
      });
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    loadConfig();
  }, []);

  if (loading || !config) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-4 text-muted-foreground" />
          <p className="text-muted-foreground">Loading configuration...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="space-y-2">
              <div className="flex items-center gap-3">

                <CardTitle className="text-3xl font-display font-semibold ">Settings</CardTitle>
              </div>
              <CardDescription className="text-base text-muted-foreground ml-13">
                配置密钥等相关内容
              </CardDescription>
            </div>
            
            <div className="flex items-center gap-2">
              <Button
                onClick={loadConfig}
                disabled={loading}
                size="sm"
                variant="outline"
              >
                <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                Reload
              </Button>
              
              <Button
                onClick={saveConfig}
                disabled={saving}
                size="sm"
              >
                <Save className="mr-2 h-4 w-4" />
                {saving ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto overflow-x-hidden px-8 py-8" style={{ overflowY: 'auto' }}>
          <div className="max-w-4xl mx-auto space-y-8">
            {/* Appearance Settings */}
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                  <Palette className="w-4 h-4 text-primary" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">
                  Appearance
                </h3>
                <span className="text-sm text-muted-foreground">(界面外观)</span>
              </div>
              <div>
                <label className="text-sm font-medium block mb-3">Theme Mode</label>
                <div className="grid grid-cols-3 gap-4">
                  <Button
                    variant={theme === 'light' ? 'default' : 'outline'}
                    className={`justify-start h-auto py-3 px-4 ${theme === 'light' ? 'ring-2 ring-primary ring-offset-2' : ''}`}
                    onClick={() => setTheme('light')}
                  >
                    <Sun className="w-4 h-4 mr-2" />
                    <div className="flex flex-col items-start">
                      <span className="font-medium">Light</span>
                      <span className="text-xs opacity-70">明亮模式</span>
                    </div>
                  </Button>
                  <Button
                    variant={theme === 'dark' ? 'default' : 'outline'}
                    className={`justify-start h-auto py-3 px-4 ${theme === 'dark' ? 'ring-2 ring-primary ring-offset-2' : ''}`}
                    onClick={() => setTheme('dark')}
                  >
                    <Moon className="w-4 h-4 mr-2" />
                    <div className="flex flex-col items-start">
                      <span className="font-medium">Dark</span>
                      <span className="text-xs opacity-70">暗黑模式</span>
                    </div>
                  </Button>
                  <Button
                    variant={theme === 'system' ? 'default' : 'outline'}
                    className={`justify-start h-auto py-3 px-4 ${theme === 'system' ? 'ring-2 ring-primary ring-offset-2' : ''}`}
                    onClick={() => setTheme('system')}
                  >
                    <Monitor className="w-4 h-4 mr-2" />
                    <div className="flex flex-col items-start">
                      <span className="font-medium">System</span>
                      <span className="text-xs opacity-70">跟随系统</span>
                    </div>
                  </Button>
                </div>
              </div>
            </div>

            <Separator className="bg-border/50" />

            {/* Embedder Configuration */}
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                  <Sparkles className="w-4 h-4 text-primary" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">
                  Embedding Service
                </h3>
                <span className="text-sm text-muted-foreground">(向量搜索相关)</span>
              </div>
              <div className="grid gap-4">
                <div>
                  <label className="text-sm font-medium block mb-2">API Base URL</label>
                  <input
                    type="text"
                    value={config.Embedder.BaseURL}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Embedder: { ...config.Embedder, BaseURL: e.target.value }
                      })
                    )}
                    className="w-full px-4 py-3 bg-background/50 border border-input rounded-xl text-sm focus:ring-2 focus:ring-primary/20 transition-all"
                    placeholder="国内推荐: https://api.siliconflow.cn/v1"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium">API Key</label>
                  <input
                    type="password"
                    value={config.Embedder.APIKey}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Embedder: { ...config.Embedder, APIKey: e.target.value }
                      })
                    )}
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="sk-..."
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium">Model Name</label>
                    <input
                      type="text"
                      value={config.Embedder.ModelName}
                      onChange={(e) => setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          Embedder: { ...config.Embedder, ModelName: e.target.value }
                        })
                      )}
                      className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                      placeholder="text-embedding-3-small"
                    />
                  </div>

                  <div>
                    <label className="text-sm font-medium">Dimension</label>
                    <input
                      type="number"
                      value={config.Embedder.Dim}
                      onChange={(e) => setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          Embedder: { ...config.Embedder, Dim: parseInt(e.target.value) }
                        })
                      )}
                      className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                      placeholder="1536"
                    />
                  </div>
                </div>
              </div>
            </div>

            <Separator className="bg-border/50" />

            {/* Database Configuration */}
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-info/10 flex items-center justify-center">
                  <DatabaseIcon className="w-4 h-4 text-info" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">Database</h3>
              </div>
              <div>
                <label className="text-sm font-medium">Database Path</label>
                <input
                  type="text"
                  value={config.Database.Path}
                  onChange={(e) => setConfig(
                    models.config.AppConfig.createFrom({
                      ...config,
                      Database: { ...config.Database, Path: e.target.value }
                    })
                  )}
                  className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                  placeholder="data/quicksearch.db"
                />
              </div>
            </div>

            <Separator className="bg-border/50" />

            {/* Zotero Integration */}
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-success/10 flex items-center justify-center">
                  <BookOpen className="w-4 h-4 text-success" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">Zotero Integration</h3>
              </div>
              <div className="grid gap-4">
                <div>
                  <label className="text-sm font-medium">User ID</label>
                  <input
                    type="text"
                    value={config.Zotero.UserID}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Zotero: { ...config.Zotero, UserID: e.target.value }
                      })
                    )}
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="12345678"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium">API Key</label>
                  <input
                    type="password"
                    value={config.Zotero.APIKey}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Zotero: { ...config.Zotero, APIKey: e.target.value }
                      })
                    )}
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="Zotero API key"
                  />
                </div>
              </div>
            </div>

            {/* FeiShu Integration */}
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-success/10 flex items-center justify-center">
                  <BookOpen className="w-4 h-4 text-success" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">FeiShu Integration</h3>
              </div>

              <div className="grid gap-4">
                <div>
                  <label className="text-sm font-medium">App ID</label>
                  <input
                    type="text"
                    value={config.FeiShu.AppID}
                    onChange={(e) =>
                      setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          FeiShu: models.core.FeiShuConfig.createFrom({
                            AppID: e.target.value,
                            AppSecret: config.FeiShu.AppSecret,
                          }),
                        })
                      )
                    }
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="cli_xxx"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium">App Secret</label>
                  <input
                    type="password"
                    value={config.FeiShu.AppSecret}
                    onChange={(e) =>
                      setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          FeiShu: models.core.FeiShuConfig.createFrom({
                            AppID: config.FeiShu.AppID,
                            AppSecret: e.target.value,
                          }),
                        })
                      )
                    }
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="******"
                  />
                </div>
              </div>
            </div>

            <Separator className="bg-border/50" />

            {/* Platform Settings */}
            <div className="glass-card p-6 rounded-2xl space-y-8">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-warning/10 flex items-center justify-center">
                  <Globe className="w-4 h-4 text-warning" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">Platform Settings</h3>
              </div>
              
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-medium mb-3">arXiv</h4>
                  <div className="grid gap-4">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Proxy</label>
                      <input
                        type="text"
                        value={config.Arxiv.Proxy}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            Arxiv: { ...config.Arxiv, Proxy: e.target.value }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        placeholder="http://127.0.0.1:7890"
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Step Size</label>
                        <input
                          type="number"
                          value={config.Arxiv.Step}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              Arxiv: { ...config.Arxiv, Step: parseInt(e.target.value)}
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        />
                      </div>

                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Timeout (s)</label>
                        <input
                          type="number"
                          value={config.Arxiv.Timeout}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              Arxiv: { ...config.Arxiv, Timeout: parseInt(e.target.value) }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="text-sm font-medium mb-3">ACL Anthology</h4>
                  <div className="grid gap-4">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Base URL</label>
                      <input
                        type="text"
                        value={config.ACL.BaseURL}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            ACL: { ...config.ACL, BaseURL: e.target.value }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        placeholder="https://aclanthology.org"
                      />
                    </div>

                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Proxy</label>
                      <input
                        type="text"
                        value={config.ACL.Proxy}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            ACL: { ...config.ACL, Proxy: e.target.value }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        placeholder="http://127.0.0.1:7890"
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Step Size</label>
                        <input
                          type="number"
                          value={config.ACL.Step}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              ACL: { ...config.ACL, Step: parseInt(e.target.value) }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        />
                      </div>

                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Timeout (s)</label>
                        <input
                          type="number"
                          value={config.ACL.Timeout}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              ACL: { ...config.ACL, Timeout: parseInt(e.target.value) }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                          placeholder="600"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div className="flex items-center space-x-2">
                        <input
                          type="checkbox"
                          id="acl-use-rss"
                          checked={config.ACL.UseRSS}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              ACL: { ...config.ACL, UseRSS: e.target.checked }
                            })
                          )}
                          className="rounded border-input"
                        />
                        <label htmlFor="acl-use-rss" className="text-sm font-medium text-muted-foreground">
                          Use RSS (获取最新 1000 篇)
                        </label>
                      </div>

                      <div className="flex items-center space-x-2">
                        <input
                          type="checkbox"
                          id="acl-use-bibtex"
                          checked={config.ACL.UseBibTeX}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              ACL: { ...config.ACL, UseBibTeX: e.target.checked }
                            })
                          )}
                          className="rounded border-input"
                        />
                        <label htmlFor="acl-use-bibtex" className="text-sm font-medium text-muted-foreground">
                          Use BibTeX (全量数据)
                        </label>
                      </div>
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="text-sm font-medium mb-3">OpenReview</h4>
                  <div className="grid gap-4">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Proxy</label>
                      <input
                        type="text"
                        value={config.OpenReview.Proxy}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            OpenReview: { ...config.OpenReview, Proxy: e.target.value }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        placeholder="http://127.0.0.1:7890"
                      />
                    </div>

                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Timeout (s)</label>
                      <input
                        type="number"
                        value={config.OpenReview.Timeout}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            OpenReview: { ...config.OpenReview, Timeout: parseInt(e.target.value) }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                      />
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="text-sm font-medium mb-3">SSRN</h4>
                  <div className="grid gap-4">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Base URL</label>
                      <input
                        type="text"
                        value={config.SSRN.BaseURL}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            SSRN: { ...config.SSRN, BaseURL: e.target.value }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        placeholder="https://papers.ssrn.com"
                      />
                    </div>

                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Proxy</label>
                      <input
                        type="text"
                        value={config.SSRN.Proxy}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            SSRN: { ...config.SSRN, Proxy: e.target.value }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        placeholder="http://127.0.0.1:7890"
                      />
                    </div>

                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Timeout (s)</label>
                      <input
                        type="number"
                        value={config.SSRN.Timeout}
                        onChange={(e) => setConfig(
                          models.config.AppConfig.createFrom({
                            ...config,
                            SSRN: { ...config.SSRN, Timeout: parseInt(e.target.value) }
                          })
                        )}
                        className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Page Size</label>
                        <input
                          type="number"
                          value={config.SSRN.PageSize}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              SSRN: { ...config.SSRN, PageSize: parseInt(e.target.value) }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        />
                      </div>

                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Max Pages</label>
                        <input
                          type="number"
                          value={config.SSRN.MaxPages}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              SSRN: { ...config.SSRN, MaxPages: parseInt(e.target.value) }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Rate Limit (per second)</label>
                        <input
                          type="number"
                          step="0.1"
                          value={config.SSRN.RateLimitPerSecond}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              SSRN: { ...config.SSRN, RateLimitPerSecond: parseFloat(e.target.value) }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                        />
                      </div>

                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Sort</label>
                        <input
                          type="text"
                          value={config.SSRN.Sort}
                          onChange={(e) => setConfig(
                            models.config.AppConfig.createFrom({
                              ...config,
                              SSRN: { ...config.SSRN, Sort: e.target.value }
                            })
                          )}
                          className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                          placeholder="AB_Date_D"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <Separator className="bg-border/50" />

            {/* LLM Configuration */}
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                  <Sparkles className="w-4 h-4 text-primary" />
                </div>
                <h3 className="text-lg font-display font-semibold text-foreground">
                  LLM Configuration
                </h3>
                <span className="text-sm text-muted-foreground">(用于 Agent)</span>
              </div>
              <div className="grid gap-4">
                <div>
                  <label className="text-sm font-medium block mb-2">API Base URL</label>
                  <input
                    type="text"
                    value={config.LLM?.BaseURL || ''}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        LLM: models.config.LLMConfig.createFrom({
                          BaseURL: e.target.value,
                          ModelName: config.LLM?.ModelName || '',
                          APIKey: config.LLM?.APIKey || ''
                        })
                      })
                    )}
                    className="w-full px-4 py-3 bg-background/50 border border-input rounded-xl text-sm focus:ring-2 focus:ring-primary/20 transition-all"
                    placeholder="https://openrouter.ai/api/v1"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium">Model Name</label>
                  <input
                    type="text"
                    value={config.LLM?.ModelName || ''}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        LLM: models.config.LLMConfig.createFrom({
                          BaseURL: config.LLM?.BaseURL || '',
                          ModelName: e.target.value,
                          APIKey: config.LLM?.APIKey || ''
                        })
                      })
                    )}
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="deepseek/deepseek-v3"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium">API Key</label>
                  <input
                    type="password"
                    value={config.LLM?.APIKey || ''}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        LLM: models.config.LLMConfig.createFrom({
                          BaseURL: config.LLM?.BaseURL || '',
                          ModelName: config.LLM?.ModelName || '',
                          APIKey: e.target.value
                        })
                      })
                    )}
                    className="mt-1 w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                    placeholder="sk-..."
                  />
                </div>
              </div>
            </div>
          </div>
        </CardContent>

          <div className="border-t border-border/30 px-8 py-4 bg-card/30 backdrop-blur-sm flex-shrink-0">
          <div className="flex items-center gap-3 max-w-4xl mx-auto">
            <div className="w-8 h-8 rounded-lg bg-info/10 flex items-center justify-center flex-shrink-0">
              <Settings className="w-4 h-4 text-info" />
            </div>
            <p className="text-sm text-muted-foreground">
              更新配置后请先保存再重载
            </p>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default SettingsView;

