import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
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
import { Label } from './ui/label';
import { Badge } from './ui/badge';
import { Separator } from './ui/separator';
import SearchLineIcon from 'remixicon-react/SearchLineIcon';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { Download, FileText, ExternalLink, RefreshCw, Star, BookOpen, Calendar, TrendingUp, X, Copy, Globe } from 'lucide-react';
import { useToast } from './ui/use-toast';

interface Paper {
  id: string; // 数据库内部ID
  sourceId: string; // 平台特定ID（如arXiv ID、SSRN ID等）
  title: string;
  authors: string[];
  abstract: string;
  url: string;
  published: string;
  source: string;
  similarity: number;
}

const PapersView: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [papers, setPapers] = useState<Paper[]>([]);
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [selectedPapers, setSelectedPapers] = useState<string[]>([]);
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const { toast } = useToast();


  const [sourceFilter, setSourceFilter] = useState<'all' | 'arxiv' | 'openreview' | 'acl' | 'ssrn'>('all');
  const [dateFrom, setDateFrom] = useState('');
  const [dateUntil, setDateUntil] = useState('');
  const [topK, setTopK] = useState<number>(100);
  const [semantic, setSemantic] = useState<boolean>(true);
  const [examplesText, setExamplesText] = useState('');
  const [exportOpen, setExportOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<'csv'|'json'|'zotero'|'feishu'>('csv');
  const [exportOutput, setExportOutput] = useState('');
  const [exportCollection, setExportCollection] = useState('');
  const [exportFeishuName, setExportFeishuName] = useState('');
  const [feishuUrls, setFeishuUrls] = useState<Array<{url: string; name: string; timestamp: string}>>([]);

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      toast({
        title: "请输入搜索关键词",
        description: "请填写查询关键词后再开始搜索",
        variant: "destructive",
      });
      return;
    }

    setSearching(true);
    try {
      const { SearchWithOptions } = await import('../../wailsjs/go/main/App');
      // 解析 examples: 每行一个 JSON，或简单的 "title|abstract" 形式
      const examples: any[] = [];
      examplesText.split('\n').forEach(line => {
        const t = line.trim();
        if (!t) return;
        try {
          const obj = JSON.parse(t);
          if (obj.title || obj.abstract) examples.push({ Title: obj.title || '', Abstract: obj.abstract || '' });
        } catch {
          const parts = t.split('|');
          if (parts.length >= 1) {
            examples.push({ Title: parts[0]?.trim() || '', Abstract: (parts[1]||'').trim() });
          }
        }
      });

      const resp = await SearchWithOptions({
        Query: searchQuery,
        Examples: examples,
        Semantic: semantic,
        TopK: topK,
        Limit: 0,
        Source: sourceFilter === 'all' ? '' : sourceFilter,
        From: dateFrom,
        Until: dateUntil,
        ComputeEmbed: false,
        EmbedBatch: 100
      } as any);

      // resp 是 JSON 字符串，解析为 SimilarPaper[]
      const data = JSON.parse(resp || '[]') as any[];
      const mapped: Paper[] = data.map((item, idx) => {
        const p = item.Paper || {};
        const sim = item.Similarity || 0;
        
        // 尝试多种可能的字段名
        let sourceId = p.SourceID || p.source_id || p.SourceId || '';
        
        // 如果 sourceId 为空，尝试从 URL 提取（备用方案）
        if (!sourceId && p.URL) {
          // 尝试从 URL 中提取 ID
          if (p.Source === 'ssrn') {
            const match = p.URL.match(/abstract_id=([^&]+)/);
            if (match) sourceId = match[1];
          } else if (p.Source === 'arxiv') {
            const match = p.URL.match(/(\d{4}\.\d{4,5})/);
            if (match) sourceId = match[1];
          } else if (p.Source === 'openreview') {
            const match = p.URL.match(/forum\?id=([^&]+)/);
            if (match) sourceId = match[1];
          }
        }
        
        return {
          id: String(p.ID ?? idx),
          sourceId: sourceId,
          title: p.Title || '',
          authors: Array.isArray(p.Authors) ? p.Authors : [],
          abstract: p.Abstract || '',
          url: p.URL || '',
          published: p.FirstAnnouncedAt || p.PublishedAt || '',
          source: p.Source || '',
          similarity: sim,
        } as Paper;
      });
      setPapers(mapped);
      toast({ title: "搜索完成", description: `找到 ${mapped.length} 篇相关论文` });
    } catch (error) {
      console.error('Search failed:', error);
      const errorMessage = error instanceof Error ? error.message : String(error);

      if (errorMessage.includes("missing APIKey")) {
        toast({
          title: "需要配置 API Key",
          description: "语义搜索功能需要配置 Embedder API Key。请前往设置页面进行配置。",
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
          title: "搜索失败",
          description: errorMessage || "搜索过程中出现错误，请重试",
          variant: "destructive",
        });
      }
    } finally {
      setSearching(false);
    }
  };

  const handleSelectPaper = (paperId: string) => {
    setSelectedPapers(prev => 
      prev.includes(paperId) 
        ? prev.filter(id => id !== paperId)
        : [...prev, paperId]
    );
  };

  const handleSelectAll = () => {
    if (selectedPapers.length === papers.length) {
      setSelectedPapers([]);
    } else {
      setSelectedPapers(papers.map(p => p.id));
    }
  };

  const handleExport = () => {
    if (selectedPapers.length === 0) {
      toast({
        title: "请选择要导出的论文",
        description: "请先选择要导出的论文",
        variant: "destructive",
      });
      return;
    }
    setExportOpen(true);
  };

  const confirmExport = async () => {
    try {
      const selectedPaperObjs = papers.filter(p => selectedPapers.includes(p.id));
      
      if (selectedPaperObjs.length === 0) {
        toast({
          title: '导出失败',
          description: '没有选择任何论文进行导出',
          variant: 'destructive',
        });
        return;
      }
      
      // 检查是否所有论文都有有效的 sourceId 和 source
      const validPapers = selectedPaperObjs.filter(p => p.sourceId && p.source);
      
      if (validPapers.length === 0) {
        // 检查是否所有论文都缺少 sourceId
        const papersWithoutSourceId = selectedPaperObjs.filter(p => !p.sourceId || !p.source);
        if (papersWithoutSourceId.length === selectedPaperObjs.length) {
          toast({
            title: '导出失败',
            description: `所有 ${selectedPaperObjs.length} 篇论文都缺少必要的平台信息（sourceId），无法导出。这可能是因为数据不完整，请尝试重新搜索。`,
            variant: 'destructive',
          });
          return;
        }
      }
      
      // 如果有部分论文缺少 sourceId，给出提示但继续导出有效的论文
      const papersWithoutSourceId = selectedPaperObjs.filter(p => !p.sourceId || !p.source);
      if (papersWithoutSourceId.length > 0) {
        toast({
          title: '提示',
          description: `${papersWithoutSourceId.length} 篇论文缺少 sourceId，将被跳过。将导出 ${validPapers.length} 篇有效论文。`,
          variant: 'default',
        });
      }
      
      // 检查所有选中论文的 source 是否一致
      const sources = new Set(validPapers.map(p => p.source));
      const uniqueSources = Array.from(sources);
      
      if (uniqueSources.length === 0) {
        toast({
          title: '导出失败',
          description: '无法确定论文来源，请重新选择',
          variant: 'destructive',
        });
        return;
      }
      
      // 使用 sourceId（平台特定ID）而不是数据库内部ID
      // 只导出有 sourceId 的论文
      const sourceIdsToExport = validPapers
        .map(p => p.sourceId)
        .filter(id => id && id.trim() !== ''); // 确保不为空
      
      // 如果只有一个 source，使用简单的 ExportSelection
      if (uniqueSources.length === 1) {
        const source = uniqueSources[0];
        const { ExportSelection } = await import('../../wailsjs/go/main/App');
        const result = await ExportSelection(
          exportFormat,
          source,
          sourceIdsToExport, // 使用 sourceId
          exportFormat === 'csv' || exportFormat === 'json' ? exportOutput : '',
          exportFormat === 'feishu' ? (exportFeishuName || 'Papers') : '',
          exportFormat === 'zotero' ? exportCollection : '',
        );
        handleExportSuccess(result);
      } else {
        // 多个 source，使用 ExportSelectionByPapers 方法
        // 构建 paperPairs 格式: [{source: 'arxiv', id: '123'}, {source: 'ssrn', id: '456'}, ...]
        const paperPairs = validPapers.map(p => ({
          source: p.source,
          id: p.sourceId
        }));
        
        try {
          const { ExportSelectionByPapers } = await import('../../wailsjs/go/main/App');
          const result = await ExportSelectionByPapers(
            exportFormat,
            paperPairs as any, // Wails 会自动序列化
            exportFormat === 'csv' || exportFormat === 'json' ? exportOutput : '',
            exportFormat === 'feishu' ? (exportFeishuName || 'Papers') : '',
            exportFormat === 'zotero' ? exportCollection : '',
          );
          handleExportSuccess(result);
          toast({
            title: '导出成功',
            description: `已成功导出 ${uniqueSources.length} 个平台（${uniqueSources.join(', ')}）的 ${validPapers.length} 篇论文`,
            variant: 'default',
          });
        } catch (err: any) {
          // 如果 ExportSelectionByPapers 不存在（Wails绑定未更新），降级到单source导出
          if (err?.message?.includes('ExportSelectionByPapers') || err?.message?.includes('not defined')) {
            console.warn('ExportSelectionByPapers not available, falling back to single-source export');
            const firstSource = uniqueSources[0];
            const firstSourcePapers = validPapers.filter(p => p.source === firstSource);
            toast({
              title: '提示',
              description: `选择了 ${uniqueSources.length} 个不同来源的论文，当前版本只支持单个平台导出，将只导出 ${firstSource} 来源的 ${firstSourcePapers.length} 篇论文`,
              variant: 'default',
            });
            const { ExportSelection } = await import('../../wailsjs/go/main/App');
            const result = await ExportSelection(
              exportFormat,
              firstSource,
              firstSourcePapers.map(p => p.sourceId),
              exportFormat === 'csv' || exportFormat === 'json' ? exportOutput : '',
              exportFormat === 'feishu' ? (exportFeishuName || 'Papers') : '',
              exportFormat === 'zotero' ? exportCollection : '',
            );
            handleExportSuccess(result);
          } else {
            throw err;
          }
        }
      }
      setExportOpen(false);
      setExportOutput('');
      setExportCollection('');
      setExportFeishuName('');
    } catch (e: any) {
      console.error('Export failed:', e);
      let errorMessage = '请检查配置或稍后重试';
      if (e && typeof e === 'string') {
        errorMessage = e;
      } else if (e && e.message) {
        errorMessage = e.message;
      }
      toast({ 
        title: '导出失败', 
        description: errorMessage, 
        variant: 'destructive' 
      });
    }
  };

  // 从 localStorage 加载飞书 URL 历史
  const loadFeishuUrls = () => {
    try {
      const stored = localStorage.getItem('feishu_export_urls');
      if (stored) {
        const urls = JSON.parse(stored);
        setFeishuUrls(urls);
      }
    } catch (e) {
      console.error('Failed to load feishu URLs:', e);
    }
  };

  // 保存飞书 URL 到 localStorage
  const saveFeishuUrl = (url: string, name: string) => {
    if (!url || !url.trim()) return;
    
    try {
      const newEntry = {
        url: url.trim(),
        name: name || '未命名导出',
        timestamp: new Date().toISOString()
      };
      
      const existing = JSON.parse(localStorage.getItem('feishu_export_urls') || '[]') as Array<{url: string; name: string; timestamp: string}>;
      
      // 避免重复添加相同的 URL
      const isDuplicate = existing.some(item => item.url === newEntry.url);
      if (!isDuplicate) {
        const updated = [newEntry, ...existing].slice(0, 50); // 最多保存50条
        localStorage.setItem('feishu_export_urls', JSON.stringify(updated));
        setFeishuUrls(updated);
      } else {
        // 即使重复也更新列表状态
        setFeishuUrls(existing);
      }
    } catch (e) {
      console.error('Failed to save feishu URL:', e);
    }
  };

  // 删除飞书 URL
  const removeFeishuUrl = (url: string) => {
    try {
      const updated = feishuUrls.filter(item => item.url !== url);
      localStorage.setItem('feishu_export_urls', JSON.stringify(updated));
      setFeishuUrls(updated);
      toast({
        title: '已删除',
        description: '已从历史记录中删除',
      });
    } catch (e) {
      console.error('Failed to remove feishu URL:', e);
    }
  };

  // 复制 URL 到剪贴板
  const copyFeishuUrl = (url: string) => {
    navigator.clipboard.writeText(url).then(() => {
      toast({
        title: '已复制',
        description: 'URL 已复制到剪贴板',
      });
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  };

  // 组件加载时读取 localStorage
  useEffect(() => {
    loadFeishuUrls();
  }, []);

  const handleExportSuccess = (result: string) => {
    // result: 对于 csv/json 返回文件路径；feishu 返回 URL；zotero 返回空串
    if (exportFormat === 'feishu' && result) {
      // 保存飞书 URL
      saveFeishuUrl(result, exportFeishuName || 'Papers');
      
      toast({
        title: '导出完成',
        description: (
          <span>
            已上传到飞书: <a className="underline cursor-pointer text-primary" onClick={()=>BrowserOpenURL(result)}>{result}</a>
          </span>
        ),
        duration: 5000,
      });
    } else if ((exportFormat==='csv'||exportFormat==='json') && result) {
      toast({
        title: '导出完成',
        description: (
          <span>
            已保存到: <a className="underline cursor-pointer text-primary" onClick={()=>BrowserOpenURL(`file://${result}`)}>{result}</a>
          </span>
        )
      });
    } else {
      toast({ title: '导出完成', description: '操作成功' });
    }
  };

  const handleImportJSON = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (file) {
        const reader = new FileReader();
        reader.onload = (e) => {
          try {
            const jsonData = JSON.parse(e.target?.result as string);
            console.log('Imported JSON data:', jsonData);
            toast({
              title: "JSON 导入成功",
              description: `已导入 ${jsonData.length || 0} 个查询示例`,
            });
          } catch (error) {
            console.error('JSON parse error:', error);
            toast({
              title: "JSON 导入失败",
              description: "文件格式不正确，请检查 JSON 格式",
              variant: "destructive",
            });
          }
        };
        reader.readAsText(file);
      }
    };
    input.click();
  };

  const openPaper = (url: string) => {
    if (!url) return;
    try { BrowserOpenURL(url); } catch { window.open(url, '_blank'); }
  };

  const getSimilarityColor = (similarity: number) => {
    if (similarity >= 0.9) return 'text-success';
    if (similarity >= 0.8) return 'text-info';
    if (similarity >= 0.7) return 'text-warning';
    return 'text-muted-foreground';
  };

  const getSimilarityBg = (similarity: number) => {
    if (similarity >= 0.9) return 'bg-success/10 border-success/20';
    if (similarity >= 0.8) return 'bg-info/10 border-info/20';
    if (similarity >= 0.7) return 'bg-warning/10 border-warning/20';
    return 'bg-muted/50 border-border';
  };

  const formatAuthors = (authors: string[]) => {
    if (!authors || authors.length === 0) return 'Unknown';
    if (authors.length <= 3) return authors.join(', ');
    return `${authors.slice(0, 3).join(', ')} +${authors.length - 3} more`;
  };

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <SearchLineIcon className="w-5 h-5 text-primary" />
                </div>
                <CardTitle className="text-3xl font-display font-semibold">Search Papers</CardTitle>
              </div>
              <CardDescription className="text-base text-muted-foreground ml-13">
                依据关键词或导入现有 json 文件快速向量化查询最相关的论文
              </CardDescription>
            </div>
            
            <div className="flex items-center gap-2">
              <Button
                onClick={() => setPapers([])}
                disabled={loading}
                size="sm"
                variant="outline"
                className="hover-lift"
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                Clear
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto overflow-x-hidden px-8 py-8" style={{ overflowY: 'auto' }}>
          <div className="max-w-7xl mx-auto space-y-6">
            {/* Search Area */}
            <div className="glass-card p-6 rounded-2xl space-y-4">
              <div className="flex items-end gap-4">
                <div className="flex-1">
                  <Label htmlFor="search-query" className="text-sm font-medium mb-2 block">
                    Search Query
                  </Label>
                  <Input
                    id="search-query"
                    placeholder="e.g., machine learning, deep learning, NLP..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                    className="h-12 text-base"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    onClick={handleImportJSON}
                    variant="outline"
                    size="lg"
                    className="hover-lift h-12"
                  >
                    <FileText className="mr-2 h-4 w-4" />
                    Import JSON
                  </Button>
                  <Button
                    onClick={handleSearch}
                    disabled={searching || !searchQuery.trim()}
                    size="lg"
                    className="gradient-primary hover:shadow-lg hover:shadow-primary/30 transition-all duration-300 h-12 px-8"
                  >
                    {searching ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                        Searching...
                      </>
                    ) : (
                      <>
                        <SearchLineIcon className="mr-2 h-5 w-5" />
                        Search
                      </>
                    )}
                  </Button>
                </div>
              </div>

              {/* Filters */}
              <div className="grid grid-cols-4 gap-4">
                <div>
                  <Label className="text-sm font-medium mb-2 block">Source</Label>
                  <Select value={sourceFilter} onValueChange={(v:any)=>setSourceFilter(v)}>
                    <SelectTrigger>
                      <SelectValue placeholder="All" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All</SelectItem>
                      <SelectItem value="arxiv">arXiv</SelectItem>
                      <SelectItem value="openreview">OpenReview</SelectItem>
                      <SelectItem value="acl">ACL</SelectItem>
                      <SelectItem value="ssrn">SSRN</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label className="text-sm font-medium mb-2 block">From</Label>
                  <Input type="date" value={dateFrom} onChange={(e)=>setDateFrom(e.target.value)} />
                </div>
                <div>
                  <Label className="text-sm font-medium mb-2 block">Until</Label>
                  <Input type="date" value={dateUntil} onChange={(e)=>setDateUntil(e.target.value)} />
                </div>
                <div>
                  <Label className="text-sm font-medium mb-2 block">TopK</Label>
                  <Input type="number" value={topK} onChange={(e)=>setTopK(parseInt(e.target.value)||100)} />
                  <label className="flex items-center gap-2 text-xs mt-2">
                    <input type="checkbox" checked={semantic} onChange={(e)=>setSemantic(e.target.checked)} />
                    Semantic
                  </label>
                </div>
              </div>

              {/* Examples Input */}
              <div className="space-y-2">
                <Label className="text-sm font-medium mb-2 block">Examples (每行一条 JSON 或 title|abstract)</Label>
                <textarea
                  value={examplesText}
                  onChange={(e)=>setExamplesText(e.target.value)}
                  rows={4}
                  className="w-full px-3 py-2 bg-background border border-input rounded-md text-sm"
                  placeholder='{"title":"...","abstract":"..."}\nTitle only\nTitle|Abstract'
                />
              </div>
            </div>

            {/* Results Area */}
            {papers.length > 0 && (
              <div className="space-y-6 animate-slide-in">
                {/* Stats and Actions Bar */}
                <div className="glass-card p-4 rounded-xl">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-6">
                      <div className="flex items-center gap-2">
                        <BookOpen className="w-4 h-4 text-muted-foreground" />
                        <span className="text-sm font-medium">
                          {papers.length} Papers Found
                        </span>
                      </div>
                      <Separator orientation="vertical" className="h-4" />
                      <div className="flex items-center gap-2">
                        <Star className="w-4 h-4 text-warning" />
                        <span className="text-sm font-medium">
                          {selectedPapers.length} Selected
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        onClick={handleSelectAll}
                        variant="outline"
                        size="sm"
                        className="hover-lift"
                      >
                        {selectedPapers.length === papers.length ? 'Deselect All' : 'Select All'}
                      </Button>
                      <Button
                        onClick={handleExport}
                        disabled={selectedPapers.length === 0}
                        size="sm"
                        className="gradient-primary"
                      >
                        <Download className="mr-2 h-4 w-4" />
                        Export ({selectedPapers.length})
                      </Button>
                    </div>
                  </div>
                </div>

                {/* Papers Grid */}
                <div className="grid grid-cols-1 gap-4">
                  {papers.map((paper, index) => {
                    const isSelected = selectedPapers.includes(paper.id);
                    return (
                      <div
                        key={paper.id}
                        className={`glass-card p-6 rounded-2xl hover-lift transition-all duration-300 ${
                          isSelected ? 'ring-2 ring-primary shadow-lg shadow-primary/10' : ''
                        }`}
                        style={{ animationDelay: `${index * 50}ms` }}
                      >
                        <div className="flex gap-4">
                          {/* Checkbox */}
                          <div className="flex-shrink-0 pt-1">
                            <input
                              type="checkbox"
                              checked={isSelected}
                              onChange={() => handleSelectPaper(paper.id)}
                              className="w-5 h-5 rounded border-2 border-input cursor-pointer"
                            />
                          </div>

                          {/* Content */}
                          <div className="flex-1 space-y-3">
                            {/* Header */}
                            <div className="flex items-start justify-between gap-4">
                              <div className="flex-1">
                                <h3 className="text-lg font-display font-semibold text-foreground leading-tight mb-2 hover:text-primary transition-colors cursor-pointer">
                                  {paper.title}
                                </h3>
                                <div className="flex items-center gap-3 text-sm text-muted-foreground">
                                  <span className="flex items-center gap-1">
                                    <Calendar className="w-3.5 h-3.5" />
                                    {new Date(paper.published).toLocaleDateString()}
                                  </span>
                                  <Separator orientation="vertical" className="h-3" />
                                  <Badge variant="outline" className="text-xs">
                                    {paper.source}
                                  </Badge>
                                </div>
                              </div>

                              {/* Similarity Score */}
                              <div className={`flex-shrink-0 px-4 py-2 rounded-xl border ${getSimilarityBg(paper.similarity)}`}>
                                <div className="flex items-center gap-2">
                                  <TrendingUp className={`w-4 h-4 ${getSimilarityColor(paper.similarity)}`} />
                                  <span className={`text-sm font-bold ${getSimilarityColor(paper.similarity)}`}>
                                    {(paper.similarity * 100).toFixed(1)}%
                                  </span>
                                </div>
                              </div>
                            </div>

                            {/* Authors */}
                            <div className="flex items-center gap-2 text-sm">
                              <span className="text-muted-foreground">Authors:</span>
                              <span className="text-foreground">
                                {paper.authors.slice(0, 3).join(', ')}
                                {paper.authors.length > 3 && ` +${paper.authors.length - 3} more`}
                              </span>
                            </div>

                            {/* Abstract */}
                            <p className="text-sm text-muted-foreground leading-relaxed line-clamp-2">
                              {paper.abstract}
                            </p>

                            {/* Actions */}
                            <div className="flex items-center gap-2 pt-2">
                              <Button
                                onClick={() => openPaper(paper.url)}
                                size="sm"
                                variant="outline"
                                className="hover-lift"
                              >
                                <ExternalLink className="mr-2 h-3.5 w-3.5" />
                                View Paper
                              </Button>
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Feishu URLs History - Show when no papers */}
            {papers.length === 0 && !searching && feishuUrls.length > 0 && (
              <div className="glass-card p-6 rounded-2xl">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-2">
                    <Globe className="w-5 h-5 text-primary" />
                    <h3 className="text-lg font-semibold">飞书导出历史</h3>
                    <Badge variant="outline">{feishuUrls.length}</Badge>
                  </div>
                  <Button
                    onClick={() => {
                      localStorage.removeItem('feishu_export_urls');
                      setFeishuUrls([]);
                      toast({ title: '已清空', description: '已清空所有历史记录' });
                    }}
                    variant="outline"
                    size="sm"
                  >
                    清空历史
                  </Button>
                </div>
                <div className="space-y-2 max-h-96 overflow-y-auto">
                  {feishuUrls.map((item, idx) => (
                    <div
                      key={idx}
                      className="flex items-center gap-3 p-3 rounded-lg bg-background/50 hover:bg-background/80 transition-colors group border border-border/50"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-sm font-medium">{item.name}</span>
                          <span className="text-xs text-muted-foreground">
                            {new Date(item.timestamp).toLocaleString('zh-CN', {
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit'
                            })}
                          </span>
                        </div>
                        <a
                          href={item.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-sm text-primary hover:underline truncate block"
                          onClick={(e) => {
                            e.stopPropagation();
                            BrowserOpenURL(item.url);
                          }}
                        >
                          {item.url}
                        </a>
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          onClick={() => copyFeishuUrl(item.url)}
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          title="复制 URL"
                        >
                          <Copy className="w-4 h-4" />
                        </Button>
                        <Button
                          onClick={() => removeFeishuUrl(item.url)}
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0 text-destructive"
                          title="删除"
                        >
                          <X className="w-4 h-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Empty State */}
            {papers.length === 0 && !searching && feishuUrls.length === 0 && (
              <div className="glass-card p-12 rounded-2xl text-center">
                <div className="max-w-md mx-auto space-y-4">
                  <div className="w-20 h-20 rounded-2xl gradient-primary/10 flex items-center justify-center mx-auto">
                    <SearchLineIcon className="w-10 h-10 text-primary" />
                  </div>
                  <h3 className="text-xl font-display font-semibold">No Papers Found</h3>
                  <p className="text-muted-foreground">
                    Enter a search query or import a JSON file to find relevant papers
                  </p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Feishu URLs History */}
      {feishuUrls.length > 0 && (
        <div className="glass-card p-4 rounded-xl mb-4">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <Globe className="w-4 h-4 text-primary" />
              <Label className="text-sm font-medium">飞书导出历史</Label>
              <Badge variant="outline" className="text-xs">{feishuUrls.length}</Badge>
            </div>
            <Button
              onClick={() => {
                localStorage.removeItem('feishu_export_urls');
                setFeishuUrls([]);
                toast({ title: '已清空', description: '已清空所有历史记录' });
              }}
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-xs"
            >
              清空
            </Button>
          </div>
          <div className="space-y-2 max-h-48 overflow-y-auto">
            {feishuUrls.map((item, idx) => (
              <div
                key={idx}
                className="flex items-center gap-2 p-2 rounded-lg bg-background/50 hover:bg-background/80 transition-colors group"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs font-medium truncate">{item.name}</span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(item.timestamp).toLocaleString('zh-CN', {
                        month: 'short',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit'
                      })}
                    </span>
                  </div>
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-primary hover:underline truncate block"
                    onClick={(e) => {
                      e.stopPropagation();
                      BrowserOpenURL(item.url);
                    }}
                  >
                    {item.url}
                  </a>
                </div>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Button
                    onClick={() => copyFeishuUrl(item.url)}
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 p-0"
                    title="复制 URL"
                  >
                    <Copy className="w-3.5 h-3.5" />
                  </Button>
                  <Button
                    onClick={() => removeFeishuUrl(item.url)}
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 p-0 text-destructive"
                    title="删除"
                  >
                    <X className="w-3.5 h-3.5" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Export Dialog */}
      <AlertDialog open={exportOpen} onOpenChange={setExportOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>导出所选论文</AlertDialogTitle>
            <AlertDialogDescription>
              选择导出格式并填写必要参数。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label className="text-sm font-medium">格式</Label>
              <Select value={exportFormat} onValueChange={(v:any)=>setExportFormat(v)}>
                <SelectTrigger>
                  <SelectValue placeholder="选择导出格式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="csv">CSV</SelectItem>
                  <SelectItem value="json">JSON</SelectItem>
                  <SelectItem value="zotero">Zotero</SelectItem>
                  <SelectItem value="feishu">飞书</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {(exportFormat==='csv'||exportFormat==='json') && (
              <div className="space-y-2">
                <Label>输出路径 (可选)</Label>
                <Input value={exportOutput} onChange={(e)=>setExportOutput(e.target.value)} placeholder="out/papers.csv 或 papers.json" />
              </div>
            )}
            {exportFormat==='zotero' && (
              <div className="space-y-2">
                <Label>Collection Key (可选)</Label>
                <Input value={exportCollection} onChange={(e)=>setExportCollection(e.target.value)} placeholder="如 ABC123XY；为空则默认位置" />
              </div>
            )}
            {exportFormat==='feishu' && (
              <div className="space-y-2">
                <Label>飞书数据集名称</Label>
                <Input value={exportFeishuName} onChange={(e)=>setExportFeishuName(e.target.value)} placeholder="例如: 论文数据集" />
              </div>
            )}
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmExport}>开始导出</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default PapersView;