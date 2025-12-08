import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table"
import { Input } from "./ui/input"
import { Button } from "./ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "./ui/card"
import { Badge } from "./ui/badge"
import { Label } from './ui/label';
import { Separator } from './ui/separator';
import { Checkbox } from './ui/checkbox';
import { 
    Search, 
    ChevronLeft, 
    ChevronRight, 
    ExternalLink, 
    RefreshCw, 

    Sparkles,
    Download,
    Filter,
    Calendar,
    TrendingUp,
   
} from "lucide-react"
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime"
import { useToast } from "./ui/use-toast"
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "./ui/sheet"

interface Paper {
    ID: number;
    Source: string;
    SourceID: string;
    Title: string;
    Authors: string[];
    Abstract: string;
    URL: string;
    FirstAnnouncedAt: string;
    Similarity?: number;
}

interface PaperListResponse {
    papers: Paper[];
    total: number;
}

const LibraryView: React.FC = () => {
    const { t } = useTranslation();
    const [mode, setMode] = useState<'basic' | 'semantic'>('basic');
    const [papers, setPapers] = useState<Paper[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [taskId, setTaskId] = useState<string | null>(null);
    

    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [search, setSearch] = useState('');
    const [source, setSource] = useState('all');
    const [dateFrom, setDateFrom] = useState('');
    const [dateUntil, setDateUntil] = useState('');


    const [semantic, setSemantic] = useState<boolean>(true);
    const [topK, setTopK] = useState<number>(100);
    const [examplesText, setExamplesText] = useState('');
    const [showAdvancedSemantic, setShowAdvancedSemantic] = useState(false);


    const [selectedPapers, setSelectedPapers] = useState<string[]>([]);
    const [exportOpen, setExportOpen] = useState(false);
    const [exportFormat, setExportFormat] = useState<'csv'|'json'|'zotero'|'feishu'>('csv');
    const [exportOutput, setExportOutput] = useState('');
    const [exportCollection, setExportCollection] = useState('');
    const [exportFeishuName, setExportFeishuName] = useState('');
    const [feishuUrls, setFeishuUrls] = useState<Array<{url: string; name: string; timestamp: string}>>([]);
    

    const [selectedPaper, setSelectedPaper] = useState<Paper | null>(null);
    const { toast } = useToast();

    const loadTaskFromHash = useCallback(() => {
        const hash = window.location.hash || '';
        const match = hash.match(/taskId=([^&]+)/);
        setTaskId(match ? decodeURIComponent(match[1]) : null);
    }, []);
    const loadTaskPapers = useCallback(async (id: string) => {
        setLoading(true);
        setSelectedPapers([]);
        try {
            const { GetCrawlTaskPapers } = await import('../../wailsjs/go/main/App');
            const data = await GetCrawlTaskPapers(id);
            const list = JSON.parse(data || '[]') as Paper[];
            setPapers(list || []);
            setTotal((list || []).length);
            setPage(1);
        } catch (error) {
            console.error("Failed to load task papers:", error);
            toast({
                title: t('common.error'),
                description: t('library.toast.loadError'),
                variant: "destructive",
            });
        } finally {
            setLoading(false);
        }
    }, [toast, t]);


    const fetchPapers = useCallback(async () => {
        if (taskId) return; // 任务视图不走常规列表
        if (mode === 'semantic') {
             handleSemanticSearch();
             return;
        }

        setLoading(true);
        try {
            // @ts-ignore
            const { GetPapers } = await import('../../wailsjs/go/main/App');
            const result = await GetPapers(page, pageSize, source, search) as PaperListResponse;
            setPapers(result.papers || []);
            setTotal(result.total || 0);
        } catch (error) {
            console.error("Failed to fetch papers:", error);
            toast({
                title: t('common.error'),
                description: t('library.toast.loadError'),
                variant: "destructive",
            });
        } finally {
            setLoading(false);
        }
    }, [page, pageSize, source, search, mode, toast, taskId, t]); 
    useEffect(() => {
        if (taskId) return;
        if (mode === 'basic') {
            const timer = setTimeout(() => {
                fetchPapers();
            }, 500);
            return () => clearTimeout(timer);
        }
    }, [fetchPapers, mode, taskId]);

    useEffect(() => {
        loadTaskFromHash();
        const handler = () => loadTaskFromHash();
        window.addEventListener('hashchange', handler);
        return () => window.removeEventListener('hashchange', handler);
    }, [loadTaskFromHash]);

    useEffect(() => {
        if (taskId) {
            loadTaskPapers(taskId);
        } else {
            fetchPapers();
        }
    }, [taskId, loadTaskPapers, fetchPapers]);

    // 加载飞书的导出历史
    useEffect(() => {
        try {
          const stored = localStorage.getItem('feishu_export_urls');
          if (stored) {
            setFeishuUrls(JSON.parse(stored));
          }
        } catch (e) {
          console.error('Failed to load feishu URLs:', e);
        }
    }, []);

    // 切换到语义检索模式
    const handleSemanticSearch = async () => {
        if (!search.trim()) {
             if (mode === 'semantic') {
                 return;
             }
        }

        setLoading(true);
        try {
            const { SearchWithOptions } = await import('../../wailsjs/go/main/App');
            
            const examples: any[] = [];
            if (examplesText) {
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
            }

            const resp = await SearchWithOptions({
                Query: search,
                Examples: examples,
                Semantic: semantic,
                TopK: topK,
                Limit: 0,
                Source: source === 'all' ? '' : source,
                From: dateFrom,
                Until: dateUntil,
                ComputeEmbed: false, //默认存在 embedding 部分
                EmbedBatch: 100
            } as any);

            const data = JSON.parse(resp || '[]') as any[];
            

            const mapped: Paper[] = data.map((item: any, idx: number) => {
                const p = item.Paper || {};
                const sim = item.Similarity || 0;
                return {
                    ID: p.ID ?? idx,
                    Source: p.Source || '',
                    SourceID: p.SourceID || '',
                    Title: p.Title || '',
                    Authors: p.Authors || [],
                    Abstract: p.Abstract || '',
                    URL: p.URL || '',
                    FirstAnnouncedAt: p.FirstAnnouncedAt || '',
                    Similarity: sim
                };
            });

            setPapers(mapped);
            setTotal(mapped.length); 
            setPage(1); 
            
            toast({ title: t('library.semanticSearch'), description: t('library.toast.searchSuccess', { count: mapped.length }) });

        } catch (error) {
            console.error('Search failed:', error);
            toast({
                title: t('library.toast.searchError'),
                description: t('library.toast.searchError'),
                variant: "destructive",
            });
        } finally {
            setLoading(false);
        }
    };

    const toggleMode = () => {
        const newMode = mode === 'basic' ? 'semantic' : 'basic';
        setMode(newMode);
        setPapers([]);
        setTotal(0);
        if (newMode === 'basic') {
            // Trigger fetch for basic mode
            setTimeout(() => fetchPapers(), 100);
        }
    };

    const openPaper = (url: string) => {
        if (!url) return;
        BrowserOpenURL(url);
    };


    const handleSelectPaper = (paperId: string) => {
        setSelectedPapers(prev => 
          prev.includes(paperId) 
            ? prev.filter(id => id !== paperId)
            : [...prev, paperId]
        );
    };

    const handleSelectAll = () => {
        const currentIds = papers.map(p => String(p.ID));
        const allSelected = currentIds.every(id => selectedPapers.includes(id));
        
        if (allSelected) {
            setSelectedPapers(prev => prev.filter(id => !currentIds.includes(id)));
        } else {
            const newSelected = new Set([...selectedPapers, ...currentIds]);
            setSelectedPapers(Array.from(newSelected));
        }
    };


    const handleExport = () => {
        if (selectedPapers.length === 0) {
          toast({
            title: t('library.toast.noSelection'),
            description: t('library.toast.noSelection'),
            variant: "destructive",
          });
          return;
        }
        setExportOpen(true);
    };

    const confirmExport = async () => {
        try {
             const selectedPaperObjs = papers.filter(p => selectedPapers.includes(String(p.ID)));
             
             if (selectedPaperObjs.length < selectedPapers.length) {
                 toast({
                     title: t('library.exportDialog.warning'),
                     description: t('library.exportDialog.warning'),
                     variant: "default"
                 });
             }

             if (selectedPaperObjs.length === 0) {
                return;
             }

             const sources = new Set(selectedPaperObjs.map(p => p.Source));
             const uniqueSources = Array.from(sources);
             const validPapers = selectedPaperObjs.filter(p => p.SourceID && p.Source);
             
             if (uniqueSources.length === 1) {
                const source = uniqueSources[0];
                const { ExportSelection } = await import('../../wailsjs/go/main/App');
                const result = await ExportSelection(
                  exportFormat,
                  source,
                  validPapers.map(p => p.SourceID),
                  exportFormat === 'csv' || exportFormat === 'json' ? exportOutput : '',
                  exportFormat === 'feishu' ? (exportFeishuName || 'Papers') : '',
                  exportFormat === 'zotero' ? exportCollection : '',
                );
                handleExportSuccess(result);
             } else {
                const paperPairs = validPapers.map(p => ({
                    source: p.Source,
                    id: p.SourceID
                }));
                const { ExportSelectionByPapers } = await import('../../wailsjs/go/main/App');
                const result = await ExportSelectionByPapers(
                    exportFormat,
                    paperPairs,
                    exportFormat === 'csv' || exportFormat === 'json' ? exportOutput : '',
                    exportFormat === 'feishu' ? (exportFeishuName || 'Papers') : '',
                    exportFormat === 'zotero' ? exportCollection : '',
                );
                handleExportSuccess(result);
             }
             setExportOpen(false);

        } catch (error: any) {
            console.error("Export failed", error);
            toast({ title: t('library.toast.exportFailed'), description: error.message || "Unknown error", variant: "destructive" });
        }
    };

    const handleExportSuccess = (result: string) => {
        if (exportFormat === 'feishu' && result) {
            const newEntry = {
                url: result,
                name: exportFeishuName || 'Papers',
                timestamp: new Date().toISOString()
            };
            const updated = [newEntry, ...feishuUrls].slice(0, 50);
            localStorage.setItem('feishu_export_urls', JSON.stringify(updated));
            setFeishuUrls(updated);
            
            toast({
                title: t('library.toast.exportSuccess'),
                description: (
                  <span>
                    {t('library.toast.feishuSuccess')}: <a className="underline cursor-pointer text-primary" onClick={()=>BrowserOpenURL(result)}>Open</a>
                  </span>
                ),
            });
        } else if ((exportFormat==='csv'||exportFormat==='json') && result) {
             toast({
                title: t('library.toast.exportSuccess'),
                description: (
                  <span>
                    {t('library.toast.savedTo')}: <a className="underline cursor-pointer text-primary" onClick={()=>BrowserOpenURL(`file://${result}`)}>{result}</a>
                  </span>
                )
              });
        } else {
            toast({ title: t('library.toast.exportSuccess'), description: "Operation completed." });
        }
    };

    const getSimilarityColor = (similarity?: number) => {
        if (!similarity) return 'text-muted-foreground';
        if (similarity >= 0.9) return 'text-success';
        if (similarity >= 0.8) return 'text-info';
        if (similarity >= 0.7) return 'text-warning';
        return 'text-muted-foreground';
    };

    const formatDate = (dateStr: string) => {
        if (!dateStr) return 'N/A';
        return new Date(dateStr).toLocaleDateString(undefined, { 
            year: 'numeric', 
            month: 'short', 
            day: 'numeric' 
        });
    };

    return (
        <div className="flex flex-col h-full overflow-hidden animate-fade-in">
             <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
                <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-6 flex-shrink-0">
                    <div className="flex items-center justify-between">
                         <div className="space-y-1">
                            <div className="flex items-center gap-3">
                               
                                <CardTitle className="text-3xl font-display font-semibold">{t('library.title')}</CardTitle>
                                <Badge variant="outline" className="ml-2">
                                    {mode === 'semantic' ? t('library.semanticSearch') : t('library.standardView')}
                                </Badge>
                            </div>
                             <CardDescription className="text-base text-muted-foreground ml-13">
                                {taskId ? t('library.taskViewDesc') : t('library.defaultDesc')}
                            </CardDescription>
                         </div>
                         <div className="flex items-center gap-2">
                            <Button 
                                variant={mode === 'semantic' ? "secondary" : "ghost"}
                                size="sm" 
                                onClick={toggleMode}
                            >
                                {mode === 'semantic' ? t('library.switchToStandard') : t('library.switchToSemantic')}
                            </Button>
                            <Separator orientation="vertical" className="h-6 mx-1" />
                            <Button variant="outline" size="sm" onClick={() => {
                                if (mode === 'semantic') handleSemanticSearch();
                                else fetchPapers();
                            }} disabled={loading}>
                                <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                                {t('library.refresh')}
                            </Button>
                            <Button variant="outline" size="sm" onClick={handleExport} disabled={selectedPapers.length === 0}>
                                <Download className="w-4 h-4 mr-2" />
                                {t('library.export')} ({selectedPapers.length})
                            </Button>
                         </div>
                    </div>
                </CardHeader>

                <CardContent className="flex-1 overflow-hidden p-0 flex flex-col">

                    <div className="p-4 border-b border-border/30 bg-card/10 flex flex-col gap-4">
                        <div className="flex gap-4 items-center">
                            <div className="relative flex-1">
                                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                                <Input 
                                    placeholder={mode === 'semantic' ? t('library.searchPlaceholderSemantic') : t('library.searchPlaceholderStandard')}
                                    className="pl-9 bg-background/50 h-10"
                                    value={search}
                                    onChange={(e) => { setSearch(e.target.value); if (mode === 'basic') setPage(1); }}
                                    onKeyDown={(e) => e.key === 'Enter' && mode === 'semantic' && handleSemanticSearch()}
                                />
                            </div>
                            
                            <Select value={source} onValueChange={(v) => { setSource(v); if (mode === 'basic') setPage(1); }}>
                                <SelectTrigger className="w-[140px] bg-background/50">
                                    <SelectValue placeholder={t('library.source')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="all">{t('library.allSources')}</SelectItem>
                                    <SelectItem value="arxiv">arXiv</SelectItem>
                                    <SelectItem value="openreview">OpenReview</SelectItem>
                                    <SelectItem value="acl">ACL</SelectItem>
                                    <SelectItem value="ssrn">SSRN</SelectItem>
                                </SelectContent>
                            </Select>

                            {mode === 'semantic' && (
                                <Button 
                                    onClick={handleSemanticSearch} 
                                    disabled={loading || !search.trim()}
                                    variant="default"
                                >
                                    {t('library.search')}
                                </Button>
                            )}
                            
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button variant="outline" size="icon">
                                        <Filter className="w-4 h-4" />
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end" className="w-64">
                                    <DropdownMenuLabel>{t('library.filters')}</DropdownMenuLabel>
                                    <DropdownMenuSeparator />
                                    <div className="p-2 space-y-2">
                                        <div className="grid gap-1.5">
                                            <Label className="text-xs">{t('library.fromDate')}</Label>
                                            <Input type="date" className="h-8 text-xs" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} />
                                        </div>
                                        <div className="grid gap-1.5">
                                            <Label className="text-xs">{t('library.untilDate')}</Label>
                                            <Input type="date" className="h-8 text-xs" value={dateUntil} onChange={(e) => setDateUntil(e.target.value)} />
                                        </div>
                                    </div>
                                    {mode === 'semantic' && (
                                        <>
                                            <DropdownMenuSeparator />
                                            <DropdownMenuLabel>{t('library.semanticOptions')}</DropdownMenuLabel>
                                            <div className="p-2 space-y-2">
                                                <div className="flex items-center gap-2">
                                                    <Checkbox id="sem-enabled" checked={semantic} onCheckedChange={(c) => setSemantic(!!c)} />
                                                    <Label htmlFor="sem-enabled" className="text-xs">{t('library.enableVectorSearch')}</Label>
                                                </div>
                                                <div className="grid gap-1.5">
                                                    <Label className="text-xs">{t('library.topK')}: {topK}</Label>
                                                    <Input type="number" className="h-8 text-xs" value={topK} onChange={(e) => setTopK(parseInt(e.target.value)||100)} />
                                                </div>
                                                <Button size="sm" variant="ghost" className="w-full text-xs justify-start h-6 px-0" onClick={() => setShowAdvancedSemantic(!showAdvancedSemantic)}>
                                                    {showAdvancedSemantic ? t('library.hideExamples') : t('library.showExamples')}
                                                </Button>
                                                {showAdvancedSemantic && (
                                                    <textarea
                                                        className="w-full h-20 text-xs border rounded p-1 bg-background"
                                                        placeholder={t('library.jsonExamples')}
                                                        value={examplesText}
                                                        onChange={(e) => setExamplesText(e.target.value)}
                                                    />
                                                )}
                                            </div>
                                        </>
                                    )}
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </div>
                    </div>

                    {/* Table */}
                    <div className="flex-1 overflow-auto">
                        <Table>
                            <TableHeader className="sticky top-0 bg-background/95 backdrop-blur z-10 shadow-sm">
                                <TableRow>
                                    <TableHead className="w-[30px] pl-4">
                                        <Checkbox 
                                            checked={papers.length > 0 && papers.every(p => selectedPapers.includes(String(p.ID)))}
                                            onCheckedChange={handleSelectAll}
                                        />
                                    </TableHead>
                                    <TableHead className="w-[40%]">{t('library.table.title')}</TableHead>
                                    <TableHead className="w-[20%]">{t('library.table.authors')}</TableHead>
                                    <TableHead className="w-[10%]">{t('library.table.source')}</TableHead>
                                    <TableHead className="w-[12%]">{t('library.table.date')}</TableHead>
                                    {mode === 'semantic' && <TableHead className="w-[10%]">{t('library.table.similarity')}</TableHead>}
                                    <TableHead className="w-[8%] text-right pr-6">{t('library.table.actions')}</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {loading && papers.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={mode === 'semantic' ? 7 : 6} className="h-24 text-center">{t('library.loading')}</TableCell>
                                    </TableRow>
                                ) : papers.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={mode === 'semantic' ? 7 : 6} className="h-24 text-center text-muted-foreground">{t('library.noPapers')}</TableCell>
                                    </TableRow>
                                ) : (
                                    papers.map((paper) => (
                                        <TableRow 
                                            key={paper.ID} 
                                            className={`group hover:bg-muted/50 transition-colors cursor-pointer ${selectedPapers.includes(String(paper.ID)) ? "bg-muted/30" : ""}`}
                                            onClick={() => setSelectedPaper(paper)}
                                        >
                                            <TableCell className="pl-4 py-3" onClick={(e) => e.stopPropagation()}>
                                                <Checkbox 
                                                    checked={selectedPapers.includes(String(paper.ID))}
                                                    onCheckedChange={() => handleSelectPaper(String(paper.ID))}
                                                />
                                            </TableCell>
                                            <TableCell className="py-3">
                                                <div className="font-medium text-base leading-tight mb-1 line-clamp-2 group-hover:text-primary transition-colors" title={paper.Title}>
                                                    {paper.Title}
                                                </div>
                                            </TableCell>
                                            <TableCell className="py-3">
                                                <div className="text-sm text-muted-foreground line-clamp-1" title={paper.Authors?.join(', ')}>
                                                    {paper.Authors?.slice(0, 2).join(', ')}
                                                    {paper.Authors?.length > 2 && ` +${paper.Authors.length - 2}`}
                                                </div>
                                            </TableCell>
                                            <TableCell className="py-3">
                                                <Badge variant="secondary" className="uppercase text-[10px] tracking-wider font-semibold bg-secondary/50">
                                                    {paper.Source}
                                                </Badge>
                                            </TableCell>
                                            <TableCell className="py-3 text-sm text-muted-foreground whitespace-nowrap">
                                                <div className="flex items-center gap-1">
                                                    <Calendar className="w-3 h-3 opacity-70" />
                                                    {formatDate(paper.FirstAnnouncedAt)}
                                                </div>
                                            </TableCell>
                                            {mode === 'semantic' && (
                                                <TableCell className="py-3">
                                                    <div className={`flex items-center gap-1 font-medium text-xs ${getSimilarityColor(paper.Similarity)}`}>
                                                        <TrendingUp className="w-3 h-3" />
                                                        {((paper.Similarity || 0) * 100).toFixed(1)}%
                                                    </div>
                                                </TableCell>
                                            )}
                                            <TableCell className="text-right pr-6 py-3">
                                                <Button 
                                                    variant="ghost" 
                                                    size="icon" 
                                                    className="h-8 w-8 text-muted-foreground hover:text-primary hover:bg-primary/10 transition-colors"
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        openPaper(paper.URL);
                                                    }}
                                                    title="Open Link"
                                                >
                                                    <ExternalLink className="w-4 h-4" />
                                                </Button>
                                            </TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </TableBody>
                        </Table>
                    </div>


                    {mode === 'basic' && (
                        <div className="p-4 border-t border-border/30 bg-card/10 flex items-center justify-between">
                            <div className="text-sm text-muted-foreground">
                                {t('library.showing', { from: (page - 1) * pageSize + 1, to: Math.min(page * pageSize, total), total })}
                            </div>
                            <div className="flex items-center gap-2">
                                <Button 
                                    variant="outline" 
                                    size="sm" 
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    disabled={page === 1 || loading}
                                    className="h-8 w-8 p-0"
                                >
                                    <ChevronLeft className="w-4 h-4" />
                                </Button>
                                <span className="text-sm font-medium min-w-[3rem] text-center">
                                    {page} / {Math.ceil(total / pageSize) || 1}
                                </span>
                                <Button 
                                    variant="outline" 
                                    size="sm" 
                                    onClick={() => setPage(p => Math.min(Math.ceil(total / pageSize), p + 1))}
                                    disabled={page >= Math.ceil(total / pageSize) || loading}
                                    className="h-8 w-8 p-0"
                                >
                                    <ChevronRight className="w-4 h-4" />
                                </Button>
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Paper Detail Sheet */}
            <Sheet open={!!selectedPaper} onOpenChange={(open) => !open && setSelectedPaper(null)}>
                <SheetContent className="w-[600px] sm:w-[540px] overflow-y-auto">
                    {selectedPaper && (
                        <>
                            <SheetHeader className="mb-6">
                                <div className="flex items-center gap-2 mb-2">
                                    <Badge variant="secondary" className="uppercase text-xs font-bold tracking-wider">
                                        {selectedPaper.Source}
                                    </Badge>
                                    <span className="text-sm text-muted-foreground">
                                        {formatDate(selectedPaper.FirstAnnouncedAt)}
                                    </span>
                                </div>
                                <SheetTitle className="text-2xl font-display leading-tight">
                                    {selectedPaper.Title}
                                </SheetTitle>
                            </SheetHeader>
                            
                            <div className="space-y-6">
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-2 uppercase tracking-wider">{t('library.authors')}</h4>
                                    <div className="flex flex-wrap gap-2">
                                        {selectedPaper.Authors?.map((author, i) => (
                                            <Badge key={i} variant="outline" className="font-normal">
                                                {author}
                                            </Badge>
                                        ))}
                                    </div>
                                </div>

                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-2 uppercase tracking-wider">{t('library.abstract')}</h4>
                                    <p className="text-sm leading-relaxed text-foreground/90 text-justify">
                                        {selectedPaper.Abstract}
                                    </p>
                                </div>

                                {selectedPaper.SourceID && (
                                    <div>
                                        <h4 className="text-sm font-medium text-muted-foreground mb-2 uppercase tracking-wider">{t('library.id')}</h4>
                                        <code className="text-xs bg-muted px-2 py-1 rounded">
                                            {selectedPaper.SourceID}
                                        </code>
                                    </div>
                                )}

                                <Separator />

                                <div className="flex gap-3">
                                    <Button onClick={() => openPaper(selectedPaper.URL)} className="flex-1">
                                        <ExternalLink className="w-4 h-4 mr-2" />
                                        {t('library.readFullPaper')}
                                    </Button>
                                    {/* Additional actions could go here */}
                                </div>
                            </div>
                        </>
                    )}
                </SheetContent>
            </Sheet>

            {/* Export Dialog */}
            <AlertDialog open={exportOpen} onOpenChange={setExportOpen}>
                <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('library.exportDialog.title')}</AlertDialogTitle>
                    <AlertDialogDescription>
                    {t('library.exportDialog.description')}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <div className="space-y-4">
                    <div className="space-y-2">
                    <Label className="text-sm font-medium">{t('export.format')}</Label>
                    <Select value={exportFormat} onValueChange={(v:any)=>setExportFormat(v)}>
                        <SelectTrigger>
                        <SelectValue placeholder="Select Format" />
                        </SelectTrigger>
                        <SelectContent>
                        <SelectItem value="csv">CSV</SelectItem>
                        <SelectItem value="json">JSON</SelectItem>
                        <SelectItem value="zotero">Zotero</SelectItem>
                        <SelectItem value="feishu">Feishu</SelectItem>
                        </SelectContent>
                    </Select>
                    </div>

                    {(exportFormat==='csv'||exportFormat==='json') && (
                    <div className="space-y-2">
                        <Label>{t('export.outputPath')}</Label>
                        <Input value={exportOutput} onChange={(e)=>setExportOutput(e.target.value)} placeholder="out/papers.csv or papers.json" />
                    </div>
                    )}
                    {exportFormat==='zotero' && (
                    <div className="space-y-2">
                        <Label>{t('export.collectionKey')}</Label>
                        <Input value={exportCollection} onChange={(e)=>setExportCollection(e.target.value)} placeholder="e.g., ABC123XY" />
                    </div>
                    )}
                    {exportFormat==='feishu' && (
                    <div className="space-y-2">
                        <Label>{t('export.feishuName')}</Label>
                        <Input value={exportFeishuName} onChange={(e)=>setExportFeishuName(e.target.value)} placeholder="e.g., Papers Dataset" />
                    </div>
                    )}
                </div>
                <AlertDialogFooter>
                    <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                    <AlertDialogAction onClick={confirmExport}>{t('common.confirm')}</AlertDialogAction>
                </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
};

export default LibraryView;
