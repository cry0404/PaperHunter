import React, { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Badge } from './ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Checkbox } from './ui/checkbox';
import StarLineIcon from 'remixicon-react/StarLineIcon';
import SearchLineIcon from 'remixicon-react/SearchLineIcon';
import DownloadLineIcon from 'remixicon-react/DownloadLineIcon';
import RefreshLineIcon from 'remixicon-react/RefreshLineIcon';
import ExternalLinkLineIcon from 'remixicon-react/ExternalLinkLineIcon';
import BookOpenLineIcon from 'remixicon-react/BookOpenLineIcon';
import CalendarLineIcon from 'remixicon-react/CalendarLineIcon';
import FileCopyLineIcon from 'remixicon-react/FileCopyLineIcon';
import ArrowLeftLineIcon from 'remixicon-react/ArrowLeftLineIcon';
import TerminalBoxLineIcon from 'remixicon-react/TerminalBoxLineIcon';
import ArrowDownLineIcon from 'remixicon-react/ArrowDownLineIcon';
import { BrowserOpenURL, EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { useToast } from './ui/use-toast';
import { GetDailyRecommendations, ExportSelectionByPapers } from '../../wailsjs/go/main/App';
import * as models from '../../wailsjs/go/models';
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

interface Paper {
  id?: string;
  sourceId: string;
  title: string;
  authors: string[];
  abstract: string;
  url: string;
  published?: string;
  firstSubmittedAt?: string;
  source: string;
  similarity: number;
}

interface RecommendationGroup {
  zoteroPaper: {
    title: string;
    authors: string[];
    abstract: string;
  };
  papers: Paper[];
}

interface RecommendResult {
  crawledToday: boolean;
  crawlCount: number;
  zoteroPaperCount: number;
  recommendations: RecommendationGroup[];
  message: string;
  agentLogs?: AgentLogEntry[];
}

interface AgentLogEntry {
  type: 'user' | 'assistant' | 'tool_call' | 'tool_result' | 'error';
  content: string;
  timestamp: string;
}

const RecommendView: React.FC = () => {
  const [interestQuery, setInterestQuery] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [recommendations, setRecommendations] = useState<RecommendationGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedPapers, setSelectedPapers] = useState<Set<string>>(new Set());
  const [exportOpen, setExportOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<'csv' | 'json' | 'zotero' | 'feishu'>('csv');
  const [exportOutput, setExportOutput] = useState('');
  const [exportCollection, setExportCollection] = useState('');
  const [exportFeishuName, setExportFeishuName] = useState('');
  const [agentLogs, setAgentLogs] = useState<AgentLogEntry[]>([]);
  const [showLogs, setShowLogs] = useState(false);
  const [showRecommendations, setShowRecommendations] = useState(false); // 控制是否显示全屏推荐
  const [mergedPapers, setMergedPapers] = useState<Paper[]>([]); // 合并后的论文列表
  const [autoScroll, setAutoScroll] = useState(true);
  const logScrollRef = useRef<HTMLDivElement>(null);
  const { toast } = useToast();

  // 监听后端流式日志
  useEffect(() => {
    const cancelLogListener = EventsOn("agent-log", (logEntry: AgentLogEntry) => {
      setAgentLogs(prev => [...prev, logEntry]);
      // 当收到第一条日志时，自动展开日志面板
      if (!showLogs && loading) {
        setShowLogs(true);
      }
    });

    return () => {
      EventsOff("agent-log");
    };
  }, [loading, showLogs]);

  // 自动滚动日志到底部
  useEffect(() => {
    if (showLogs && autoScroll && logScrollRef.current) {
      // 使用 requestAnimationFrame 确保 DOM 更新完成后再滚动
      requestAnimationFrame(() => {
        if (logScrollRef.current) {
          logScrollRef.current.scrollTop = logScrollRef.current.scrollHeight;
        }
      });
    }
  }, [agentLogs, showLogs, autoScroll]);

  const handleRecommend = async () => {
    setLoading(true);
    // 清空之前的日志和结果
    setAgentLogs([]);
    setRecommendations([]);
    setMergedPapers([]);
    setShowLogs(true); // 开始时自动展开日志

    try {
      const resultJson = await GetDailyRecommendations({
        interestQuery: interestQuery.trim() || '',
        platforms: ['arxiv', 'openreview', 'acl'],
        zoteroCollection: '',
        topK: 5,
        maxRecommendations: 20,
        forceCrawl: false,
        dateFrom: dateFrom.trim() || '',
        dateTo: dateTo.trim() || '',
      } as models.main.RecommendOptions);

      const result: RecommendResult = JSON.parse(resultJson);
      
      
      // 检查是否有推荐数据
      if (!result.recommendations || !Array.isArray(result.recommendations)) {
        console.error('推荐数据格式错误:', result.recommendations);
        toast({
          title: "数据格式错误",
          description: "返回的推荐数据格式不正确",
          variant: "destructive",
        });
        return;
      }
      
      // 转换数据格式，确保每个 paper 有 id
      // 注意：SimilarPaper 包含嵌套的 Paper 字段，需要访问 paper.paper
      const transformedRecommendations: RecommendationGroup[] = result.recommendations
        .filter((group: any) => group && group.papers && Array.isArray(group.papers) && group.papers.length > 0)
        .map((group: any, groupIdx: number) => {
          const papers = group.papers
            .map((similarPaper: any, paperIdx: number) => {
              // SimilarPaper 结构：{ Paper: {...}, Similarity: number } 或 { paper: {...}, similarity: number }
              // 需要访问嵌套的 Paper 字段
              const paper = similarPaper.paper || similarPaper.Paper || similarPaper;
              const similarity = similarPaper.similarity || similarPaper.Similarity || 0;
              
              // 调试：打印每篇论文的结构
              if (paperIdx === 0 && groupIdx === 0) {
                console.log('转换第一篇论文，原始 similarPaper:', similarPaper);
                console.log('提取的 paper:', paper);
              }
              
              return {
                id: `${groupIdx}-${paperIdx}-${paper.source || paper.Source || ''}-${paper.sourceId || paper.SourceID || paper.source_id || ''}`,
                sourceId: paper.sourceId || paper.SourceID || paper.source_id || '',
                title: paper.title || paper.Title || '',
                authors: Array.isArray(paper.authors) ? paper.authors : (Array.isArray(paper.Authors) ? paper.Authors : []),
                abstract: paper.abstract || paper.Abstract || '',
                url: paper.url || paper.URL || '',
                published: paper.published || paper.firstSubmittedAt || paper.FirstSubmittedAt || '',
                source: paper.source || paper.Source || '',
                similarity: similarity,
              };
            })
            .filter((p: any) => p.title && p.title.trim() !== ''); // 过滤掉没有标题的论文
          
          return {
            zoteroPaper: {
              title: group.zoteroPaper?.title || group.zoteroPaper?.Title || '',
              authors: Array.isArray(group.zoteroPaper?.authors) 
                ? group.zoteroPaper.authors 
                : (Array.isArray(group.zoteroPaper?.Authors) ? group.zoteroPaper.Authors : []),
              abstract: group.zoteroPaper?.abstract || group.zoteroPaper?.Abstract || '',
            },
            papers: papers,
          };
        })
        .filter((group: RecommendationGroup) => group.papers.length > 0); // 过滤掉没有论文的组

      setRecommendations(transformedRecommendations);
      console.log('已设置 recommendations，数量:', transformedRecommendations.length);
      console.log('转换后的推荐数据:', JSON.stringify(transformedRecommendations, null, 2));
      console.log('转换后的推荐数据总论文数:', transformedRecommendations.reduce((sum, g) => sum + g.papers.length, 0));
      
      // 合并所有推荐组中的论文，去重
      const papersMap = new Map<string, Paper>();
      transformedRecommendations.forEach((group, groupIdx) => {
        group.papers.forEach((paper, paperIdx) => {
          // 使用 source:sourceId 作为唯一标识去重
          const uniqueKey = `${paper.source}:${paper.sourceId}`;
          if (!papersMap.has(uniqueKey)) {
            // 为每篇论文生成唯一 ID
            const paperWithId = {
              ...paper,
              id: paper.id || `${groupIdx}-${paperIdx}-${paper.source}-${paper.sourceId}`,
            };
            papersMap.set(uniqueKey, paperWithId);
          } else {
            // 如果已存在，保留相似度更高的
            const existing = papersMap.get(uniqueKey)!;
            if (paper.similarity > existing.similarity) {
              papersMap.set(uniqueKey, {
                ...paper,
                id: existing.id, // 保持原有 ID
              });
            }
          }
        });
      });
      
      const merged = Array.from(papersMap.values());
      // 按相似度排序
      merged.sort((a, b) => b.similarity - a.similarity);
      setMergedPapers(merged);
      
      // 如果转换后没有数据，但原始数据有，说明转换逻辑有问题
      if (transformedRecommendations.length === 0 && result.recommendations && result.recommendations.length > 0) {
        console.error('数据转换失败：原始数据有', result.recommendations.length, '个组，但转换后为 0');
        console.error('原始第一个组:', JSON.stringify(result.recommendations[0], null, 2));
      }
      
      // 如果有推荐结果，切换到全屏显示
      if (merged.length > 0) {
        setShowRecommendations(true);
      }
      
      const logs = result.agentLogs || [];
      setAgentLogs(logs);
      // 默认不展开日志，优先显示推荐列表
      // 只有在没有推荐结果时才展开日志
      if (transformedRecommendations.length === 0 && logs.length > 0) {
        console.log('没有推荐结果，展开日志');
        setShowLogs(true);
      } else {
        setShowLogs(false);
      }
      toast({
        title: "推荐完成",
        description: merged.length > 0 
          ? `找到 ${merged.length} 篇推荐论文（已去重）`
          : (result.message || "未找到推荐论文")
      });
    } catch (error) {
      console.error('Recommend failed:', error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      
      if (errorMessage.includes("missing APIKey")) {
        toast({
          title: "需要配置 API Key",
          description: "语义推荐功能需要配置 Embedder API Key。请前往设置页面进行配置。",
          duration: 5000,
          action: (
            <Button 
              variant="outline" 
              size="sm" 
              onClick={() => window.location.hash = '#/settings'}
              className="bg-primary text-primary-foreground hover:bg-primary/90"
            >
              去配置
            </Button>
          ),
        });
      } else {
        toast({
          title: "推荐失败",
          description: errorMessage || "获取推荐时出现错误，请重试",
          variant: "destructive",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  const togglePaperSelection = (paperId: string) => {
    const newSelected = new Set(selectedPapers);
    if (newSelected.has(paperId)) {
      newSelected.delete(paperId);
    } else {
      newSelected.add(paperId);
    }
    setSelectedPapers(newSelected);
  };

  const selectAllPapers = () => {
    const allIds = new Set<string>();
    mergedPapers.forEach(paper => {
      if (paper.id) {
        allIds.add(paper.id);
      }
    });
    setSelectedPapers(allIds);
  };

  const clearSelection = () => {
    setSelectedPapers(new Set());
  };

  const handleExport = async () => {
    if (selectedPapers.size === 0) {
      toast({
        title: "请选择论文",
        description: "请至少选择一篇论文进行导出",
        variant: "destructive",
      });
      return;
    }

    if ((exportFormat === 'csv' || exportFormat === 'json') && !exportOutput.trim()) {
      toast({
        title: "请输入输出路径",
        description: "CSV/JSON 格式需要指定输出文件路径",
        variant: "destructive",
      });
      return;
    }

    if (exportFormat === 'feishu' && !exportFeishuName.trim()) {
      toast({
        title: "请输入飞书表格名称",
        description: "飞书导出需要指定表格名称",
        variant: "destructive",
      });
      return;
    }

    try {
      // 构建论文对列表
      const paperPairs: Array<{ source: string; id: string }> = [];
      mergedPapers.forEach(paper => {
        if (paper.id && selectedPapers.has(paper.id)) {
          paperPairs.push({
            source: paper.source,
            id: paper.sourceId
          });
        }
      });

      const result = await ExportSelectionByPapers(
        exportFormat,
        paperPairs,
        exportOutput,
        exportFeishuName,
        exportCollection
      );

      // 如果是飞书导出且有链接，显示可点击的链接
      if (exportFormat === 'feishu' && result) {
        toast({
          title: "导出成功",
          description: (
            <div className="space-y-2">
              <p>已导出到飞书</p>
              <a
                href={result}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => {
                  e.stopPropagation();
                  BrowserOpenURL(result);
                }}
                className="text-primary hover:underline cursor-pointer break-all"
                style={{ userSelect: 'text' }}
                data-selectable="true"
              >
                {result}
              </a>
            </div>
          ),
        });
      } else {
        toast({
          title: "导出成功",
          description: `已导出 ${selectedPapers.size} 篇论文${result ? `到 ${result}` : ''}`,
        });
      }
      
      setExportOpen(false);
      // 保持选择状态，方便用户查看已导出的论文
      // clearSelection();
      
      // 滚动到推荐列表顶部，确保用户能看到推荐内容
      setTimeout(() => {
        const recommendationsElement = document.querySelector('[data-recommendations-list]');
        if (recommendationsElement) {
          recommendationsElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
      }, 100);
    } catch (error) {
      console.error('Export failed:', error);
      toast({
        title: "导出失败",
        description: error instanceof Error ? error.message : "导出过程中出现错误",
        variant: "destructive",
      });
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'Unknown';
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString('zh-CN');
    } catch {
      return dateStr;
    }
  };

  const getSourceBadgeColor = (source: string) => {
    const colors: Record<string, string> = {
      arxiv: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
      openreview: 'bg-purple-500/10 text-purple-600 dark:text-purple-400',
      acl: 'bg-green-500/10 text-green-600 dark:text-green-400',
      ssrn: 'bg-orange-500/10 text-orange-600 dark:text-orange-400',
    };
    return colors[source] || 'bg-gray-500/10 text-gray-600 dark:text-gray-400';
  };

  // 返回搜索界面
  const handleBackToSearch = () => {
    setShowRecommendations(false);
    setMergedPapers([]);
    setRecommendations([]);
    setSelectedPapers(new Set());
    setAgentLogs([]);
    setShowLogs(false);
  };

  // 如果显示推荐结果，全屏显示
  if (showRecommendations && mergedPapers.length > 0) {
    return (
      <div className="flex flex-col h-full overflow-hidden animate-fade-in">
        <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
          {/* 顶部栏：返回按钮和标题 */}
          <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-6 flex-shrink-0">
            <div className="flex items-center gap-4">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleBackToSearch}
                className="gap-2"
              >
                <ArrowLeftLineIcon className="w-4 h-4" />
                返回
              </Button>
              <div className="flex items-center gap-3 flex-1">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <StarLineIcon className="w-5 h-5 text-primary" />
                </div>
                <CardTitle className="text-3xl font-display font-semibold ">
                  推荐论文 ({mergedPapers.length} 篇)
                </CardTitle>
              </div>
            </div>
          </CardHeader>

          <CardContent className="flex-1 flex flex-col overflow-hidden p-8">
            {/* 操作栏 */}
            <div className="flex items-center justify-between mb-4 pb-4 border-b border-border/30 flex-shrink-0">
              <div className="flex items-center gap-4">
                <span className="text-sm text-muted-foreground">
                  已选择 {selectedPapers.size} 篇论文
                </span>
                {selectedPapers.size > 0 && (
                  <>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={selectAllPapers}
                    >
                      全选
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={clearSelection}
                    >
                      清空选择
                    </Button>
                  </>
                )}
              </div>
              {selectedPapers.size > 0 && (
                <Button
                  onClick={() => setExportOpen(true)}
                  className="gap-2"
                >
                  <DownloadLineIcon className="w-4 h-4" />
                  导出选中 ({selectedPapers.size})
                </Button>
              )}
            </div>

            {/* 推荐列表 */}
            <div className="flex-1 overflow-y-auto space-y-3" data-recommendations-list>
              {mergedPapers.map((paper) => {
                const paperId = paper.id || `${paper.source}-${paper.sourceId}`;
                return (
                  <div
                    key={paperId}
                    className={`p-4 rounded-lg border transition-all ${
                      selectedPapers.has(paperId)
                        ? 'border-primary bg-primary/5'
                        : 'border-border/50 hover:border-border hover:bg-secondary/30'
                    }`}
                    style={{ userSelect: 'text' }}
                  >
                    <div className="flex items-start gap-3">
                      <div 
                        onClick={(e) => {
                          e.stopPropagation();
                          togglePaperSelection(paperId);
                        }}
                        className="cursor-pointer"
                        style={{ userSelect: 'none' }}
                      >
                        <Checkbox
                          checked={selectedPapers.has(paperId)}
                          onCheckedChange={() => togglePaperSelection(paperId)}
                          className="mt-1"
                        />
                      </div>
                      <div className="flex-1 min-w-0 select-text" style={{ userSelect: 'text' }}>
                        <div className="flex items-start justify-between gap-2 mb-2">
                          <h4 className="font-medium text-sm leading-snug line-clamp-2 flex-1">
                            {paper.title}
                          </h4>
                          <div className="flex items-center gap-2 flex-shrink-0">
                            <Badge className={getSourceBadgeColor(paper.source)}>
                              {paper.source.toUpperCase()}
                            </Badge>
                            {paper.similarity > 0 && (
                              <Badge variant="outline" className="text-xs">
                                <StarLineIcon className="w-3 h-3 mr-1" />
                                {(paper.similarity * 100).toFixed(0)}%
                              </Badge>
                            )}
                          </div>
                        </div>
                        <div className="text-xs text-muted-foreground mb-2 flex items-center gap-4">
                          <span className="flex items-center gap-1">
                            <BookOpenLineIcon className="w-3 h-3" />
                            {paper.authors.slice(0, 3).join(', ')}
                            {paper.authors.length > 3 && ' et al.'}
                          </span>
                          <span className="flex items-center gap-1">
                            <CalendarLineIcon className="w-3 h-3" />
                            {formatDate(paper.published || '')}
                          </span>
                        </div>
                        <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                          {paper.abstract}
                        </p>
                        <div className="flex items-center gap-2">
                          {paper.url && (
                            <>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-xs"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  BrowserOpenURL(paper.url!);
                                }}
                              >
                                <ExternalLinkLineIcon className="w-3 h-3 mr-1" />
                                查看原文
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-xs"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  navigator.clipboard.writeText(paper.url!);
                                  toast({
                                    title: "已复制",
                                    description: "链接已复制到剪贴板",
                                  });
                                }}
                              >
                                <FileCopyLineIcon className="w-3 h-3 mr-1" />
                                复制链接
                              </Button>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>

        {/* 导出对话框 */}
        <AlertDialog open={exportOpen} onOpenChange={setExportOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>导出论文</AlertDialogTitle>
              <AlertDialogDescription>
                选择导出格式并填写相关信息
              </AlertDialogDescription>
            </AlertDialogHeader>
            <div className="space-y-4 py-4">
              <div>
                <Label>导出格式</Label>
                <Select
                  value={exportFormat}
                  onValueChange={(value) => setExportFormat(value as 'csv' | 'json' | 'zotero' | 'feishu')}
                >
                  <SelectTrigger className="w-full mt-1">
                    <SelectValue placeholder="选择导出格式" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="csv">CSV</SelectItem>
                    <SelectItem value="json">JSON</SelectItem>
                    <SelectItem value="zotero">Zotero</SelectItem>
                    <SelectItem value="feishu">飞书多维表格</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {(exportFormat === 'csv' || exportFormat === 'json') && (
                <div>
                  <Label>输出文件路径</Label>
                  <Input
                    value={exportOutput}
                    onChange={(e) => setExportOutput(e.target.value)}
                    placeholder="例如: papers.csv"
                    className="mt-1"
                  />
                </div>
              )}
              {exportFormat === 'zotero' && (
                <div>
                  <Label>Collection Key (可选)</Label>
                  <Input
                    value={exportCollection}
                    onChange={(e) => setExportCollection(e.target.value)}
                    placeholder="留空则添加到默认位置"
                    className="mt-1"
                  />
                </div>
              )}
              {exportFormat === 'feishu' && (
                <div>
                  <Label>表格名称</Label>
                  <Input
                    value={exportFeishuName}
                    onChange={(e) => setExportFeishuName(e.target.value)}
                    placeholder="例如: 推荐论文"
                    className="mt-1"
                  />
                </div>
              )}
            </div>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction onClick={handleExport}>导出</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    );
  }

  // 默认显示搜索界面
  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="space-y-2">
            <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                <StarLineIcon className="w-5 h-5 text-primary" />
              </div>
              <CardTitle className="text-3xl font-display font-semibold">Daily Recommendations</CardTitle>
            </div>
            <CardDescription className="text-muted-foreground">
              基于您的 Zotero 库或输入的兴趣关键词，为您推荐指定日期范围内新发布的相似论文（需要配置对应的 Zotero key）。默认推荐今天的论文。
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="flex-1 flex flex-col overflow-hidden p-8">
          {/* 搜索区域 */}
          <div className="space-y-4 mb-6">
            <div className="flex gap-4">
              <div className="flex-1">
                <Label htmlFor="interest-query" className="text-sm font-medium mb-2 block">
                  今日感兴趣的主题（可选，留空则基于 Zotero 推荐）
                </Label>
                <div className="flex gap-2">
                  <Input
                    id="interest-query"
                    placeholder="例如：transformer, attention mechanism, large language models..."
                    value={interestQuery}
                    onChange={(e) => setInterestQuery(e.target.value)}
                    onKeyPress={(e) => e.key === 'Enter' && !loading && handleRecommend()}
                    className="flex-1"
                  />
                  <Button
                    onClick={handleRecommend}
                    disabled={loading}
                    className="px-6"
                  >
                    {loading ? (
                      <>
                        <RefreshLineIcon className="w-4 h-4 mr-2 animate-spin" />
                        推荐中...
                      </>
                    ) : (
                      <>
                        <SearchLineIcon className="w-4 h-4 mr-2" />
                        获取推荐
                      </>
                    )}
                  </Button>
                </div>
              </div>
            </div>
            <div className="flex gap-4">
              <div className="flex-1">
                <Label htmlFor="date-from" className="text-sm font-medium mb-2 block">
                  开始日期（可选，默认今天）
                </Label>
                <Input
                  id="date-from"
                  type="date"
                  value={dateFrom}
                  onChange={(e) => setDateFrom(e.target.value)}
                  className="flex-1"
                />
              </div>
              <div className="flex-1">
                <Label htmlFor="date-to" className="text-sm font-medium mb-2 block">
                  结束日期（可选，默认今天）
                </Label>
                <Input
                  id="date-to"
                  type="date"
                  value={dateTo}
                  onChange={(e) => setDateTo(e.target.value)}
                  className="flex-1"
                />
              </div>
            </div>
            <div className="text-xs text-muted-foreground">
              注意：arXiv 等平台在周末和节假日不发刊，这些日期会被自动跳过
            </div>
          </div>

          {/* Agent 日志展示 */}
          {agentLogs.length > 0 && (
            <div className="mb-6">
              <Card className="border-border/50 bg-card/50 overflow-hidden">
                <CardHeader className="pb-3 border-b border-border/30 bg-muted/30">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <TerminalBoxLineIcon className="w-5 h-5 text-primary" />
                      <CardTitle className="text-base font-medium">
                        LLM 交互日志
                      </CardTitle>
                      <Badge variant="secondary" className="ml-2">
                        {agentLogs.length} 条
                      </Badge>
                    </div>
                    <div className="flex items-center gap-3">
                      {showLogs && (
                        <div className="flex items-center gap-2">
                          <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
                            <Checkbox
                              checked={autoScroll}
                              onCheckedChange={(checked) => setAutoScroll(checked === true)}
                              className="h-4 w-4"
                            />
                            <span className="text-xs text-muted-foreground">自动滚动</span>
                          </label>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 px-2 text-xs"
                            onClick={() => setAgentLogs([])}
                            title="清空日志"
                          >
                            清空
                          </Button>
                        </div>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => setShowLogs(!showLogs)}
                      >
                        <ArrowDownLineIcon className={`w-4 h-4 transition-transform duration-200 ${showLogs ? 'rotate-180' : ''}`} />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                
                {showLogs && (
                  <CardContent className="p-0 relative">
                    <div 
                      ref={logScrollRef}
                      className="h-[400px] w-full bg-card overflow-y-auto overflow-x-hidden scrollbar-thin scrollbar-thumb-secondary scrollbar-track-transparent"
                      onScroll={(e) => {
                        // 如果用户向上滚动，暂停自动滚动
                        const target = e.target as HTMLDivElement;
                        if (target.scrollHeight - target.scrollTop - target.clientHeight > 50) {
                          if (autoScroll) setAutoScroll(false);
                        } else {
                          // 如果用户滚动到底部，恢复自动滚动（可选，或者只通过按钮恢复）
                          // if (!autoScroll) setAutoScroll(true);
                        }
                      }}
                    >
                      <div className="p-4 space-y-2">
                        {agentLogs.map((log, idx) => {
                          const getLogStyle = (type: string) => {
                            switch (type) {
                              case 'user': return 'bg-blue-50/50 border-blue-200 text-blue-900 dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-300';
                              case 'assistant': return 'bg-green-50/50 border-green-200 text-green-900 dark:bg-green-900/20 dark:border-green-800 dark:text-green-300';
                              case 'error': return 'bg-red-50/50 border-red-200 text-red-900 dark:bg-red-900/20 dark:border-red-800 dark:text-red-300';
                              case 'tool_call': return 'bg-purple-50/50 border-purple-200 text-purple-900 dark:bg-purple-900/20 dark:border-purple-800 dark:text-purple-300';
                              case 'tool_result': return 'bg-gray-50/50 border-gray-200 text-gray-900 dark:bg-gray-800/20 dark:border-gray-700 dark:text-gray-300';
                              default: return 'bg-gray-50/50 border-gray-200 text-gray-900 dark:bg-gray-800/20 dark:border-gray-700 dark:text-gray-300';
                            }
                          };

                          const getLogLabel = (type: string) => {
                             switch (type) {
                               case 'user': return '用户';
                               case 'assistant': return '助手';
                               case 'error': return '错误';
                               case 'tool_call': return '工具调用';
                               case 'tool_result': return '工具结果';
                               default: return '其他';
                             }
                          };

                          return (
                            <div
                              key={idx}
                              className={`p-3 rounded-lg border text-sm transition-all ${getLogStyle(log.type)}`}
                            >
                              <div className="flex items-center justify-between gap-2 mb-1.5 border-b border-black/5 dark:border-white/5 pb-1.5">
                                <Badge
                                  variant={log.type === 'error' ? 'destructive' : 'outline'}
                                  className="text-[10px] h-5 px-1.5 uppercase tracking-wider bg-background/50 backdrop-blur-sm"
                                >
                                  {getLogLabel(log.type)}
                                </Badge>
                                <span className="text-[10px] opacity-70 font-mono">
                                  {log.timestamp}
                                </span>
                              </div>
                              <div className="text-sm whitespace-pre-wrap break-words font-mono leading-relaxed select-text pb-2 w-full" style={{ userSelect: 'text' }}>
                                {log.content}
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                    {autoScroll && (
                       <div className="absolute bottom-2 right-4 pointer-events-none flex justify-end">
                         <Badge variant="secondary" className="bg-background/80 backdrop-blur-sm shadow-sm animate-fade-in text-[10px] h-5">
                           <ArrowDownLineIcon className="w-3 h-3 mr-1 animate-bounce" />
                           自动滚动中
                         </Badge>
                       </div>
                    )}
                  </CardContent>
                )}
              </Card>
            </div>
          )}

          {/* 空状态 */}
          {mergedPapers.length === 0 && !loading && (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center space-y-4">
                <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto">
                  <StarLineIcon className="w-8 h-8 text-primary" />
                </div>
                <div>
                  <h3 className="text-lg font-medium mb-1">开始获取推荐</h3>
                  <p className="text-sm text-muted-foreground">
                    输入您感兴趣的主题，或留空以基于 Zotero 库获取推荐
                  </p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 导出对话框 */}
      <AlertDialog open={exportOpen} onOpenChange={setExportOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>导出论文</AlertDialogTitle>
            <AlertDialogDescription>
              选择导出格式并填写相关信息
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-4 py-4">
            <div>
              <Label>导出格式</Label>
              <Select
                value={exportFormat}
                onValueChange={(value) => setExportFormat(value as 'csv' | 'json' | 'zotero' | 'feishu')}
              >
                <SelectTrigger className="w-full mt-1">
                  <SelectValue placeholder="选择导出格式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="csv">CSV</SelectItem>
                  <SelectItem value="json">JSON</SelectItem>
                  <SelectItem value="zotero">Zotero</SelectItem>
                  <SelectItem value="feishu">飞书多维表格</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {(exportFormat === 'csv' || exportFormat === 'json') && (
              <div>
                <Label>输出文件路径</Label>
                <Input
                  value={exportOutput}
                  onChange={(e) => setExportOutput(e.target.value)}
                  placeholder="例如: papers.csv"
                  className="mt-1"
                />
              </div>
            )}
            {exportFormat === 'zotero' && (
              <div>
                <Label>Collection Key (可选)</Label>
                <Input
                  value={exportCollection}
                  onChange={(e) => setExportCollection(e.target.value)}
                  placeholder="留空则添加到默认位置"
                  className="mt-1"
                />
              </div>
            )}
            {exportFormat === 'feishu' && (
              <div>
                <Label>表格名称</Label>
                <Input
                  value={exportFeishuName}
                  onChange={(e) => setExportFeishuName(e.target.value)}
                  placeholder="例如: 推荐论文"
                  className="mt-1"
                />
              </div>
            )}
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleExport}>导出</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default RecommendView;

