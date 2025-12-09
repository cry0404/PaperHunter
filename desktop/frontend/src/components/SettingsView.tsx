import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Separator } from './ui/separator';
import { 
  Save, 
  RefreshCw, 
  Settings, 
  Sparkles, 
  Database as DatabaseIcon, 
  BookOpen, 
  Globe, 
  Palette, 
  Moon, 
  Sun, 
  Monitor, 
  Languages,
  BookMarked,
  Share2
} from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
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
  llm?: {
    baseurl: string;
    modelname: string;
    apikey: string;
  };
  zotero?: {
    userid: string;
    apikey: string;
    librarytype: string;
  };
  feishu?: {
    appid: string;
    appsecret: string;
  };
}

const SettingsView: React.FC = () => {
  const { t, i18n } = useTranslation();
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
        title: t('common.success'),
        description: "Configuration loaded successfully",
        duration: 2000,
      });
    } catch (error) {
      console.error('Failed to load config:', error);
      toast({
        title: t('common.error'),
        description: "Failed to load configuration",
        variant: "destructive",
        duration: 3000,
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
        title: t('common.success'),
        description: "Configuration saved successfully",
        duration: 3000,
      });
    } catch (error) {
      console.error('Failed to save config:', error);
      toast({
        title: t('common.error'),
        description: "Failed to save configuration",
        variant: "destructive",
        duration: 5000,
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
      <div className="flex items-center justify-center h-full bg-background">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-4 text-muted-foreground" />
          <p className="text-muted-foreground font-sans">{t('common.loading')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <CardTitle className="text-3xl font-sans font-medium tracking-tight">{t('settings.title')}</CardTitle>
              </div>
              <CardDescription className="text-base text-muted-foreground font-serif">
                {t('settings.subtitle')}
              </CardDescription>
            </div>
            
            <div className="flex items-center gap-2">
              <Button
                onClick={loadConfig}
                disabled={loading}
                size="sm"
                variant="outline"
                className="font-sans"
              >
                <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                {t('settings.reload')}
              </Button>
              
              <Button
                onClick={saveConfig}
                disabled={saving}
                size="sm"
                className="font-sans bg-anthropic-dark text-anthropic-light hover:bg-anthropic-dark/90"
              >
                <Save className="mr-2 h-4 w-4" />
                {saving ? t('settings.saving') : t('settings.save')}
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-4xl mx-auto space-y-8 pb-12">
            
            {/* Appearance Settings */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <Palette className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-sans font-medium text-foreground">
                  {t('settings.appearance.title')}
                </h3>
              </div>
              <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-6">
                
                {/* Theme Mode */}
              <div>
                  <label className="text-sm font-medium block mb-4 font-sans">{t('settings.appearance.theme')}</label>
                <div className="grid grid-cols-3 gap-4">
                    {(['light', 'dark', 'system'] as const).map((mode) => (
                  <Button
                        key={mode}
                        variant={theme === mode ? 'default' : 'outline'}
                        className={`justify-start h-auto py-3 px-4 font-sans ${
                          theme === mode ? 'ring-2 ring-primary ring-offset-2' : ''
                        }`}
                        onClick={() => setTheme(mode)}
                      >
                        {mode === 'light' && <Sun className="w-4 h-4 mr-2" />}
                        {mode === 'dark' && <Moon className="w-4 h-4 mr-2" />}
                        {mode === 'system' && <Monitor className="w-4 h-4 mr-2" />}
                    <div className="flex flex-col items-start">
                          <span className="font-medium capitalize">
                            {mode === 'light' ? t('settings.appearance.light') : 
                             mode === 'dark' ? t('settings.appearance.dark') : 
                             t('settings.appearance.system')}
                          </span>
                    </div>
                  </Button>
                    ))}
                  </div>
                </div>

                <Separator className="bg-border/20" />

                {/* Language */}
                <div>
                  <label className="text-sm font-medium block mb-4 font-sans">{t('settings.appearance.language')}</label>
                  <div className="grid grid-cols-2 gap-4">
                  <Button
                      variant={i18n.language === 'en' ? 'default' : 'outline'}
                      className={`justify-start h-auto py-3 px-4 font-sans ${
                        i18n.language === 'en' ? 'ring-2 ring-primary ring-offset-2' : ''
                      }`}
                      onClick={() => i18n.changeLanguage('en')}
                    >
                      <Languages className="w-4 h-4 mr-2" />
                    <div className="flex flex-col items-start">
                        <span className="font-medium">English</span>
                    </div>
                  </Button>
                  <Button
                      variant={i18n.language === 'zh' ? 'default' : 'outline'}
                      className={`justify-start h-auto py-3 px-4 font-sans ${
                        i18n.language === 'zh' ? 'ring-2 ring-primary ring-offset-2' : ''
                      }`}
                      onClick={() => i18n.changeLanguage('zh')}
                    >
                      <Globe className="w-4 h-4 mr-2" />
                    <div className="flex flex-col items-start">
                        <span className="font-medium">中文</span>
                    </div>
                  </Button>
                  </div>
                </div>

              </div>
            </div>

            <Separator className="bg-border/40" />

            {/* Embedder Configuration */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <Sparkles className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-sans font-medium text-foreground">
                  {t('settings.embedding.title')}
                </h3>
              </div>
              <div className="p-6 rounded-xl border border-border/40 bg-card/30 grid gap-5">
                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.embedding.baseUrl')}</label>
                  <input
                    type="text"
                    value={config.Embedder.BaseURL}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Embedder: { ...config.Embedder, BaseURL: e.target.value }
                      })
                    )}
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono focus:ring-2 focus:ring-primary/20 transition-all"
                    placeholder="https://api.siliconflow.cn/v1"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.embedding.apiKey')}</label>
                  <input
                    type="password"
                    value={config.Embedder.APIKey}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Embedder: { ...config.Embedder, APIKey: e.target.value }
                      })
                    )}
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono focus:ring-2 focus:ring-primary/20 transition-all"
                    placeholder="sk-..."
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                  <div className="w-full">
                    <label className="text-sm font-medium block mb-2 font-sans">{t('settings.embedding.modelName')}</label>
                    <input
                      type="text"
                      value={config.Embedder.ModelName}
                      onChange={(e) => setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          Embedder: { ...config.Embedder, ModelName: e.target.value }
                        })
                      )}
                      className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                      placeholder="text-embedding-3-small"
                    />
                  </div>

                  <div className="w-full">
                    <label className="text-sm font-medium block mb-2 font-sans">{t('settings.embedding.dimension')}</label>
                    <input
                      type="number"
                      value={config.Embedder.Dim}
                      onChange={(e) => setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          Embedder: { ...config.Embedder, Dim: parseInt(e.target.value) }
                        })
                      )}
                      className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                      placeholder="1536"
                    />
                  </div>
                </div>
              </div>
            </div>

            <Separator className="bg-border/40" />

            {/* Database Configuration */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <DatabaseIcon className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-sans font-medium text-foreground">{t('settings.database.title')}</h3>
              </div>
              <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                <label className="text-sm font-medium block mb-2 font-sans">{t('settings.database.path')}</label>
                <input
                  type="text"
                  value={config.Database.Path}
                  onChange={(e) => setConfig(
                    models.config.AppConfig.createFrom({
                      ...config,
                      Database: { ...config.Database, Path: e.target.value }
                    })
                  )}
                  className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                  placeholder="~/.quicksearch/quicksearch.db"
                />
              </div>
            </div>

            <Separator className="bg-border/40" />

            {/* LLM Configuration */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <Sparkles className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-sans font-medium text-foreground">
                  {t('settings.llm.title')}
                </h3>
              </div>
              <div className="p-6 rounded-xl border border-border/40 bg-card/30 grid gap-5">
                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.llm.baseUrl')}</label>
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
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                    placeholder="https://openrouter.ai/api/v1"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.llm.modelName')}</label>
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
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                    placeholder="anthropic/claude-3-haiku"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.llm.apiKey')}</label>
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
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                    placeholder="sk-..."
                  />
                </div>
              </div>
            </div>

            <Separator className="bg-border/40" />

            {/* Zotero Configuration */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <BookMarked className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-sans font-medium text-foreground">
                  {t('settings.zotero.title')}
                </h3>
              </div>
              <div className="p-6 rounded-xl border border-border/40 bg-card/30 grid gap-5">
                <div className="grid grid-cols-2 gap-5">
                  <div>
                    <label className="text-sm font-medium block mb-2 font-sans">{t('settings.zotero.userId')}</label>
                    <input
                      type="text"
                      value={config.Zotero?.UserID || ''}
                      onChange={(e) => setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          Zotero: { 
                            ...config.Zotero, 
                            UserID: e.target.value,
                            APIKey: config.Zotero?.APIKey || '',
                            LibraryType: config.Zotero?.LibraryType || 'user'
                          }
                        })
                      )}
                      className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                      placeholder="1234567"
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium block mb-2 font-sans">{t('settings.zotero.libraryType')}</label>
                    <Select
                      value={config.Zotero?.LibraryType || 'user'}
                      onValueChange={(value) => setConfig(
                        models.config.AppConfig.createFrom({
                          ...config,
                          Zotero: { 
                            ...config.Zotero, 
                            LibraryType: value,
                            UserID: config.Zotero?.UserID || '',
                            APIKey: config.Zotero?.APIKey || ''
                          }
                        })
                      )}
                    >
                      <SelectTrigger className="w-full h-[42px] px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono">
                        <SelectValue placeholder={t('settings.zotero.user')} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="user">{t('settings.zotero.user')}</SelectItem>
                        <SelectItem value="group">{t('settings.zotero.group')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.zotero.apiKey')}</label>
                  <input
                    type="password"
                    value={config.Zotero?.APIKey || ''}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        Zotero: { 
                          ...config.Zotero, 
                          APIKey: e.target.value,
                          UserID: config.Zotero?.UserID || '',
                          LibraryType: config.Zotero?.LibraryType || 'user'
                        }
                      })
                    )}
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                    placeholder="key..."
                  />
                </div>
              </div>
            </div>

            <Separator className="bg-border/40" />

            {/* Feishu Configuration */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <Share2 className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-sans font-medium text-foreground">
                  {t('settings.feishu.title')}
                </h3>
              </div>
              <div className="p-6 rounded-xl border border-border/40 bg-card/30 grid gap-5">
                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.feishu.appId')}</label>
                  <input
                    type="text"
                    value={config.FeiShu?.AppID || ''}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        FeiShu: { 
                          ...config.FeiShu, 
                          AppID: e.target.value,
                          AppSecret: config.FeiShu?.AppSecret || ''
                        }
                      })
                    )}
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                    placeholder="cli_..."
                  />
          </div>

                <div>
                  <label className="text-sm font-medium block mb-2 font-sans">{t('settings.feishu.appSecret')}</label>
                  <input
                    type="password"
                    value={config.FeiShu?.AppSecret || ''}
                    onChange={(e) => setConfig(
                      models.config.AppConfig.createFrom({
                        ...config,
                        FeiShu: { 
                          ...config.FeiShu, 
                          AppSecret: e.target.value,
                          AppID: config.FeiShu?.AppID || ''
                        }
                      })
                    )}
                    className="w-full px-4 py-2.5 bg-background border border-input rounded-lg text-sm font-mono"
                    placeholder="..."
                  />
                </div>
              </div>
            </div>
            
            <div className="flex items-center gap-3 p-4 bg-secondary/30 rounded-lg border border-border/50">
              <Settings className="w-5 h-5 text-muted-foreground" />
              <p className="text-sm text-muted-foreground font-sans">
                Make sure to save your changes before leaving this page.
              </p>
            </div>

          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default SettingsView;
