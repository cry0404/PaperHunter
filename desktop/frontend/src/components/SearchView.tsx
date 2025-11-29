import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { Separator } from './ui/separator';
import { Checkbox } from './ui/checkbox';
import DownloadLineIcon from 'remixicon-react/DownloadLineIcon';
import PlayLineIcon from 'remixicon-react/PlayLineIcon';
import SettingsLineIcon from 'remixicon-react/SettingsLineIcon';
import DatabaseLineIcon from 'remixicon-react/DatabaseLineIcon';
import CalendarLineIcon from 'remixicon-react/CalendarLineIcon';
import PriceTag3LineIcon from 'remixicon-react/PriceTag3LineIcon';
import HashtagIcon from 'remixicon-react/HashtagIcon';
import TerminalBoxLineIcon from 'remixicon-react/TerminalBoxLineIcon';
import { GetConfig } from '../../wailsjs/go/main/App';
import * as models from '../../wailsjs/go/models';
import { useToast } from './ui/use-toast';
import { useCrawlContext } from '../context/CrawlContext';

const SearchView: React.FC = () => {
  const [config, setConfig] = useState<models.config.AppConfig | null>(null);
  
  // 使用 Context 管理的状态
  const {
    crawlParams,
    setCrawlParams,
    keywordInput,
    setKeywordInput,
    categoryInput,
    setCategoryInput,
    isCrawling,
    setIsCrawling,
    setCurrentTaskId,
    currentTaskId
  } = useCrawlContext();
  
  const [loading, setLoading] = useState(false);
  const { toast } = useToast();

  const loadConfig = async () => {
    setLoading(true);
    try {
      const cfg = await GetConfig();
      setConfig(cfg);
    } catch (error) {
      console.error('Failed to load config:', error);
      toast({
        title: "配置加载失败",
        description: "无法获取配置信息，请重试",
        variant: "destructive",
      });
    } finally {
      setLoading(false);
    }
  };

  const handleCrawl = async () => {
    // 根据平台特性验证输入
    if (crawlParams.platform === 'openreview') {
      if (!crawlParams.venueId.trim()) {
        toast({
          title: "请输入会议名称",
          description: "OpenReview 需要指定会议名称 (venue_id)",
          variant: "destructive",
        });
        return;
      }
    } else if (crawlParams.platform === 'acl') {
      // ACL 不需要额外验证，会自动获取最新论文
    } else if (crawlParams.platform === 'arxiv') {
      if (crawlParams.keywords.length === 0 && crawlParams.categories.length === 0) {
        toast({
          title: "请输入搜索条件",
          description: "arXiv 需要至少填写关键词或类别",
          variant: "destructive",
        });
        return;
      }
    } else if (crawlParams.platform === 'ssrn') {
      if (crawlParams.keywords.length === 0) {
        toast({
          title: "请输入关键词",
          description: "SSRN 需要至少填写一个关键词",
          variant: "destructive",
        });
        return;
      }
    }

    setIsCrawling(true);
    
    try {
      console.log('Starting crawl with params:', crawlParams);
      
      toast({
        title: "爬取开始",
        description: `正在从 ${crawlParams.platform} 爬取论文...`,
      });
      
      // 调用后端爬取API
      const taskId = await startCrawlTask();
      
      // 设置当前任务ID，但不强制跳转
      setCurrentTaskId(taskId);
      
      toast({
        title: "任务已提交",
        description: "爬取任务正在后台运行，您可以点击日志按钮查看进度",
        action: (
          <Button 
            variant="default"
              size="sm" 
              onClick={() => window.location.hash = `#/logs?taskId=${taskId}`}
            >
              查看日志
            </Button>
        ),
      });
      
    } catch (error) {
      console.error('Crawl failed:', error);
      setIsCrawling(false); // 只有失败时才重置状态，成功时由后端状态或用户手动重置
      
      toast({
        title: "爬取失败",
        description: "爬取过程中出现错误，请检查网络连接和配置",
        variant: "destructive",
      });
    }
    // finally 不重置 isCrawling，因为任务是在后台运行的
  };

  // 调用后端爬取API
  const startCrawlTask = async (): Promise<string> => {
    // 构建参数对象
    const params: Record<string, any> = {
      keywords: crawlParams.keywords,
      categories: crawlParams.categories,
      dateFrom: crawlParams.dateFrom,
      dateTo: crawlParams.dateUntil,
      limit: crawlParams.limit,
      update: crawlParams.update,
      useAPI: crawlParams.useAPI,
    };

    // 平台特定参数
    if (crawlParams.platform === 'openreview') {
      params.venueId = crawlParams.venueId;
    } else if (crawlParams.platform === 'acl') {
      params.useRSS = crawlParams.useRSS;
      params.useBibTeX = crawlParams.useBibTeX;
    }

    console.log('Calling backend crawl API with params:', params);
    
    // 调用Wails API
    const { CrawlPapers } = await import('../../wailsjs/go/main/App');
    const taskId = await CrawlPapers(crawlParams.platform, params);
    
    console.log('Crawl task started with ID:', taskId);
    
    return taskId;
  };

  const addKeyword = () => {
    if (keywordInput.trim()) {
      setCrawlParams(prev => ({
        ...prev,
        keywords: [...prev.keywords, keywordInput.trim()]
      }));
      setKeywordInput('');
    }
  };

  const removeKeyword = (index: number) => {
    setCrawlParams(prev => ({
      ...prev,
      keywords: prev.keywords.filter((_, i) => i !== index)
    }));
  };

  const addCategory = () => {
    if (categoryInput.trim()) {
      setCrawlParams(prev => ({
        ...prev,
        categories: [...prev.categories, categoryInput.trim()]
      }));
      setCategoryInput('');
    }
  };

  const removeCategory = (index: number) => {
    setCrawlParams(prev => ({
      ...prev,
      categories: prev.categories.filter((_, i) => i !== index)
    }));
  };

  useEffect(() => {
    loadConfig();
  }, []);

  if (loading || !config) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <DatabaseLineIcon className="w-8 h-8 animate-spin mx-auto mb-4 text-muted-foreground" />
          <p className="text-muted-foreground">Loading configuration...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-6 flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <DownloadLineIcon className="w-5 h-5 text-primary" />
                </div>
                <CardTitle className="text-3xl font-display font-semibold">Crawl Papers</CardTitle>
              </div>
              <CardDescription className="text-sm text-muted-foreground ml-13">
                从学术平台爬取论文标题和摘要
              </CardDescription>
            </div>
            
            <div className="flex items-center gap-2">
              {currentTaskId && (
                <Button
                  onClick={() => window.location.hash = `#/logs?taskId=${currentTaskId}`}
                  size="sm"
                  variant="secondary"
                  className="hover-lift"
                >
                  <TerminalBoxLineIcon className="mr-2 h-4 w-4" />
                  查看任务日志
                  {isCrawling && <div className="w-2 h-2 rounded-full bg-green-500 ml-2 animate-pulse" />}
                </Button>
              )}
              <Button
                onClick={loadConfig}
                disabled={loading}
                size="sm"
                variant="outline"
                className="hover-lift"
              >
                <SettingsLineIcon className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                Reload Config
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto overflow-x-hidden px-8 py-6" style={{ overflowY: 'auto' }}>
          <div className="max-w-5xl mx-auto space-y-6">
            {/* Platform Selection */}
            <Tabs 
              value={crawlParams.platform} 
              onValueChange={(value) => setCrawlParams(prev => ({ ...prev, platform: value as any }))}
              className="w-full"
            >
              <TabsList className="grid w-full grid-cols-4 p-1 bg-secondary/50">
                <TabsTrigger value="arxiv" className="data-[state=active]:bg-card data-[state=active]:shadow-sm">
                  arXiv
                </TabsTrigger>
                <TabsTrigger value="acl" className="data-[state=active]:bg-card data-[state=active]:shadow-sm">
                  ACL Anthology
                </TabsTrigger>
                <TabsTrigger value="openreview" className="data-[state=active]:bg-card data-[state=active]:shadow-sm">
                  OpenReview
                </TabsTrigger>
                <TabsTrigger value="ssrn" className="data-[state=active]:bg-card data-[state=active]:shadow-sm">
                  SSRN
                </TabsTrigger>
              </TabsList>

              {/* Platform-specific Parameters */}
              <TabsContent value={crawlParams.platform} className="space-y-6 mt-6">
                <div className="glass-card p-6 rounded-xl space-y-6">
                  {/* arXiv Parameters */}
                  {crawlParams.platform === 'arxiv' && (
                    <>
                      {/* Keywords */}
                      <div className="space-y-3">
                        <Label className="text-sm font-medium flex items-center gap-2">
                          <PriceTag3LineIcon className="w-4 h-4" />
                          关键词 (Keywords)
                        </Label>
                        <div className="flex gap-2">
                          <Input
                            placeholder="例如: machine learning, deep learning"
                            value={keywordInput}
                            onChange={(e) => setKeywordInput(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && addKeyword()}
                            className="flex-1"
                          />
                          <Button onClick={addKeyword} variant="outline" size="sm">
                            添加
                          </Button>
                        </div>
                        {crawlParams.keywords.length > 0 && (
                          <div className="flex flex-wrap gap-2">
                            {crawlParams.keywords.map((keyword, index) => (
                              <div
                                key={index}
                                className="inline-flex items-center gap-1 px-3 py-1 bg-primary/10 text-primary rounded-full text-sm"
                              >
                                <span>{keyword}</span>
                                <button
                                  onClick={() => removeKeyword(index)}
                                  className="hover:text-primary/70"
                                >
                                  ×
                                </button>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>

                      {/* Categories */}
                      <div className="space-y-3">
                        <Label className="text-sm font-medium flex items-center gap-2">
                          <HashtagIcon className="w-4 h-4" />
                          类别 (Categories)
                        </Label>
                        <div className="flex gap-2">
                          <Input
                            placeholder="例如: cs.AI, cs.LG, cs.CL"
                            value={categoryInput}
                            onChange={(e) => setCategoryInput(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && addCategory()}
                            className="flex-1"
                          />
                          <Button onClick={addCategory} variant="outline" size="sm">
                            添加
                          </Button>
                        </div>
                        {crawlParams.categories.length > 0 && (
                          <div className="flex flex-wrap gap-2">
                            {crawlParams.categories.map((category, index) => (
                              <div
                                key={index}
                                className="inline-flex items-center gap-1 px-3 py-1 bg-info/10 text-info rounded-full text-sm"
                              >
                                <span>{category}</span>
                                <button
                                  onClick={() => removeCategory(index)}
                                  className="hover:text-info/70"
                                >
                                  ×
                                </button>
                              </div>
                            ))}
                          </div>
                        )}
                        <p className="text-xs text-muted-foreground">
                          常用类别: cs.AI (人工智能), cs.LG (机器学习), cs.CL (计算语言学), cs.CV (计算机视觉)
                        </p>
                      </div>
                    </>
                  )}

                  {/* OpenReview Parameters */}
                  {crawlParams.platform === 'openreview' && (
                    <div className="space-y-3">
                      <Label className="text-sm font-medium flex items-center gap-2">
                        <HashtagIcon className="w-4 h-4" />
                        会议名称 (Venue ID)
                      </Label>
                      <Input
                        placeholder="例如: ICLR.cc/2026/Conference/Submission"
                        value={crawlParams.venueId}
                        onChange={(e) => setCrawlParams(prev => ({ ...prev, venueId: e.target.value }))}
                        className="flex-1"
                      />
                      <p className="text-xs text-muted-foreground">
                        OpenReview 只支持按会议名称检索，不支持关键词搜索
                      </p>
                    </div>
                  )}

                  {/* ACL Parameters */}
                  {crawlParams.platform === 'acl' && (
                    <div className="space-y-3">
                      <Label className="text-sm font-medium">检索模式</Label>
                      <div className="space-y-2">
                        <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors">
                          <Checkbox
                            checked={crawlParams.useRSS}
                            onCheckedChange={(checked: boolean) => 
                              setCrawlParams(prev => ({ 
                                ...prev, 
                                useRSS: checked,
                                useBibTeX: checked ? false : prev.useBibTeX
                              }))
                            }
                          />
                          <span>RSS 模式 (获取最新 1000 篇论文)</span>
                        </label>
                        <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors">
                          <Checkbox
                            checked={crawlParams.useBibTeX}
                            onCheckedChange={(checked: boolean) => 
                              setCrawlParams(prev => ({ 
                                ...prev, 
                                useBibTeX: checked,
                                useRSS: checked ? false : prev.useRSS
                              }))
                            }
                          />
                          <span>BibTeX 模式 (获取全量论文数据)</span>
                        </label>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        ACL Anthology 不支持关键词搜索，只能获取最新或全量论文
                      </p>
                    </div>
                  )}

                  {/* SSRN Parameters */}
                  {crawlParams.platform === 'ssrn' && (
                    <>
                      {/* Keywords */}
                      <div className="space-y-3">
                        <Label className="text-sm font-medium flex items-center gap-2">
                          <PriceTag3LineIcon className="w-4 h-4" />
                          关键词 (Keywords)
                        </Label>
                        <div className="flex gap-2">
                          <Input
                            placeholder="例如: machine learning, finance, economics"
                            value={keywordInput}
                            onChange={(e) => setKeywordInput(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && addKeyword()}
                            className="flex-1"
                          />
                          <Button onClick={addKeyword} variant="outline" size="sm">
                            添加
                          </Button>
                        </div>
                        {crawlParams.keywords.length > 0 && (
                          <div className="flex flex-wrap gap-2">
                            {crawlParams.keywords.map((keyword, index) => (
                              <div
                                key={index}
                                className="inline-flex items-center gap-1 px-3 py-1 bg-primary/10 text-primary rounded-full text-sm"
                              >
                                <span>{keyword}</span>
                                <button
                                  onClick={() => removeKeyword(index)}
                                  className="hover:text-primary/70"
                                >
                                  ×
                                </button>
                              </div>
                            ))}
                          </div>
                        )}
                        <p className="text-xs text-muted-foreground">
                          SSRN 支持关键词搜索，不支持类别和日期筛选
                        </p>
                      </div>
                    </>
                  )}

                  {/* Date Range - Only for arXiv */}
                  {crawlParams.platform === 'arxiv' && (
                    <>
                      <Separator className="bg-border/50" />
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label className="text-sm font-medium flex items-center gap-2">
                            <CalendarLineIcon className="w-4 h-4" />
                            开始日期 (From)
                          </Label>
                          <Input
                            type="date"
                            value={crawlParams.dateFrom}
                            onChange={(e) => setCrawlParams(prev => ({ ...prev, dateFrom: e.target.value }))}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium flex items-center gap-2">
                            <CalendarLineIcon className="w-4 h-4" />
                            结束日期 (Until)
                          </Label>
                          <Input
                            type="date"
                            value={crawlParams.dateUntil}
                            onChange={(e) => setCrawlParams(prev => ({ ...prev, dateUntil: e.target.value }))}
                          />
                        </div>
                      </div>
                    </>
                  )}

                  <Separator className="bg-border/50" />

                  {/* Limit */}
                  <div className="space-y-2">
                    <Label className="text-sm font-medium">
                      数量限制 (Limit)
                    </Label>
                    <Input
                      type="number"
                      value={crawlParams.limit}
                      onChange={(e) => setCrawlParams(prev => ({ ...prev, limit: parseInt(e.target.value) || 0 }))}
                      placeholder="0 表示无限制"
                    />
                    <p className="text-xs text-muted-foreground">
                      设置为 0 表示不限制爬取数量
                    </p>
                  </div>

                  <Separator className="bg-border/50" />

                  {/* Options - Platform specific */}
                  <div className="space-y-3">
                    <Label className="text-sm font-medium">选项</Label>
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors">
                        <Checkbox
                          checked={crawlParams.update}
                          onCheckedChange={(checked: boolean) => 
                            setCrawlParams(prev => ({ ...prev, update: checked }))
                          }
                        />
                        <span>增量更新模式 (Update Mode)</span>
                      </label>
                      {/* API选项只对arXiv显示 */}
                      {crawlParams.platform === 'arxiv' && (
                        <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors">
                          <Checkbox
                            checked={crawlParams.useAPI}
                            onCheckedChange={(checked: boolean) => 
                              setCrawlParams(prev => ({ ...prev, useAPI: checked }))
                            }
                          />
                          <span>使用官方 API (Use Official API)</span>
                        </label>
                      )}
                    </div>
                  </div>
                </div>
              </TabsContent>
            </Tabs>

            <Separator className="bg-border/50" />

            {/* Action Bar */}
            <div className="glass-card p-4 rounded-xl">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-muted flex items-center justify-center">
                    <SettingsLineIcon className="w-4 h-4 text-muted-foreground" />
                  </div>
                  <div className="text-sm text-muted-foreground">
                    配置参数来自 Settings 页面
                  </div>
                </div>
                <Button
                  onClick={handleCrawl}
                  disabled={isCrawling || (
                    crawlParams.platform === 'arxiv' && 
                    crawlParams.keywords.length === 0 && 
                    crawlParams.categories.length === 0
                  ) || (
                    crawlParams.platform === 'openreview' && 
                    !crawlParams.venueId.trim()
                  ) || (
                    crawlParams.platform === 'ssrn' && 
                    crawlParams.keywords.length === 0
                  )}
                  size="lg"
                  className="bg-primary hover:bg-primary/90 transition-all duration-300 h-11 px-8"
                >
                  {isCrawling ? (
                    <>
                      <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                      Task Running...
                    </>
                  ) : (
                    <>
                      <PlayLineIcon className="mr-2 h-5 w-5" />
                      Start Crawl
                    </>
                  )}
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

    </div>
  );
};

export default SearchView;