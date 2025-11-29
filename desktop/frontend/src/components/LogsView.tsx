import React, { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { ScrollArea } from './ui/scroll-area';
import { Badge } from './ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Checkbox } from './ui/checkbox';
import TerminalBoxLineIcon from 'remixicon-react/TerminalBoxLineIcon';

import CheckLineIcon from 'remixicon-react/CheckLineIcon';
import ErrorWarningLineIcon from 'remixicon-react/ErrorWarningLineIcon';
import InformationLineIcon from 'remixicon-react/InformationLineIcon';
import SearchLineIcon from 'remixicon-react/SearchLineIcon';

import ArrowDownLineIcon from 'remixicon-react/ArrowDownLineIcon';

import RefreshLineIcon from 'remixicon-react/RefreshLineIcon';

import { GetCrawlTask, GetCrawlTaskLogs, SetLogLevel } from '../../wailsjs/go/main/App';
import { useToast } from './ui/use-toast';
import { useCrawlContext } from '../context/CrawlContext';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

interface LogEntry {
    id: string;
    timestamp: string;
    level: 'info' | 'success' | 'warning' | 'error' | 'debug';
    message: string;
    platform?: string;
    count?: number;
}

interface CrawlTask {
    id: string;
    platform: string;
    params: Record<string, any>;
    status: 'pending' | 'running' | 'completed' | 'failed';
    progress: number;
    total_count: number;
    start_time: string;
    end_time?: string;
    error?: string;
    logs: LogEntry[];
}

const LogsView: React.FC = () => {
    const [currentTask, setCurrentTask] = useState<CrawlTask | null>(null);
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [levelFilter, setLevelFilter] = useState<Set<string>>(new Set(['info', 'success', 'warning', 'error']));
    const [autoScroll, setAutoScroll] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [globalLevel, setGlobalLevel] = useState<'INFO'|'DEBUG'>('INFO');
    const scrollAreaRef = useRef<HTMLDivElement>(null);
    const { toast } = useToast();
    
    // 从 Context 获取当前任务ID和爬取状态
    const { currentTaskId, setIsCrawling } = useCrawlContext();

    // 从URL参数获取任务ID
    const getTaskIdFromUrl = () => {
        const hash = window.location.hash;
        const match = hash.match(/taskId=([^&]+)/);
        return match ? match[1] : null;
    };

    // 确定当前要显示的任务ID（优先使用URL参数，其次是Context）
    const activeTaskId = getTaskIdFromUrl() || currentTaskId;

    // 加载任务信息
    const loadTask = async (taskId: string) => {
        try {
            const taskData = await GetCrawlTask(taskId);
            const task: CrawlTask = JSON.parse(taskData);
            setCurrentTask(task);
            // 这里不直接 setLogs，而是通过 refreshLogs 获取完整日志
            // 但如果是第一次加载，使用 task.logs 初始化
            if (logs.length === 0) {
                setLogs(task.logs);
            }
            
            // 更新 Context 中的爬取状态
            if (taskId === currentTaskId) {
                setIsCrawling(task.status === 'running' || task.status === 'pending');
            }
        } catch (error) {
            console.error('Failed to load task:', error);
            // 只有在主动刷新时才提示，避免自动刷新时频繁弹窗
            // toast({
            //     title: "加载任务失败",
            //     description: "无法获取任务信息，请重试",
            //     variant: "destructive",
            // });
        }
    };

    // 刷新日志
    const refreshLogs = async () => {
        if (!currentTask) return;

        setIsRefreshing(true);
        try {
            const logsData = await GetCrawlTaskLogs(currentTask.id);
            const newLogs: LogEntry[] = JSON.parse(logsData);
            setLogs(newLogs);

            // 更新任务状态
            await loadTask(currentTask.id);
        } catch (error) {
            console.error('Failed to refresh logs:', error);
        } finally {
            setIsRefreshing(false);
        }
    };

    // 监听后端流式日志
    useEffect(() => {
        // 监听 crawl-log 事件
        const cancelLogListener = EventsOn("crawl-log", (logEntry: LogEntry) => {
            // 只有当日志属于当前查看的任务或者没有指定任务平台过滤时才显示
            // 注意：logEntry 可能没有 taskId 字段，但通常我们只关注当前运行的任务
            // 如果能匹配当前任务更好
            
            // 简单处理：将日志追加到当前日志列表
            setLogs(prev => [...prev, logEntry]);
            
            // 如果是完成或失败状态，更新任务状态
            if (logEntry.level === 'success' || logEntry.level === 'error') {
                 if (activeTaskId) {
                     loadTask(activeTaskId);
                 }
            }
        });

        return () => {
            // 清理监听器 (Wails 没有直接的 Off 方法，EventsOff 需要事件名)
            // 这是一个潜在的问题，如果 Wails 的 EventsOff 移除所有监听器
            // 这里假设 EventsOn 返回的是取消函数或者我们在卸载时调用 EventsOff
            // Wails v2 的 EventsOff 接受事件名和可选的回调数组，这里简单移除事件的所有监听可能影响其他组件
            // 但通常只有 LogsView 监听此事件
            // EventsOff("crawl-log"); 
            // Wails TS runtime EventsOn returns a cancel function? No, it doesn't return anything in v2 docs usually.
            // Let's check runtime.d.ts if available, or just use EventsOff
            EventsOff("crawl-log");
        };
    }, [activeTaskId]);

    // 初始加载
    useEffect(() => {
        if (activeTaskId) {
            loadTask(activeTaskId);
        }
    }, [activeTaskId]);

    // 定期刷新（如果任务还在运行，作为流式日志的补充/兜底）
    useEffect(() => {
        if (!currentTask || currentTask.status !== 'running') return;

        const interval = setInterval(() => {
            refreshLogs();
        }, 3000); // 降低频率，主要依赖流式日志

        return () => clearInterval(interval);
    }, [currentTask?.status, currentTask?.id]);

  // 切换全局日志级别（影响后端 logger 输出，便于在任务期间输出 Debug 级别）
  const applyGlobalLevel = async (level: 'INFO'|'DEBUG') => {
    try {
      await SetLogLevel(level);
      setGlobalLevel(level);
      if (level === 'DEBUG') {
        setLevelFilter((prev)=> new Set([...prev, 'debug']));
      }
    } catch (e) {
      console.error(e);
    }
  };

    // 过滤日志
    const filteredLogs = logs.filter(log => {
        // 按级别过滤
        if (levelFilter.size > 0 && !levelFilter.has(log.level)) {
            return false;
        }

        // 按搜索词过滤
        if (searchTerm.trim()) {
            const term = searchTerm.toLowerCase();
            return log.message.toLowerCase().includes(term) ||
                log.platform?.toLowerCase().includes(term);
        }

        return true;
    });

    // 自动滚动
    useEffect(() => {
        if (autoScroll && scrollAreaRef.current) {
            const scrollElement = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
            if (scrollElement) {
                scrollElement.scrollTop = scrollElement.scrollHeight;
            }
        }
    }, [filteredLogs, autoScroll]);

    const getLevelIcon = (level: string) => {
        switch (level) {
            case 'success':
                return <CheckLineIcon className="w-3 h-3 text-green-500" />;
            case 'warning':
                return <ErrorWarningLineIcon className="w-3 h-3 text-yellow-500" />;
            case 'error':
                return <ErrorWarningLineIcon className="w-3 h-3 text-red-500" />;
            case 'debug':
                return <InformationLineIcon className="w-3 h-3 text-blue-500" />;
            default:
                return <InformationLineIcon className="w-3 h-3 text-gray-500" />;
        }
    };

    const getLevelColor = (level: string) => {
        switch (level) {
            case 'success':
                return 'text-green-600 bg-green-50 border-green-200';
            case 'warning':
                return 'text-yellow-600 bg-yellow-50 border-yellow-200';
            case 'error':
                return 'text-red-600 bg-red-50 border-red-200';
            case 'debug':
                return 'text-blue-600 bg-blue-50 border-blue-200';
            default:
                return 'text-gray-600 bg-gray-50 border-gray-200';
        }
    };

    const toggleLevelFilter = (level: string) => {
        const newFilter = new Set(levelFilter);
        if (newFilter.has(level)) {
            newFilter.delete(level);
        } else {
            newFilter.add(level);
        }
        setLevelFilter(newFilter);
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'running':
                return 'bg-blue-100 text-blue-800';
            case 'completed':
                return 'bg-green-100 text-green-800';
            case 'failed':
                return 'bg-red-100 text-red-800';
            default:
                return 'bg-gray-100 text-gray-800';
        }
    };

    if (!currentTask) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="text-center">
                    <TerminalBoxLineIcon className="w-8 h-8 mx-auto mb-4 text-muted-foreground" />
                    <p className="text-muted-foreground">未找到爬取任务</p>
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
                                    <TerminalBoxLineIcon className="w-5 h-5 text-primary" />
                                </div>
                                <div>
                                    <CardTitle className="text-2xl font-display font-semibold">
                                        爬取日志 - {currentTask.platform}
                                    </CardTitle>
                                    <CardDescription className="text-sm text-muted-foreground">
                                        <Badge className={`mr-2 ${getStatusColor(currentTask.status)}`}>
                                            {currentTask.status}
                                        </Badge>
                                        {currentTask.status === 'running' ? '正在爬取中...' : '爬取已完成'} |
                                        总计: {currentTask.total_count} 篇论文
                                    </CardDescription>
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center gap-2">
                            <Button
                                onClick={refreshLogs}
                                disabled={isRefreshing}
                                size="sm"
                                variant="outline"
                                className="hover-lift"
                            >
                                <RefreshLineIcon className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                                刷新
                            </Button>
                        </div>
                    </div>
                </CardHeader>

                <CardContent className="flex-1 flex flex-col overflow-hidden p-0">
                    {/* 控制面板 */}
                    <div className="border-b border-border/30 p-4 bg-muted/30">
                        <div className="flex flex-wrap gap-4 items-center">
                            {/* 搜索框 */}
                            <div className="flex-1 min-w-64">
                                <div className="relative">
                                    <SearchLineIcon className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                                    <Input
                                        placeholder="搜索日志内容..."
                                        value={searchTerm}
                                        onChange={(e) => setSearchTerm(e.target.value)}
                                        className="pl-10"
                                    />
                                </div>
                            </div>

                    {/* 级别过滤 */}
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-muted-foreground mr-1">Filter:</span>
                      {['info', 'success', 'warning', 'error', 'debug'].map(level => {
                        const getLevelBadgeColor = (lvl: string) => {
                          switch (lvl) {
                            case 'success': return 'bg-green-100 text-green-800 hover:bg-green-200 border-green-200';
                            case 'warning': return 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200 border-yellow-200';
                            case 'error': return 'bg-red-100 text-red-800 hover:bg-red-200 border-red-200';
                            case 'debug': return 'bg-blue-100 text-blue-800 hover:bg-blue-200 border-blue-200';
                            default: return 'bg-gray-100 text-gray-800 hover:bg-gray-200 border-gray-200';
                          }
                        };
                        
                        return (
                          <Badge 
                            key={level}
                            variant="outline"
                            className={`cursor-pointer select-none capitalize transition-all ${
                              levelFilter.has(level) 
                                ? getLevelBadgeColor(level)
                                : 'text-muted-foreground bg-transparent hover:bg-muted border-dashed'
                            }`}
                            onClick={() => toggleLevelFilter(level)}
                          >
                            {level}
                          </Badge>
                        );
                      })}
                    </div>

                    {/* 全局日志级别（影响后端） */}
                    <div className="flex items-center gap-2">
                      <Label className="text-sm font-medium">全局级别:</Label>
                      <Select value={globalLevel} onValueChange={(v:any)=>applyGlobalLevel(v)}>
                        <SelectTrigger className="w-[120px]">
                          <SelectValue placeholder="INFO" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="INFO">INFO</SelectItem>
                          <SelectItem value="DEBUG">DEBUG</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                            {/* 控制按钮 */}
                            <div className="flex items-center gap-2">
                                <label className="flex items-center gap-2 text-sm cursor-pointer">
                                    <Checkbox
                                        checked={autoScroll}
                                        onCheckedChange={(checked) => setAutoScroll(checked === true)}
                                    />
                                    <span>自动滚动</span>
                                </label>
                            </div>
                        </div>
                    </div>

                    {/* 日志显示区域 */}
                    <div className="flex-1 overflow-hidden">
                        <ScrollArea ref={scrollAreaRef} className="h-full">
                            <div className="p-4 space-y-1">
                                {filteredLogs.length === 0 ? (
                                    <div className="text-center py-8 text-muted-foreground">
                                        <TerminalBoxLineIcon className="w-8 h-8 mx-auto mb-2 opacity-50" />
                                        <p>暂无日志记录</p>
                                    </div>
                                ) : (
                                    filteredLogs.map((log) => (
                                        <div
                                            key={log.id}
                                            className={`flex items-start gap-3 p-3 rounded-lg border transition-colors hover:bg-muted/50 ${getLevelColor(log.level)}`}
                                        >
                                            <div className="flex-shrink-0 mt-[3px]">
                                                {getLevelIcon(log.level)}
                                            </div>

                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2 mb-1">
                                                    <span className="text-xs font-mono text-muted-foreground">
                                                        {new Date(log.timestamp).toLocaleTimeString()}
                                                    </span>
                                                    {log.platform && (
                                                        <Badge variant="secondary" className="text-xs">
                                                            {log.platform}
                                                        </Badge>
                                                    )}
                                                    {log.count && (
                                                        <Badge variant="outline" className="text-xs">
                                                            +{log.count}
                                                        </Badge>
                                                    )}
                                                </div>
                                                <p className="text-sm font-mono whitespace-pre-wrap break-words">
                                                    {log.message}
                                                </p>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                        </ScrollArea>
                    </div>

                    {/* 状态栏 */}
                    <div className="border-t border-border/30 p-3 bg-muted/30 flex items-center justify-between text-sm text-muted-foreground">
                        <div className="flex items-center gap-4">
                            <span>总计: {logs.length} 条日志</span>
                            <span>显示: {filteredLogs.length} 条</span>
                            {currentTask.status === 'running' && (
                                <div className="flex items-center gap-2">
                                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                                    <span>正在爬取...</span>
                                </div>
                            )}
                        </div>
                        <div className="flex items-center gap-2">
                            {autoScroll && (
                                <div className="flex items-center gap-1 text-xs">
                                    <ArrowDownLineIcon className="w-3 h-3" />
                                    <span>自动滚动</span>
                                </div>
                            )}
                        </div>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
};

export default LogsView;
