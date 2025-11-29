import React, { useState, useEffect, useRef, useMemo } from "react";
import { GetLogs, ClearLogs, SetLogLevel } from '../../wailsjs/go/main/App';
import { Button } from "./ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select";
import { Checkbox } from "./ui/checkbox";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { 
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "./ui/alert-dialog";
import { RefreshCw, Trash2, Activity } from "lucide-react";

interface LogEntry {
  timestamp: string;
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
  raw: string;
}

type FilterLevel = 'ALL' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

const LogViewer: React.FC = () => {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [autoScroll, setAutoScroll] = useState(true);
  const [filterLevel, setFilterLevel] = useState<FilterLevel>('ALL');
  const [currentLogLevel, setCurrentLogLevel] = useState('INFO');
  const [showClearDialog, setShowClearDialog] = useState(false);
  const logEndRef = useRef<HTMLDivElement>(null);
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  const parseLogLine = (line: string): LogEntry | null => {
    if (!line.trim()) return null;
    
    const regex = /^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}) \[(DEBUG|INFO|WARN|ERROR)\] (.+)$/;
    const match = line.match(regex);
    
    if (match) {
      return {
        timestamp: match[1],
        level: match[2] as LogEntry['level'],
        message: match[3],
        raw: line
      };
    }
    
    return null;
  };

  const loadLogs = async () => {
    try {
      setLoading(true);
      const content = await GetLogs();
      const lines = content.split('\n');
      const parsedLogs = lines
        .map(parseLogLine)
        .filter((log): log is LogEntry => log !== null);
      
      // 只有在日志内容真的变化时才更新
      setLogs(prevLogs => {
        if (JSON.stringify(prevLogs) !== JSON.stringify(parsedLogs)) {
          return parsedLogs;
        }
        return prevLogs;
      });
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleClearLogs = async () => {
    try {
      await ClearLogs();
      setLogs([]);
      setShowClearDialog(false);
    } catch (error) {
      console.error('Failed to clear logs:', error);
    }
  };

  const handleSetLogLevel = async (level: string) => {
    try {
      await SetLogLevel(level);
      setCurrentLogLevel(level);
    } catch (error) {
      console.error('Failed to set log level:', error);
    }
  };

  useEffect(() => {
    loadLogs(); 

    if (!autoRefresh) return;

    const timer = setInterval(() => {
      loadLogs();
    }, 2000);

    return () => clearInterval(timer);
  }, [autoRefresh]);

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'auto' });
    }
  }, [logs, autoScroll]);

  // 使用 useMemo 优化过滤逻辑
  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      if (filterLevel === 'ALL') return true;
      return log.level === filterLevel;
    });
  }, [logs, filterLevel]);

  const getLevelBadgeVariant = (level: string): "default" | "secondary" | "destructive" | "outline" => {
    switch (level) {
      case 'ERROR':
        return 'destructive';
      case 'WARN':
        return 'outline';
      case 'INFO':
        return 'default';
      case 'DEBUG':
        return 'secondary';
      default:
        return 'default';
    }
  };

  // 使用 useMemo 优化日志行渲染
  const logItems = useMemo(() => {
    return filteredLogs.map((log, index) => (
      <div
        key={`${log.timestamp}-${index}`}
        className="py-1.5 px-3 rounded hover:bg-accent/50 transition-colors"
      >
        <div className="flex items-start gap-3">
          <span className="text-muted-foreground text-xs whitespace-nowrap mt-0.5 font-mono">
            {log.timestamp}
          </span>
          <Badge variant={getLevelBadgeVariant(log.level)} className="mt-0.5">
            {log.level}
          </Badge>
          <span className="flex-1 break-all text-sm">
            {log.message}
          </span>
        </div>
      </div>
    ));
  }, [filteredLogs]);

  return (
    <>
      <div className="flex flex-col h-full overflow-hidden animate-fade-in">
        <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
          <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
            <div className="flex items-center justify-between">
              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                    <Activity className="w-5 h-5 text-primary" />
                  </div>
                  <CardTitle className="text-3xl font-display font-bold gradient-text">Activity Logs</CardTitle>
                </div>
                <CardDescription className="text-base text-muted-foreground ml-13">
                  实时查看应用日志和系统活动
                </CardDescription>
              </div>
              
              <div className="flex items-center gap-2">
                <Button
                  onClick={() => setShowClearDialog(true)}
                  size="sm"
                  variant="destructive"
                  className="hover-lift"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Clear
                </Button>
              </div>
            </div>

            <div className="flex items-center gap-6 mt-6 flex-wrap">
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors">
                  <Checkbox
                    checked={autoRefresh}
                    onCheckedChange={(checked: boolean) => setAutoRefresh(checked)}
                  />
                  <span>自动刷新(2s)</span>
                </label>

                <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-foreground transition-colors">
                  <Checkbox
                    checked={autoScroll}
                    onCheckedChange={(checked: boolean) => setAutoScroll(checked)}
                  />
                  <span>自动滚动</span>
                </label>
              </div>

              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">Log Level:</span>
                  <Select value={currentLogLevel} onValueChange={handleSetLogLevel}>
                    <SelectTrigger className="w-[120px] hover-lift">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="DEBUG">DEBUG</SelectItem>
                      <SelectItem value="INFO">INFO</SelectItem>
                      <SelectItem value="WARN">WARN</SelectItem>
                      <SelectItem value="ERROR">ERROR</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">Filter:</span>
                  <Select value={filterLevel} onValueChange={(value: string) => setFilterLevel(value as FilterLevel)}>
                    <SelectTrigger className="w-[120px] hover-lift">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ALL">ALL</SelectItem>
                      <SelectItem value="DEBUG">DEBUG</SelectItem>
                      <SelectItem value="INFO">INFO</SelectItem>
                      <SelectItem value="WARN">WARN</SelectItem>
                      <SelectItem value="ERROR">ERROR</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
          </CardHeader>

          <CardContent className="flex-1 p-0 overflow-hidden bg-background/50">
            <ScrollArea className="h-full" ref={scrollAreaRef}>
              <div className="p-6">
                {filteredLogs.length === 0 ? (
                  <div className="glass-card p-12 rounded-2xl text-center">
                    <div className="max-w-md mx-auto space-y-4">
                      <div className="w-20 h-20 rounded-2xl gradient-primary/10 flex items-center justify-center mx-auto">
                        <Activity className="w-10 h-10 text-primary" />
                      </div>
                      <h3 className="text-xl font-display font-semibold">No Logs Available</h3>
                      <p className="text-muted-foreground">
                        Logs will appear here when the application starts
                      </p>
                    </div>
                  </div>
                ) : (
                  <>
                    {logItems}
                    <div ref={logEndRef} />
                  </>
                )}
              </div>
            </ScrollArea>
          </CardContent>

          <div className="border-t border-border/30 px-8 py-4 bg-card/30 backdrop-blur-sm flex-shrink-0">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <span className="text-sm text-muted-foreground">
                  Total: <span className="font-semibold text-foreground">{filteredLogs.length}</span> logs
                </span>
                {filterLevel !== 'ALL' && (
                  <Badge variant="outline" className="text-xs">
                    Filter: {filterLevel}
                  </Badge>
                )}
              </div>
              {autoRefresh && (
                <div className="flex items-center gap-2 text-xs text-success">
                  <div className="w-2 h-2 bg-success rounded-full animate-pulse" />
                  <span>Auto-refreshing</span>
                </div>
              )}
            </div>
          </div>
        </Card>
      </div>

      {/* Clear Confirmation Dialog */}
      <AlertDialog open={showClearDialog} onOpenChange={setShowClearDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Clear all logs?</AlertDialogTitle>
            <AlertDialogDescription>
              这将会导致当前的所有日志丢失
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleClearLogs}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Clear Logs
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default LogViewer;
