import React, { useState, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Badge } from './ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Checkbox } from './ui/checkbox';

import SearchLineIcon from 'remixicon-react/SearchLineIcon';
import DownloadLineIcon from 'remixicon-react/DownloadLineIcon';
import RefreshLineIcon from 'remixicon-react/RefreshLineIcon';
import ExternalLinkLineIcon from 'remixicon-react/ExternalLinkLineIcon';
import BookOpenLineIcon from 'remixicon-react/BookOpenLineIcon';
import CalendarLineIcon from 'remixicon-react/CalendarLineIcon';
import FileCopyLineIcon from 'remixicon-react/FileCopyLineIcon';
import ArrowLeftLineIcon from 'remixicon-react/ArrowLeftLineIcon';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { useToast } from './ui/use-toast';
import { ExportSelectionByPapers } from '../../wailsjs/go/main/App';
import { useRecommendContext } from '../context/RecommendContext';
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
import { Separator } from './ui/separator';

const RecommendView: React.FC = () => {
  const {
    loading,
    recommendations,
    mergedPapers,
    showRecommendations,
    interestQuery,
    dateFrom,
    dateTo,
    localFilePath,
    useLocalFile,
    selectedPapers,
    setInterestQuery,
    setDateFrom,
    setDateTo,
    setLocalFilePath,
    setUseLocalFile,
    setShowRecommendations,
    togglePaperSelection,
    selectAllPapers,
    clearSelection,
    startRecommend,
    clearResults
  } = useRecommendContext();

  const [exportOpen, setExportOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<'csv' | 'json' | 'zotero' | 'feishu'>('csv');
  const [exportOutput, setExportOutput] = useState('');
  const [exportCollection, setExportCollection] = useState('');
  const [exportFeishuName, setExportFeishuName] = useState('');
  const { toast } = useToast();

  // 本地文件导入状态
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleRecommend = async () => {
    await startRecommend();
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setLocalFilePath((file as any).path);
      setUseLocalFile(true);
      toast({
        title: "文件已选择",
        description: `已选择文件: ${file.name}`,
      });
    }
  };

  const openFileDialog = () => {
    fileInputRef.current?.click();
  };

  const clearLocalFile = () => {
    setLocalFilePath('');
    setUseLocalFile(false);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
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

  const handleBackToSearch = () => {
    setShowRecommendations(false);
    clearSelection();
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
               
                <div>
                  <CardTitle className="text-3xl font-display font-semibold ">
                    今日推荐 ({mergedPapers.length} 篇)
                  </CardTitle>
                  <p className="text-sm text-muted-foreground mt-1">
                    基于您的兴趣从今天发布的arXiv论文中智能筛选
                  </p>
                </div>
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

            {/* 推荐列表 - 直接显示所有推荐论文 */}
            <div className="flex-1 overflow-y-auto space-y-3" data-recommendations-list>
              {recommendations.map((group, groupIdx) => (
                <div key={groupIdx} className="space-y-2">
                  {/* 推荐论文列表 */}
                  <div className="space-y-2">
                    {group.papers.map((paper, paperIdx) => {
                      const paperId = paper.id || `${paper.source}-${paper.sourceId}`;
                      return (
                        <div
                          key={paperId}
                          className={`p-3 rounded-lg border transition-all ${
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
                                <h5 className="font-medium text-sm leading-snug line-clamp-2 flex-1">
                                  {paper.title}
                                </h5>
                                <div className="flex items-center gap-2 flex-shrink-0">
                                  <Badge className={getSourceBadgeColor(paper.source)}>
                                    {paper.source.toUpperCase()}
                                  </Badge>
                                  {paper.similarity > 0 && (
                                    <Badge variant="outline" className="text-xs">

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
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* 导出对话框 (复用) */}
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
              
              <CardTitle className="text-3xl font-display font-semibold">Daily Recommendations</CardTitle>
            </div>
            <CardDescription className="text-muted-foreground">
              专注于今日 arXiv 论文推荐：基于您的兴趣描述或 Zotero 库，从今天发布的 arXiv 论文中智能筛选并推荐最相关的内容。
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="flex-1 flex flex-col overflow-hidden p-8">
          {/* 搜索区域 */}
          <div className="space-y-4 mb-6">
            <div className="flex gap-4">
              <div className="flex-1">
                <Label htmlFor="interest-query" className="text-sm font-medium mb-2 block">
                  研究兴趣描述（推荐填写，用于精准匹配今日arXiv论文）
                </Label>
                <div className="flex gap-2">
                  <Input
                    id="interest-query"
                    placeholder="详细描述您的研究兴趣，例如：Multi-agent reinforcement learning for improving LLM reasoning capabilities through collaborative debate..."
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
                        分析今日arXiv...
                      </>
                    ) : (
                      <>
                        <SearchLineIcon className="w-4 h-4 mr-2" />
                        智能推荐
                      </>
                    )}
                  </Button>
                </div>
              </div>
            </div>

            {/* 本地文件导入选项 */}
            <div className="border border-border/30 rounded-lg p-4 bg-card/30">
              <div className="flex items-center justify-between mb-3">
                <Label className="text-sm font-medium">或使用本地论文文件</Label>
                {useLocalFile && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={clearLocalFile}
                    className="text-xs"
                  >
                    清除文件
                  </Button>
                )}
              </div>
              <div className="flex gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".json"
                  onChange={handleFileSelect}
                  className="hidden"
                />
                <Button
                  variant="outline"
                  onClick={openFileDialog}
                  className="flex-1"
                >
                  选择本地文件 (JSON)
                </Button>
                {localFilePath && (
                  <div className="flex-1 flex items-center gap-2 px-3 py-2 bg-secondary/50 rounded text-sm">
                    <span className="truncate">{localFilePath}</span>
                  </div>
                )}
              </div>
              {useLocalFile && (
                <p className="text-xs text-muted-foreground mt-2">
                  支持格式：JSON文件需包含title/abstract字段
                </p>
              )}
            </div>

            <div className="flex gap-4">
              <div className="flex-1">
                <Label htmlFor="date-from" className="text-sm font-medium mb-2 block">
                  推荐日期（默认今天，专注于当日arXiv论文）
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
                  结束日期（可选，通常与开始日期相同）
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
            <div className="flex justify-between items-center text-xs text-muted-foreground">
              <span>专注arXiv（因为这个更新最快，产💩最多）：系统只爬取和分析今日arXiv新发布论文，周末和节假日自动跳过（arXiv不发刊）</span>
                <Separator orientation="vertical" className="bg-border/50" />
               {loading && (
                 <span className="text-primary animate-pulse font-medium">
                    正在分析今日arXiv论文，可前往 Logs 页面查看详细进度
                 </span>
               )}
            </div>
          </div>

          {/* 空状态 / 引导 */}
          {mergedPapers.length === 0 && !loading && (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center space-y-4">
              
                <div>
                  <h3 className="text-lg font-medium mb-1">获取今日推荐</h3>
                  <p className="text-sm text-muted-foreground">
                    描述您的研究兴趣，AI将从今天发布的arXiv论文中为您推荐您可能最感兴趣的内容
                  </p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default RecommendView;
