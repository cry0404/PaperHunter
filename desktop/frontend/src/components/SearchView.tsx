import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { Separator } from './ui/separator';
import { Checkbox } from './ui/checkbox';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from './ui/dropdown-menu';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from './ui/alert-dialog';

import {
  Play,
  Settings,
  Database,
  Calendar,
  Tags,
  Hash,
  Terminal,
  Download,
  ExternalLink,
  RefreshCw,
  Loader2,
  FileDown
} from 'lucide-react';

import { GetConfig } from '../../wailsjs/go/main/App';
import * as models from '../../wailsjs/go/models';
import { useToast } from './ui/use-toast';
import { useCrawlContext } from '../context/CrawlContext';
import { EventsOn, EventsOff, BrowserOpenURL } from '../../wailsjs/runtime/runtime';

interface PaperItem {
  ID: number;
  Source: string;
  SourceID: string;
  Title: string;
  Authors: string[];
  Abstract: string;
  URL: string;
  FirstAnnouncedAt: string;
}

interface CrawlHistoryEntry {
  task_id: string;
  platform: string;
  total: number;
  start_time: string;
  end_time: string;
}

const SearchView: React.FC = () => {
  const { t } = useTranslation();
  const [config, setConfig] = useState<models.config.AppConfig | null>(null);
  const completionNotifiedRef = useRef<string | null>(null);
  
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
  const [exportFormat, setExportFormat] = useState<'csv' | 'json' | 'feishu' | 'zotero'>('csv');
  const [exportOutput, setExportOutput] = useState('');
  const [exportFeishuName, setExportFeishuName] = useState('');
  const [exportCollection, setExportCollection] = useState('');
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [exportDialogFormat, setExportDialogFormat] = useState<'csv' | 'json' | 'feishu' | 'zotero'>('csv');
  const [exportDialogPath, setExportDialogPath] = useState('');
  const [exportDialogFeishu, setExportDialogFeishu] = useState('');
  const [exportDialogCollection, setExportDialogCollection] = useState('');
  const [taskPapers, setTaskPapers] = useState<PaperItem[]>([]);
  const [taskPapersLoading, setTaskPapersLoading] = useState(false);
  const [taskPapersError, setTaskPapersError] = useState<string | null>(null);
  const [taskStatus, setTaskStatus] = useState<'pending' | 'running' | 'completed' | 'failed' | null>(null);
  const [history, setHistory] = useState<CrawlHistoryEntry[]>([]);
  const { toast } = useToast();

  const loadConfig = async () => {
    setLoading(true);
    try {
      const cfg = await GetConfig();
      setConfig(cfg);
    } catch (error) {
      console.error('Failed to load config:', error);
      toast({
        title: t('common.error'),
        description: "Could not retrieve configuration info, please retry",
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
          title: t('common.error'),
          description: "Venue ID Required",
          variant: "destructive",
        });
        return;
      }
    } else if (crawlParams.platform === 'acl') {
      // ACL 不需要额外验证，会自动获取最新论文
    } else if (crawlParams.platform === 'arxiv') {
      if (crawlParams.keywords.length === 0 && crawlParams.categories.length === 0) {
        toast({
          title: t('common.error'),
          description: "arXiv requires at least one keyword or category",
          variant: "destructive",
        });
        return;
      }
    } else if (crawlParams.platform === 'ssrn') {
      if (crawlParams.keywords.length === 0) {
        toast({
          title: t('common.error'),
          description: "SSRN requires at least one keyword",
          variant: "destructive",
        });
        return;
      }
    }

    setIsCrawling(true);
    setTaskStatus('running');
    setTaskPapers([]);
    setTaskPapersError(null);
    
    try {
      console.log('Starting crawl with params:', crawlParams);
      
      toast({
        title: t('common.success'),
        description: `Crawling papers from ${crawlParams.platform}...`,
      });
      
      // 调用后端爬取API
      const taskId = await startCrawlTask();
      
      // 设置当前任务ID，但不强制跳转
      setCurrentTaskId(taskId);
      
      
    } catch (error) {
      console.error('Crawl failed:', error);
      setIsCrawling(false); // 只有失败时才重置状态，成功时由后端状态或用户手动重置
      setTaskStatus('failed');
      
      toast({
        title: t('common.error'),
        description: "An error occurred during crawling, check logs for details",
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

  // 一键导出当前任务结果（默认 csv）
  const handleExportCurrentTask = async (format?: 'csv' | 'json' | 'feishu' | 'zotero', override?: {
    output?: string;
    feishuName?: string;
    collection?: string;
  }) => {
    const fmt = format || exportFormat;
    let out = override?.output ?? exportOutput;
    const feishuName = override?.feishuName ?? exportFeishuName;
    const collection = override?.collection ?? exportCollection;

    if (!currentTaskId) {
      toast({
        title: t('common.error'),
        description: "Please run a crawl task first",
        variant: "destructive",
      });
      return;
    }

    if ((fmt === 'csv' || fmt === 'json') && !out.trim()) {
      const now = new Date();
      const stamp = `${now.getFullYear()}${String(now.getMonth()+1).padStart(2,'0')}${String(now.getDate()).padStart(2,'0')}_${String(now.getHours()).padStart(2,'0')}${String(now.getMinutes()).padStart(2,'0')}${String(now.getSeconds()).padStart(2,'0')}`;
      out = `${currentTaskId || 'crawl'}_${stamp}.${fmt}`;
    }
    if (fmt === 'feishu' && !feishuName.trim()) {
      toast({
        title: t('common.error'),
        description: "Please provide a name for the Feishu dataset",
        variant: "destructive",
      });
      return;
    }

    try {
      const { ExportCrawlTask }: any = await import('../../wailsjs/go/main/App');
      const result = await ExportCrawlTask(currentTaskId, fmt, out, feishuName, collection);

      if ((fmt === 'csv' || fmt === 'json') && result) {
        toast({
          title: t('common.success'),
          description: (
            <div className="break-all">
              Saved to: <a className="underline cursor-pointer" onClick={() => BrowserOpenURL(`file://${result}`)}>{result}</a>
            </div>
          ),
        });
      } else if (fmt === 'feishu' && result) {
        toast({
          title: t('common.success'),
          description: (
            <div className="break-all">
              Uploaded to Feishu: <a className="underline cursor-pointer" onClick={() => BrowserOpenURL(result)}>{result}</a>
            </div>
          ),
        });
      } else {
        toast({
          title: t('common.success'),
          description: "Operation successful",
        });
      }
    } catch (error) {
      console.error('Export crawl task failed:', error);
      toast({
        title: t('common.error'),
        description: "Error during export, check configuration",
        variant: "destructive",
      });
    }
  };

  // 拉取本次任务的论文列表，供页面内展示
  const loadTaskPapers = async (taskId: string) => {
    setTaskPapersLoading(true);
    setTaskPapersError(null);
    try {
      const { GetCrawlTaskPapers }: any = await import('../../wailsjs/go/main/App');
      const data = await GetCrawlTaskPapers(taskId);
      const list = JSON.parse(data || '[]') as PaperItem[];
      setTaskPapers(list);
    } catch (error) {
      console.error('Load task papers failed:', error);
      setTaskPapersError('Failed to load papers');
    } finally {
      setTaskPapersLoading(false);
    }
  };

  const loadHistory = async () => {
    try {
      const { GetCrawlHistory }: any = await import('../../wailsjs/go/main/App');
      const data = await GetCrawlHistory(10);
      const list = JSON.parse(data || '[]') as CrawlHistoryEntry[];
      setHistory(list);
    } catch (error) {
      console.error('Load history failed:', error);
    }
  };

  const formatTime = (t?: string) => {
    if (!t) return '';
    const d = new Date(t);
    return d.toLocaleString();
  };

  const handleClearHistory = async () => {
    try {
      const { ClearCrawlHistory }: any = await import('../../wailsjs/go/main/App');
      await ClearCrawlHistory();
      setHistory([]);
      toast({ title: t('common.success'), description: "History cleared" });
    } catch (error) {
      console.error('Clear history failed:', error);
      toast({ title: t('common.error'), description: "Check logs for details", variant: "destructive" });
    }
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
    loadHistory();
  }, []);

  // 监听后端流式日志，在任务完成/失败时及时复位状态
  useEffect(() => {
    if (!currentTaskId) return;

    EventsOn("crawl-log", (logEntry: any) => {
      if (logEntry?.task_id !== currentTaskId) return;

      if (logEntry.level === 'success' || logEntry.level === 'error') {
        setIsCrawling(false);
        setTaskStatus(logEntry.level === 'success' ? 'completed' : 'failed');

        if (completionNotifiedRef.current !== currentTaskId) {
          completionNotifiedRef.current = currentTaskId;
          // 拉取本次结果列表
          if (logEntry.level === 'success') {
            loadTaskPapers(currentTaskId);
              loadHistory();
          }
          if (logEntry.level === 'error') {
            toast({
              title: t('common.error'),
              description: "Error occurred, check logs",
              variant: "destructive",
            });
          }
        }
      }
    });

    return () => {
      EventsOff("crawl-log");
    };
  }, [currentTaskId, crawlParams.platform, setIsCrawling, toast]);

  // 兜底轮询任务状态
  useEffect(() => {
    if (!isCrawling || !currentTaskId) return;

    const interval = setInterval(async () => {
      try {
        const { GetCrawlTask } = await import('../../wailsjs/go/main/App');
        const taskJson = await GetCrawlTask(currentTaskId);
        const task = JSON.parse(taskJson || '{}');
        const status = task?.status;

        if (status && status !== 'running' && status !== 'pending') {
          setIsCrawling(false);
          setTaskStatus(status);

          if (completionNotifiedRef.current !== currentTaskId) {
            completionNotifiedRef.current = currentTaskId;
            if (status === 'completed') {
              loadTaskPapers(currentTaskId);
              loadHistory();
            }
            if (status !== 'completed') {
              toast({
                title: t('common.error'),
                description: "Error occurred, check logs",
                variant: "destructive",
              });
            }
          }
        }
      } catch (error) {
        console.error('Failed to poll task status:', error);
      }
    }, 4000);

    return () => clearInterval(interval);
  }, [isCrawling, currentTaskId, crawlParams.platform, setIsCrawling, toast]);

  if (loading || !config) {
    return (
      <div className="flex items-center justify-center h-full bg-background">
        <div className="text-center">
          <Database className="w-8 h-8 animate-spin mx-auto mb-4 text-muted-foreground" />
          <p className="text-muted-foreground font-sans">{t('common.loading')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-6 flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-3">
              
                <CardTitle className="text-3xl font-sans font-medium tracking-tight">{t('search.title')}</CardTitle>
              </div>
              <CardDescription className="text-sm text-muted-foreground ml-13 font-serif">
                {t('search.subtitle')}
              </CardDescription>
            </div>
            
            <div className="flex items-center gap-2">
              {currentTaskId && (
                <Button
                  onClick={() => window.location.hash = `#/logs?taskId=${currentTaskId}`}
                  size="sm"
                  variant="secondary"
                  className="hover-lift flex items-center gap-2 font-sans"
                >
                  <Terminal className="h-4 w-4" />
                  <span>{t('search.viewLogs')}</span>
                  {isCrawling && <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />}
                </Button>
              )}
              <Button
                onClick={loadConfig}
                disabled={loading}
                size="sm"
                variant="outline"
                className="hover-lift font-sans"
              >
                <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                {t('search.reloadConfig')}
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto overflow-x-hidden px-8 py-6" style={{ overflowY: 'auto' }}>
          <div className="max-w-5xl mx-auto space-y-6">
            {/* 当前任务结果展示，仅任务完成后显示 */}
            {currentTaskId && taskStatus === 'completed' && (
              <div className="p-4 rounded-xl space-y-3 bg-card/30 border border-border/40">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-sm text-muted-foreground font-sans">{t('search.currentTask')}</div>
                    <div className="text-lg font-semibold font-mono text-foreground">{currentTaskId}</div>
                  </div>
                  <div className="flex items-center gap-2">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button size="sm" disabled={taskPapers.length === 0} className="font-sans">
                          <Download className="w-4 h-4 mr-2" />
                          {t('search.export')}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-44 font-sans">
                        <DropdownMenuLabel>Choose Format</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        {(['csv','json','feishu','zotero'] as const).map(fmt => (
                          <DropdownMenuItem
                            key={fmt}
                            className="cursor-pointer"
                            onClick={() => {
                              setExportDialogFormat(fmt);
                              if ((fmt === 'csv' || fmt === 'json') && currentTaskId) {
                                const now = new Date();
                                const stamp = `${now.getFullYear()}${String(now.getMonth()+1).padStart(2,'0')}${String(now.getDate()).padStart(2,'0')}_${String(now.getHours()).padStart(2,'0')}${String(now.getMinutes()).padStart(2,'0')}${String(now.getSeconds()).padStart(2,'0')}`;
                                setExportDialogPath(`${currentTaskId}_${stamp}.${fmt}`);
                              } else {
                                setExportDialogPath('');
                              }
                              setExportDialogFeishu('');
                              setExportDialogCollection('');
                              setExportDialogOpen(true);
                            }}
                          >
                            {fmt.toUpperCase()}
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
                {taskPapersLoading && <div className="text-sm text-muted-foreground font-sans">{t('common.loading')}</div>}
                {taskPapersError && <div className="text-sm text-destructive font-sans">{taskPapersError}</div>}
                {!taskPapersLoading && taskPapers.length === 0 && (
                  <div className="text-sm text-muted-foreground font-sans">{t('search.noPapers')}</div>
                )}
                {!taskPapersLoading && taskPapers.length > 0 && (
                  <div className="space-y-3">
                    {taskPapers.map((paper) => (
                      <div key={`${paper.Source}-${paper.SourceID}`} className="border border-border/60 rounded-lg p-3 hover:bg-card/60 transition bg-background">
                        <div className="flex items-start justify-between gap-3">
                          <div className="space-y-1">
                            <div className="text-base font-semibold leading-snug font-sans text-foreground">{paper.Title}</div>
                            <div className="text-xs text-muted-foreground font-sans">
                              {paper.Authors?.join(', ')}
                            </div>
                            <div className="text-xs text-muted-foreground font-mono">
                              {paper.Source} · {paper.SourceID}
                            </div>
                          </div>
                          {paper.URL && (
                            <Button
                              size="sm"
                              variant="ghost"
                              className="font-sans"
                              onClick={() => BrowserOpenURL(paper.URL)}
                            >
                              <ExternalLink className="w-4 h-4 mr-1" />
                              {t('search.open')}
                            </Button>
                          )}
                        </div>
                        {paper.Abstract && (
                          <div className="text-sm text-muted-foreground mt-2 line-clamp-3 font-serif leading-relaxed">
                            {paper.Abstract}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* 历史记录 */}
            {history.length > 0 && (
              <div className="p-4 rounded-xl space-y-3 bg-card/30 border border-border/40">
                <div className="flex items-center justify-between">
                  <div className="text-lg font-semibold font-sans">{t('search.history')}</div>
                  <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={loadHistory} className="font-sans">
                      <RefreshCw className="w-4 h-4 mr-2" />
                      {t('search.refresh')}
                    </Button>
                    <Button variant="outline" size="sm" onClick={handleClearHistory} className="font-sans">
                      {t('search.clear')}
                    </Button>
                  </div>
                </div>
                <div className="space-y-2 max-h-72 overflow-auto pr-1">
                  {history.map((h) => (
                    <div key={h.task_id} className="border border-border/60 rounded-lg p-3 flex items-center justify-between hover:bg-card/60 transition bg-background">
                      <div>
                        <div className="text-sm font-semibold font-sans">{h.platform} · {h.total} papers</div>
                        <div className="text-xs text-muted-foreground font-mono">Start: {formatTime(h.start_time)}</div>
                        <div className="text-xs text-muted-foreground font-mono">End: {formatTime(h.end_time)}</div>
                      </div>
                      <Button
                        size="sm"
                        variant="secondary"
                        className="font-sans"
                        onClick={() => window.location.hash = `#/library?taskId=${h.task_id}`}
                      >
                        View Library
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Platform Selection */}
            <Tabs 
              value={crawlParams.platform} 
              onValueChange={(value) => setCrawlParams(prev => ({ ...prev, platform: value as any }))}
              className="w-full"
            >
              <TabsList className="grid w-full grid-cols-4 p-1 bg-secondary/30 font-sans">
                <TabsTrigger value="arxiv" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                  arXiv
                </TabsTrigger>
                <TabsTrigger value="acl" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                  ACL Anthology
                </TabsTrigger>
                <TabsTrigger value="openreview" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                  OpenReview
                </TabsTrigger>
                <TabsTrigger value="ssrn" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                  SSRN
                </TabsTrigger>
              </TabsList>

              {/* Platform-specific Parameters */}
              <TabsContent value={crawlParams.platform} className="space-y-6 mt-6">
                <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-6">
                  {/* arXiv Parameters */}
                  {crawlParams.platform === 'arxiv' && (
                    <>
                      {/* Keywords */}
                      <div className="space-y-3">
                        <Label className="text-sm font-medium flex items-center gap-2 font-sans">
                          <Tags className="w-4 h-4" />
                          {t('search.keywords')}
                        </Label>
                        <div className="flex gap-2">
                          <Input
                            placeholder="e.g.: machine learning, deep learning"
                            value={keywordInput}
                            onChange={(e) => setKeywordInput(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && addKeyword()}
                            className="flex-1 font-sans"
                          />
                          <Button onClick={addKeyword} variant="outline" size="sm" className="font-sans">
                            {t('search.add')}
                          </Button>
                        </div>
                        {crawlParams.keywords.length > 0 && (
                          <div className="flex flex-wrap gap-2">
                            {crawlParams.keywords.map((keyword, index) => (
                              <div
                                key={index}
                                className="inline-flex items-center gap-1 px-3 py-1 bg-primary/10 text-primary rounded-full text-sm font-sans"
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
                        <Label className="text-sm font-medium flex items-center gap-2 font-sans">
                          <Hash className="w-4 h-4" />
                          {t('search.categories')}
                        </Label>
                        <div className="flex gap-2">
                          <Input
                            placeholder="e.g.: cs.AI, cs.LG, cs.CL"
                            value={categoryInput}
                            onChange={(e) => setCategoryInput(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && addCategory()}
                            className="flex-1 font-sans"
                          />
                          <Button onClick={addCategory} variant="outline" size="sm" className="font-sans">
                            {t('search.add')}
                          </Button>
                        </div>
                        {crawlParams.categories.length > 0 && (
                          <div className="flex flex-wrap gap-2">
                            {crawlParams.categories.map((category, index) => (
                              <div
                                key={index}
                                className="inline-flex items-center gap-1 px-3 py-1 bg-info/10 text-info rounded-full text-sm font-mono"
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
                        <p className="text-xs text-muted-foreground font-sans">
                          Common: cs.AI, cs.LG, cs.CL, cs.CV
                        </p>
                      </div>
                    </>
                  )}

                  {/* OpenReview Parameters */}
                  {crawlParams.platform === 'openreview' && (
                    <div className="space-y-3">
                      <Label className="text-sm font-medium flex items-center gap-2 font-sans">
                        <Hash className="w-4 h-4" />
                        {t('search.venueId')}
                      </Label>
                      <Input
                        placeholder="e.g.: ICLR.cc/2026/Conference/Submission"
                        value={crawlParams.venueId}
                        onChange={(e) => setCrawlParams(prev => ({ ...prev, venueId: e.target.value }))}
                        className="flex-1 font-mono text-sm"
                      />
                      <p className="text-xs text-muted-foreground font-sans">
                        OpenReview only supports crawling by Venue ID.
                      </p>
                    </div>
                  )}

                  {/* ACL Parameters */}
                  {crawlParams.platform === 'acl' && (
                    <div className="space-y-3">
                      <Label className="text-sm font-medium font-sans">{t('search.retrievalMode')}</Label>
                      <div className="space-y-2">
                        <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors font-sans">
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
                          <span>{t('search.rssMode')}</span>
                        </label>
                        <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors font-sans">
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
                          <span>{t('search.bibtexMode')}</span>
                        </label>
                      </div>
                      <p className="text-xs text-muted-foreground font-sans">
                        ACL Anthology retrieval is based on latest RSS feed or full BibTeX dump.
                      </p>
                    </div>
                  )}

                  {/* SSRN Parameters */}
                  {crawlParams.platform === 'ssrn' && (
                    <>
                      {/* Keywords */}
                      <div className="space-y-3">
                        <Label className="text-sm font-medium flex items-center gap-2 font-sans">
                          <Tags className="w-4 h-4" />
                          {t('search.keywords')}
                        </Label>
                        <div className="flex gap-2">
                          <Input
                            placeholder="e.g.: machine learning, finance"
                            value={keywordInput}
                            onChange={(e) => setKeywordInput(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && addKeyword()}
                            className="flex-1 font-sans"
                          />
                          <Button onClick={addKeyword} variant="outline" size="sm" className="font-sans">
                            {t('search.add')}
                          </Button>
                        </div>
                        {crawlParams.keywords.length > 0 && (
                          <div className="flex flex-wrap gap-2">
                            {crawlParams.keywords.map((keyword, index) => (
                              <div
                                key={index}
                                className="inline-flex items-center gap-1 px-3 py-1 bg-primary/10 text-primary rounded-full text-sm font-sans"
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
                        <p className="text-xs text-muted-foreground font-sans">
                          SSRN supports keywords only.
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
                          <Label className="text-sm font-medium flex items-center gap-2 font-sans">
                            <Calendar className="w-4 h-4" />
                            {t('search.dateFrom')}
                          </Label>
                          <Input
                            type="date"
                            value={crawlParams.dateFrom}
                            onChange={(e) => setCrawlParams(prev => ({ ...prev, dateFrom: e.target.value }))}
                            className="font-sans"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-sm font-medium flex items-center gap-2 font-sans">
                            <Calendar className="w-4 h-4" />
                            {t('search.dateUntil')}
                          </Label>
                          <Input
                            type="date"
                            value={crawlParams.dateUntil}
                            onChange={(e) => setCrawlParams(prev => ({ ...prev, dateUntil: e.target.value }))}
                            className="font-sans"
                          />
                        </div>
                      </div>
                    </>
                  )}

                  <Separator className="bg-border/50" />

                  {/* Limit */}
                  <div className="space-y-2">
                    <Label className="text-sm font-medium font-sans">
                      {t('search.limit')}
                    </Label>
                    <Input
                      type="number"
                      value={crawlParams.limit}
                      onChange={(e) => setCrawlParams(prev => ({ ...prev, limit: parseInt(e.target.value) || 0 }))}
                      placeholder="0 for unlimited"
                      className="font-sans"
                    />
                    <p className="text-xs text-muted-foreground font-sans">
                      0 means no limit.
                    </p>
                  </div>

                  <Separator className="bg-border/50" />

                  {/* Options - Platform specific */}
                  <div className="space-y-3">
                    <Label className="text-sm font-medium font-sans">{t('search.options')}</Label>
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors font-sans">
                        <Checkbox
                          checked={crawlParams.update}
                          onCheckedChange={(checked: boolean) => 
                            setCrawlParams(prev => ({ ...prev, update: checked }))
                          }
                        />
                        <span>{t('search.updateMode')}</span>
                      </label>
                      {/* API选项只对arXiv显示 */}
                      {crawlParams.platform === 'arxiv' && (
                        <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors font-sans">
                          <Checkbox
                            checked={crawlParams.useAPI}
                            onCheckedChange={(checked: boolean) => 
                              setCrawlParams(prev => ({ ...prev, useAPI: checked }))
                            }
                          />
                          <span>{t('search.useApi')}</span>
                        </label>
                      )}
                    </div>
                  </div>
                </div>
              </TabsContent>
            </Tabs>

            <Separator className="bg-border/50" />

            {/* Action Bar */}
            <div className="p-4 rounded-xl bg-card/30 border border-border/40">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-muted flex items-center justify-center">
                    <Settings className="w-4 h-4 text-muted-foreground" />
                  </div>
                  <div className="text-sm text-muted-foreground font-sans">
                    Parameters from Settings page are applied automatically.
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
                  className="bg-anthropic-dark text-anthropic-light hover:bg-anthropic-dark/90 transition-all duration-300 h-11 px-8 font-sans"
                >
                  {isCrawling ? (
                    <>
                      <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                      {t('search.running')}
                    </>
                  ) : (
                    <>
                      <Play className="mr-2 h-5 w-5" />
                      {t('search.startCrawl')}
                    </>
                  )}
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 导出配置弹窗 */}
      <AlertDialog open={exportDialogOpen} onOpenChange={setExportDialogOpen}>
        <AlertDialogContent className="sm:max-w-lg font-sans">
          <AlertDialogHeader>
            <AlertDialogTitle>{t('export.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('export.description')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">{t('export.format')}</Label>
              <div className="grid grid-cols-4 gap-2">
                {(['csv','json','feishu','zotero'] as const).map(fmt => (
                  <Button
                    key={fmt}
                    variant={exportDialogFormat === fmt ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setExportDialogFormat(fmt)}
                  >
                    {fmt.toUpperCase()}
                  </Button>
                ))}
              </div>
            </div>

            {(exportDialogFormat === 'csv' || exportDialogFormat === 'json') && (
              <div className="space-y-2">
                <Label className="text-sm font-medium">{t('export.outputPath')}</Label>
                <Input
                  value={exportDialogPath}
                  onChange={(e) => setExportDialogPath(e.target.value)}
                  placeholder="e.g.: out/papers.csv"
                />
                <p className="text-xs text-muted-foreground">Auto-generated if empty</p>
              </div>
            )}

            {exportDialogFormat === 'feishu' && (
              <div className="space-y-2">
                <Label className="text-sm font-medium">{t('export.feishuName')}</Label>
                <Input
                  value={exportDialogFeishu}
                  onChange={(e) => setExportDialogFeishu(e.target.value)}
                  placeholder="e.g.: Papers Dataset"
                />
              </div>
            )}

            {exportDialogFormat === 'zotero' && (
              <div className="space-y-2">
                <Label className="text-sm font-medium">{t('export.collectionKey')}</Label>
                <Input
                  value={exportDialogCollection}
                  onChange={(e) => setExportDialogCollection(e.target.value)}
                  placeholder="e.g. ABC123XY"
                />
              </div>
            )}
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                await handleExportCurrentTask(exportDialogFormat, {
                  output: exportDialogPath,
                  feishuName: exportDialogFeishu,
                  collection: exportDialogCollection,
                });
                setExportDialogOpen(false);
              }}
            >
              {t('common.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default SearchView;
