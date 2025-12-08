import React, { useState, useEffect, useRef, useMemo } from "react";
import { GetLogs, ClearLogs, SetLogLevel } from '../../wailsjs/go/main/App';
import { Button } from "./ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select";
import { Checkbox } from "./ui/checkbox";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
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
import { Trash2, Activity, MessageSquare, Terminal, User, Bot, Wrench, AlertTriangle } from "lucide-react";
import { useRecommendContext } from "../context/RecommendContext";
import { useTranslation } from "react-i18next";

// --- System Log Types & Component ---

interface LogEntry {
  timestamp: string;
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
  raw: string;
}

type FilterLevel = 'ALL' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

const SystemLogsView: React.FC<{
  onClear: () => void;
}> = ({ onClear }) => {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [autoScroll, setAutoScroll] = useState(true);
  const [filterLevel, setFilterLevel] = useState<FilterLevel>('ALL');
  const [currentLogLevel, setCurrentLogLevel] = useState('INFO');
  const logEndRef = useRef<HTMLDivElement>(null);
  
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
      const content = await GetLogs();
      const lines = content.split('\n');
      const parsedLogs = lines
        .map(parseLogLine)
        .filter((log): log is LogEntry => log !== null);
      
      setLogs(prevLogs => {
        if (JSON.stringify(prevLogs) !== JSON.stringify(parsedLogs)) {
          return parsedLogs;
        }
        return prevLogs;
      });
    } catch (error) {
      console.error('Failed to load logs:', error);
    }
  };

  useEffect(() => {
    loadLogs(); 
    if (!autoRefresh) return;
    const timer = setInterval(loadLogs, 2000);
    return () => clearInterval(timer);
  }, [autoRefresh]);

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'auto' });
    }
  }, [logs, autoScroll]);

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      if (filterLevel === 'ALL') return true;
      return log.level === filterLevel;
    });
  }, [logs, filterLevel]);

  const handleSetLogLevel = async (level: string) => {
      try {
        await SetLogLevel(level);
        setCurrentLogLevel(level);
      } catch (error) {
        console.error('Failed to set log level:', error);
      }
  };

  const getLevelBadgeVariant = (level: string): "default" | "secondary" | "destructive" | "outline" => {
      switch (level) {
        case 'ERROR': return 'destructive';
        case 'WARN': return 'outline';
        case 'INFO': return 'default';
        case 'DEBUG': return 'secondary';
        default: return 'default';
      }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden bg-background">
       <div className="border-b border-border/30 px-6 py-3 bg-card/30 backdrop-blur-sm flex-shrink-0">
          <div className="flex items-center gap-6 flex-wrap">
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-primary transition-colors font-sans">
                  <Checkbox checked={autoRefresh} onCheckedChange={(c:boolean) => setAutoRefresh(c)} />
                  <span>{t('logs.autoRefresh')}</span>
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-primary transition-colors font-sans">
                  <Checkbox checked={autoScroll} onCheckedChange={(c:boolean) => setAutoScroll(c)} />
                  <span>{t('logs.autoScroll')}</span>
                </label>
              </div>

              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium font-sans">{t('logs.logLevel')}:</span>
                  <Select value={currentLogLevel} onValueChange={handleSetLogLevel}>
                    <SelectTrigger className="w-[100px] h-8 bg-background font-sans">
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
                  <span className="text-sm font-medium font-sans">{t('logs.filter')}:</span>
                  <Select value={filterLevel} onValueChange={(v) => setFilterLevel(v as FilterLevel)}>
                    <SelectTrigger className="w-[100px] h-8 bg-background font-sans">
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
       </div>

       <div className="flex-1 overflow-hidden bg-background/50">
          <ScrollArea className="h-full">
            <div className="p-6 space-y-1">
              {filteredLogs.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground font-serif">
                  <Terminal className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p>{t('logs.system.noLogs')}</p>
                </div>
              ) : (
                filteredLogs.map((log, index) => (
                  <div key={`${log.timestamp}-${index}`} className="py-1 px-3 rounded hover:bg-accent/50 transition-colors text-sm font-mono">
                    <div className="flex items-start gap-3">
                      <span className="text-muted-foreground text-xs whitespace-nowrap mt-0.5 opacity-70">
                        {log.timestamp}
                      </span>
                      <Badge variant={getLevelBadgeVariant(log.level)} className="mt-0.5 h-5 px-1 text-[10px] font-sans">
                        {log.level}
                      </Badge>
                      <span className="flex-1 break-all whitespace-pre-wrap font-mono text-foreground/90">
                        {log.message}
                      </span>
                    </div>
                  </div>
                ))
              )}
              <div ref={logEndRef} />
            </div>
          </ScrollArea>
       </div>
       
       <div className="border-t border-border/30 px-6 py-2 bg-card/30 text-xs text-muted-foreground flex justify-between items-center font-sans">
          <span>{t('logs.totalLogs', { count: filteredLogs.length })}</span>
          {autoRefresh && <span className="text-anthropic-green flex items-center gap-1"><div className="w-1.5 h-1.5 rounded-full bg-anthropic-green animate-pulse"/> {t('logs.live')}</span>}
       </div>
    </div>
  );
};

// --- Agent Log Component ---

const AgentLogsView: React.FC = () => {
  const { t } = useTranslation();
  const { agentLogs } = useRecommendContext();
  const [autoScroll, setAutoScroll] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
        const scrollElement = scrollRef.current.querySelector('[data-radix-scroll-area-viewport]');
        if (scrollElement) {
            scrollElement.scrollTop = scrollElement.scrollHeight;
        }
    }
  }, [agentLogs, autoScroll]);

  const getIcon = (type: string) => {
    switch (type) {
      case 'user': return <User className="w-4 h-4" />;
      case 'assistant': return <Bot className="w-4 h-4" />;
      case 'tool_call': return <Wrench className="w-4 h-4" />;
      case 'tool_result': return <Terminal className="w-4 h-4" />;
      case 'error': return <AlertTriangle className="w-4 h-4" />;
      default: return <MessageSquare className="w-4 h-4" />;
    }
  };

  const getStyle = (type: string) => {
    switch (type) {
      case 'user': return 'bg-primary/5 border-primary/20';
      case 'assistant': return 'bg-secondary/30 border-secondary';
      case 'tool_call': return 'bg-blue-500/5 border-blue-500/20';
      case 'tool_result': return 'bg-slate-500/5 border-slate-500/20 font-mono text-xs';
      case 'error': return 'bg-destructive/10 border-destructive/30 text-destructive';
      default: return 'bg-muted/30 border-border';
    }
  };

  const getLabel = (type: string) => {
      // Could translate these too if needed
      switch (type) {
          case 'user': return 'User';
          case 'assistant': return 'Assistant';
          case 'tool_call': return 'Tool Call';
          case 'tool_result': return 'Tool Result';
          case 'error': return 'Error';
          default: return type;
      }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden bg-background">
      <div className="border-b border-border/30 px-6 py-3 bg-card/30 backdrop-blur-sm flex-shrink-0 flex justify-between items-center">
         <div className="flex items-center gap-2">
            <Badge variant="outline" className="h-6 font-sans">
                {agentLogs.length} {t('logs.entries')}
            </Badge>
         </div>
         <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-primary transition-colors font-sans">
            <Checkbox checked={autoScroll} onCheckedChange={(c:boolean) => setAutoScroll(c)} />
            <span>{t('logs.autoScroll')}</span>
         </label>
      </div>

      <div className="flex-1 overflow-hidden bg-background/50">
         <ScrollArea className="h-full" ref={scrollRef}>
            <div className="p-6 space-y-4 max-w-4xl mx-auto">
               {agentLogs.length === 0 ? (
                 <div className="text-center py-12 text-muted-foreground font-serif">
                   <Bot className="w-8 h-8 mx-auto mb-2 opacity-50" />
                   <p>{t('logs.agent.noLogs')}</p>
                   <p className="text-xs mt-1">{t('logs.agent.start')}</p>
                 </div>
               ) : (
                 agentLogs.map((log, index) => (
                    <div key={index} className={`rounded-lg border p-4 ${getStyle(log.type)}`}>
                       <div className="flex items-center justify-between mb-2 border-b border-black/5 dark:border-white/5 pb-2">
                          <div className="flex items-center gap-2 font-semibold text-sm font-sans">
                             {getIcon(log.type)}
                             <span>{getLabel(log.type)}</span>
                          </div>
                          <span className="text-xs opacity-50 font-mono">{log.timestamp}</span>
                       </div>
                       <div className="whitespace-pre-wrap break-words text-sm leading-relaxed font-serif">
                          {log.content}
                       </div>
                    </div>
                 ))
               )}
            </div>
         </ScrollArea>
      </div>
    </div>
  );
};


// --- Main LogViewer Component ---

const LogViewer: React.FC = () => {
  const { t } = useTranslation();
  const [showClearDialog, setShowClearDialog] = useState(false);
  const [activeTab, setActiveTab] = useState('system');
  
  const handleClearLogs = async () => {
    try {
      await ClearLogs();
      setShowClearDialog(false);
      // Force refresh logic would be inside SystemLogsView, but ClearLogs clears the file
    } catch (error) {
      console.error('Failed to clear logs:', error);
    }
  };

  return (
    <>
      <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
        <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
          <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-6 flex-shrink-0">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div>
                    <CardTitle className="text-2xl font-sans font-medium tracking-tight">{t('logs.title')} & {t('logs.activity')}</CardTitle>
                    <CardDescription className="text-base text-muted-foreground font-serif">{t('logs.subtitle')}</CardDescription>
                </div>
              </div>
              
              {activeTab === 'system' && (
                <Button
                  onClick={() => setShowClearDialog(true)}
                  size="sm"
                  variant="destructive"
                  className="hover-lift font-sans"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t('logs.clearSystem')}
                </Button>
              )}
            </div>
          </CardHeader>

          <CardContent className="flex-1 p-0 overflow-hidden">
             <Tabs defaultValue="system" value={activeTab} onValueChange={setActiveTab} className="h-full flex flex-col">
                <div className="px-8 pt-2 bg-card/30 border-b border-border/30">
                    <TabsList className="bg-transparent p-0 h-auto gap-6 justify-start rounded-none">
                        <TabsTrigger 
                            value="system"
                            className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-2 pb-3 pt-2 font-sans font-medium"
                        >
                            <Terminal className="w-4 h-4 mr-2" />
                            {t('logs.systemLogs')}
                        </TabsTrigger>
                       
                    </TabsList>
                </div>
                
                <TabsContent value="system" className="flex-1 m-0 overflow-hidden data-[state=inactive]:hidden">
                    <SystemLogsView onClear={() => {}} />
                </TabsContent>
                
                <TabsContent value="agent" className="flex-1 m-0 overflow-hidden data-[state=inactive]:hidden">
                    <AgentLogsView />
                </TabsContent>
             </Tabs>
          </CardContent>
        </Card>
      </div>

      <AlertDialog open={showClearDialog} onOpenChange={setShowClearDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="font-sans">{t('logs.clearDialog.title')}</AlertDialogTitle>
            <AlertDialogDescription className="font-serif">
              {t('logs.clearDialog.desc')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="font-sans">{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleClearLogs}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90 font-sans"
            >
              {t('common.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default LogViewer;
