import React, { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { ScrollArea } from './ui/scroll-area';
import { Badge } from './ui/badge';
import { Separator } from './ui/separator';
import { Checkbox } from './ui/checkbox';
import TerminalBoxLineIcon from 'remixicon-react/TerminalBoxLineIcon';
import DownloadLineIcon from 'remixicon-react/DownloadLineIcon';
import DatabaseLineIcon from 'remixicon-react/DatabaseLineIcon';
import CheckLineIcon from 'remixicon-react/CheckLineIcon';
import ErrorWarningLineIcon from 'remixicon-react/ErrorWarningLineIcon';
import InformationLineIcon from 'remixicon-react/InformationLineIcon';
import SearchLineIcon from 'remixicon-react/SearchLineIcon';
import DeleteBinLineIcon from 'remixicon-react/DeleteBinLineIcon';
import ArrowDownLineIcon from 'remixicon-react/ArrowDownLineIcon';
import ArrowUpLineIcon from 'remixicon-react/ArrowUpLineIcon';

interface LogEntry {
  id: string;
  timestamp: string;
  level: 'info' | 'success' | 'warning' | 'error' | 'debug';
  message: string;
  platform?: string;
  count?: number;
}

interface CrawlLogViewerProps {
  isVisible: boolean;
  onClose: () => void;
  logs: LogEntry[];
  isCrawling: boolean;
  totalCount: number;
  currentPlatform: string;
}

const CrawlLogViewer: React.FC<CrawlLogViewerProps> = ({
  isVisible,
  onClose,
  logs,
  isCrawling,
  totalCount,
  currentPlatform
}) => {
  const [levelFilter, setLevelFilter] = useState<Set<string>>(new Set(['info', 'success', 'warning', 'error']));
  const [autoScroll, setAutoScroll] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const [filteredLogs, setFilteredLogs] = useState<LogEntry[]>([]);

  // 过滤日志
  useEffect(() => {
    let filtered = logs;

    // 按级别过滤
    if (levelFilter.size > 0) {
      filtered = filtered.filter(log => levelFilter.has(log.level));
    }

    // 按搜索词过滤
    if (searchTerm.trim()) {
      const term = searchTerm.toLowerCase();
      filtered = filtered.filter(log => 
        log.message.toLowerCase().includes(term) ||
        log.platform?.toLowerCase().includes(term)
      );
    }

    setFilteredLogs(filtered);
  }, [logs, levelFilter, searchTerm]);

  // 自动滚动到底部
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

  const clearLogs = () => {
    // 这里需要从父组件传递清空函数
    console.log('Clear logs requested');
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

  if (!isVisible) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <Card className="w-full max-w-6xl h-[80vh] flex flex-col">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <TerminalBoxLineIcon className="w-5 h-5 text-primary" />
                </div>
                <div>
                  <CardTitle className="text-xl font-display font-semibold">
                    爬取日志 - {currentPlatform}
                  </CardTitle>
                  <CardDescription className="text-sm text-muted-foreground">
                    {isCrawling ? '正在爬取中...' : '爬取已完成'} | 总计: {totalCount} 篇论文
                  </CardDescription>
                </div>
              </div>
            </div>
            
            <div className="flex items-center gap-2">
              <Button
                onClick={onClose}
                size="sm"
                variant="outline"
                className="hover-lift"
              >
                关闭
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
                <Label className="text-sm font-medium">级别:</Label>
                {['info', 'success', 'warning', 'error', 'debug'].map(level => (
                  <label key={level} className="flex items-center gap-1 text-sm cursor-pointer">
                    <Checkbox
                      checked={levelFilter.has(level)}
                      onCheckedChange={() => toggleLevelFilter(level)}
                    />
                    <span className="capitalize">{level}</span>
                  </label>
                ))}
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
                
                <Button
                  onClick={clearLogs}
                  size="sm"
                  variant="outline"
                  className="text-destructive hover:text-destructive"
                >
                  <DeleteBinLineIcon className="w-4 h-4 mr-1" />
                  清空
                </Button>
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
                      <div className="flex-shrink-0 mt-0.5">
                        {getLevelIcon(log.level)}
                      </div>
                      
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-xs font-mono text-muted-foreground">
                            {log.timestamp}
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
              {isCrawling && (
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

export default CrawlLogViewer;
