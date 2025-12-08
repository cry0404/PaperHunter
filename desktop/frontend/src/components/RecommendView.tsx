import React, { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Badge } from './ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Checkbox } from './ui/checkbox';

import { 
  Search, 
  Download, 
  RefreshCcw, 
  ExternalLink, 
  BookOpen, 
  Calendar, 
  Copy, 
  ArrowLeft,
  FileJson,
  Loader2
} from 'lucide-react';

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
  const { t } = useTranslation();
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
        title: "File Selected",
        description: `Selected: ${file.name}`,
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
        title: t('common.error'),
        description: "Please select at least one paper.",
        variant: "destructive",
      });
      return;
    }

    if ((exportFormat === 'csv' || exportFormat === 'json') && !exportOutput.trim()) {
      toast({
        title: t('common.error'),
        description: "Please specify output path.",
        variant: "destructive",
      });
      return;
    }

    if (exportFormat === 'feishu' && !exportFeishuName.trim()) {
      toast({
        title: t('common.error'),
        description: "Please specify Feishu table name.",
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
          title: t('common.success'),
          description: (
            <div className="space-y-2">
              <p>Exported to Feishu</p>
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
          title: t('common.success'),
          description: `Exported ${selectedPapers.size} papers${result ? ` to ${result}` : ''}`,
        });
      }
      
      setExportOpen(false);
    } catch (error) {
      console.error('Export failed:', error);
      toast({
        title: t('common.error'),
        description: error instanceof Error ? error.message : "Export failed",
        variant: "destructive",
      });
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'Unknown';
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString();
    } catch {
      return dateStr;
    }
  };

  const getSourceBadgeColor = (source: string) => {
    // Anthropic-style badge colors
    const colors: Record<string, string> = {
      arxiv: 'bg-anthropic-blue/10 text-anthropic-blue hover:bg-anthropic-blue/20 border-anthropic-blue/20',
      openreview: 'bg-anthropic-orange/10 text-anthropic-orange hover:bg-anthropic-orange/20 border-anthropic-orange/20',
      acl: 'bg-anthropic-green/10 text-anthropic-green hover:bg-anthropic-green/20 border-anthropic-green/20',
      ssrn: 'bg-anthropic-orange/10 text-anthropic-orange hover:bg-anthropic-orange/20 border-anthropic-orange/20',
    };
    return colors[source] || 'bg-anthropic-mid/10 text-anthropic-mid hover:bg-anthropic-mid/20 border-anthropic-mid/20';
  };

  const handleBackToSearch = () => {
    setShowRecommendations(false);
    clearSelection();
  };

  // 如果显示推荐结果，全屏显示
  if (showRecommendations && mergedPapers.length > 0) {
    return (
      <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
        <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
          {/* 顶部栏：返回按钮和标题 */}
          <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-6 flex-shrink-0">
            <div className="flex items-center gap-4">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleBackToSearch}
                className="gap-2 font-sans"
              >
                <ArrowLeft className="w-4 h-4" />
                {t('recommend.back')}
              </Button>
              <div className="flex items-center gap-3 flex-1">
               
                <div>
                  <CardTitle className="text-3xl font-sans font-medium tracking-tight">
                    {t('recommend.title')} <span className="text-muted-foreground font-normal ml-2 text-xl">({mergedPapers.length})</span>
                  </CardTitle>
                  <p className="text-sm text-muted-foreground mt-1 font-serif">
                    {t('recommend.subtitle')}
                  </p>
                </div>
              </div>
            </div>
          </CardHeader>

          <CardContent className="flex-1 flex flex-col overflow-hidden p-8 bg-background">
            {/* 操作栏 */}
            <div className="flex items-center justify-between mb-6 pb-4 border-b border-border/30 flex-shrink-0">
              <div className="flex items-center gap-4">
                <span className="text-sm text-muted-foreground font-sans">
                  {t('recommend.selected')}: {selectedPapers.size}
                </span>
                {selectedPapers.size > 0 && (
                  <>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={selectAllPapers}
                      className="font-sans"
                    >
                      {t('recommend.selectAll')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={clearSelection}
                      className="font-sans"
                    >
                      {t('recommend.clearSelection')}
                    </Button>
                  </>
                )}
              </div>
              {selectedPapers.size > 0 && (
                <Button
                  onClick={() => setExportOpen(true)}
                  className="gap-2 font-sans bg-anthropic-dark text-anthropic-light hover:bg-anthropic-dark/90"
                >
                  <Download className="w-4 h-4" />
                  {t('recommend.exportSelection')} ({selectedPapers.size})
                </Button>
              )}
            </div>

            {/* 推荐列表 - 直接显示所有推荐论文 */}
            <div className="flex-1 overflow-y-auto space-y-4 pr-2" data-recommendations-list>
              {recommendations.map((group, groupIdx) => (
                <div key={groupIdx} className="space-y-4">
                  {/* 推荐论文列表 */}
                  <div className="space-y-4">
                    {group.papers.map((paper, paperIdx) => {
                      const paperId = paper.id || `${paper.source}-${paper.sourceId}`;
                      const isSelected = selectedPapers.has(paperId);
                      return (
                        <div
                          key={paperId}
                          className={`p-6 rounded-xl border transition-all duration-200 group ${
                            isSelected
                              ? 'border-primary bg-primary/5 shadow-sm'
                              : 'border-border/40 hover:border-primary/30 hover:shadow-md bg-card'
                          }`}
                          style={{ userSelect: 'text' }}
                        >
                          <div className="flex items-start gap-4">
                            <div
                              onClick={(e) => {
                                e.stopPropagation();
                                togglePaperSelection(paperId);
                              }}
                              className="cursor-pointer pt-1"
                              style={{ userSelect: 'none' }}
                            >
                              <Checkbox
                                checked={isSelected}
                                onCheckedChange={() => togglePaperSelection(paperId)}
                                className="data-[state=checked]:bg-primary data-[state=checked]:border-primary"
                              />
                            </div>
                            <div className="flex-1 min-w-0 select-text" style={{ userSelect: 'text' }}>
                              <div className="flex items-start justify-between gap-4 mb-3">
                                <h5 className="font-medium text-lg font-sans leading-tight text-foreground flex-1">
                                  {paper.title}
                                </h5>
                                <div className="flex items-center gap-2 flex-shrink-0">
                                  <Badge variant="outline" className={`${getSourceBadgeColor(paper.source)} border font-normal font-sans tracking-wide`}>
                                    {paper.source.toUpperCase()}
                                  </Badge>
                                  {paper.similarity > 0 && (
                                    <Badge variant="secondary" className="text-xs font-mono">
                                      {(paper.similarity * 100).toFixed(0)}% {t('recommend.match')}
                                    </Badge>
                                  )}
                                </div>
                              </div>
                              
                              <div className="text-sm text-muted-foreground mb-3 flex flex-wrap items-center gap-x-6 gap-y-2 font-sans">
                                <span className="flex items-center gap-1.5">
                                  <BookOpen className="w-4 h-4 opacity-70" />
                                  <span className="font-medium text-foreground/80">
                                    {paper.authors.slice(0, 3).join(', ')}
                                    {paper.authors.length > 3 && ' et al.'}
                                  </span>
                                </span>
                                <span className="flex items-center gap-1.5">
                                  <Calendar className="w-4 h-4 opacity-70" />
                                  {formatDate(paper.published || '')}
                                </span>
                              </div>
                              
                              <p className="text-base text-foreground/80 leading-relaxed font-serif line-clamp-3 mb-4 max-w-4xl">
                                {paper.abstract}
                              </p>
                              
                              <div className="flex items-center gap-3 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                                {paper.url && (
                                  <>
                                    <Button
                                      variant="secondary"
                                      size="sm"
                                      className="h-8 text-xs font-sans"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        BrowserOpenURL(paper.url!);
                                      }}
                                    >
                                      <ExternalLink className="w-3 h-3 mr-1.5" />
                                      {t('recommend.readPaper')}
                                    </Button>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-8 text-xs font-sans hover:bg-secondary"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        navigator.clipboard.writeText(paper.url!);
                                        toast({
                                          title: t('common.copied'),
                                          description: "Link copied to clipboard",
                                        });
                                      }}
                                    >
                                      <Copy className="w-3 h-3 mr-1.5" />
                                      {t('recommend.copyLink')}
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
          <AlertDialogContent className="font-sans">
            <AlertDialogHeader>
              <AlertDialogTitle>{t('export.title')}</AlertDialogTitle>
              <AlertDialogDescription>
                {t('export.description')}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <div className="space-y-4 py-4">
              <div>
                <Label>{t('export.format')}</Label>
                <Select
                  value={exportFormat}
                  onValueChange={(value) => setExportFormat(value as 'csv' | 'json' | 'zotero' | 'feishu')}
                >
                  <SelectTrigger className="w-full mt-1">
                    <SelectValue placeholder="Select format" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="csv">CSV</SelectItem>
                    <SelectItem value="json">JSON</SelectItem>
                    <SelectItem value="zotero">Zotero</SelectItem>
                    <SelectItem value="feishu">Feishu / Lark</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {(exportFormat === 'csv' || exportFormat === 'json') && (
                <div>
                  <Label>{t('export.outputPath')}</Label>
                  <Input
                    value={exportOutput}
                    onChange={(e) => setExportOutput(e.target.value)}
                    placeholder="e.g., papers.csv"
                    className="mt-1"
                  />
                </div>
              )}
              {exportFormat === 'zotero' && (
                <div>
                  <Label>{t('export.collectionKey')}</Label>
                  <Input
                    value={exportCollection}
                    onChange={(e) => setExportCollection(e.target.value)}
                    placeholder="Leave empty for default"
                    className="mt-1"
                  />
                </div>
              )}
              {exportFormat === 'feishu' && (
                <div>
                  <Label>{t('export.feishuName')}</Label>
                  <Input
                    value={exportFeishuName}
                    onChange={(e) => setExportFeishuName(e.target.value)}
                    placeholder="e.g., Recommended Papers"
                    className="mt-1"
                  />
                </div>
              )}
            </div>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
              <AlertDialogAction onClick={handleExport}>{t('common.confirm')}</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    );
  }

  // 默认显示搜索界面
  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="space-y-2 max-w-3xl">
            <div className="flex items-center gap-3">
              <CardTitle className="text-3xl font-sans font-medium tracking-tight text-foreground">
                {t('recommend.title')}
              </CardTitle>
            </div>
            <CardDescription className="text-base text-muted-foreground font-serif leading-relaxed">
              {t('recommend.subtitle')}
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="flex-1 flex flex-col overflow-hidden p-8 bg-background">
          {/* 搜索区域 */}
          <div className="space-y-8 max-w-4xl">
            
            <div className="space-y-4">
              <Label htmlFor="interest-query" className="text-base font-medium font-sans flex items-center gap-2">
                <Search className="w-4 h-4 text-primary" />
                {t('recommend.interests')}
              </Label>
              <div className="flex gap-3">
                <Input
                  id="interest-query"
                  placeholder={t('recommend.placeholder')}
                  value={interestQuery}
                  onChange={(e) => setInterestQuery(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && !loading && handleRecommend()}
                  className="flex-1 h-12 text-lg font-serif px-4 shadow-sm border-border/60 focus:border-primary/50 focus:ring-primary/20"
                />
                <Button
                  onClick={handleRecommend}
                  disabled={loading}
                  className="h-12 px-8 font-sans text-base bg-anthropic-dark text-anthropic-light hover:bg-anthropic-dark/90 shadow-sm transition-all hover:scale-[1.02]"
                >
                  {loading ? (
                    <>
                      <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                      {t('recommend.analyzing')}
                    </>
                  ) : (
                    <>
                      <Search className="w-5 h-5 mr-2" />
                      {t('recommend.recommendBtn')}
                    </>
                  )}
                </Button>
              </div>
              <p className="text-sm text-muted-foreground font-sans pl-1">
                Tip: More specific descriptions yield better matches.
              </p>
            </div>

            <Separator className="bg-border/40" />

            {/* 本地文件导入选项 */}
            <div className="space-y-4">
              <Label className="text-base font-medium font-sans flex items-center gap-2">
                <FileJson className="w-4 h-4 text-primary" />
                {t('recommend.localFile')}
              </Label>
              <div className="flex gap-3 items-center p-4 rounded-xl border border-border/40 bg-card/30 hover:bg-card/50 transition-colors">
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
                  className="font-sans border-dashed border-border hover:border-primary/50 hover:bg-secondary/50"
                >
                  {t('recommend.selectJson')}
                </Button>
                {localFilePath ? (
                  <div className="flex-1 flex items-center gap-2 px-3 py-2 bg-secondary/50 rounded-md text-sm font-mono text-muted-foreground">
                    <span className="truncate">{localFilePath}</span>
                  </div>
                ) : (
                  <span className="text-sm text-muted-foreground font-sans flex-1">
                    {t('recommend.uploadTip')}
                  </span>
                )}
                {useLocalFile && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={clearLocalFile}
                    className="text-xs text-muted-foreground hover:text-destructive"
                  >
                    {t('recommend.clear')}
                  </Button>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="date-from" className="text-sm font-medium font-sans flex items-center gap-2">
                  <Calendar className="w-4 h-4 text-muted-foreground" />
                  {t('recommend.date')}
                </Label>
                <Input
                  id="date-from"
                  type="date"
                  value={dateFrom}
                  onChange={(e) => setDateFrom(e.target.value)}
                  className="font-sans"
                />
              </div>
              {/* Optional End Date can be hidden or kept if needed, keeping simple for now */}
            </div>

            <div className="flex items-start gap-3 p-4 bg-anthropic-mid/5 rounded-lg border border-anthropic-mid/10">
              <RefreshCcw className="w-4 h-4 text-muted-foreground mt-0.5" />
              <div className="text-sm text-muted-foreground font-sans leading-relaxed">
                <strong className="font-medium text-foreground">{t('recommend.noteTitle')}:</strong> {t('recommend.noteContent')}
              </div>
            </div>
          </div>

          {/* 空状态 / 引导 */}
          {mergedPapers.length === 0 && !loading && (
            <div className="flex-1 flex items-center justify-center opacity-30 pointer-events-none select-none">
              {/* Optional background illustration or watermark */}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default RecommendView;
