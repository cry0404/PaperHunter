import React, { createContext, useContext, useState, useEffect, useRef, ReactNode } from 'react';
import { useToast } from '../components/ui/use-toast';
import { GetDailyRecommendations } from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import * as models from '../../wailsjs/go/models';

// 定义类型
export interface Paper {
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

export interface RecommendationGroup {
  seedPaper: {
    title: string;
    authors: string[];
    abstract: string;
    source?: string; // 种子论文来源（arXiv, user_interest, zotero等）
    sourceId?: string;
  };
  papers: Paper[];
}

export interface AgentLogEntry {
  type: 'user' | 'assistant' | 'tool_call' | 'tool_result' | 'error';
  content: string;
  timestamp: string;
}

interface RecommendResult {
  crawledToday: boolean;
  arxivCrawlCount: number;
  seedPaperCount: number;
  recommendations: RecommendationGroup[];
  message: string;
  agentLogs?: AgentLogEntry[];
}

interface RecommendContextType {
  // State
  loading: boolean;
  recommendations: RecommendationGroup[];
  mergedPapers: Paper[];
  agentLogs: AgentLogEntry[];
  showRecommendations: boolean;
  interestQuery: string;
  dateFrom: string;
  dateTo: string;
  localFilePath: string;
  useLocalFile: boolean;
  selectedPapers: Set<string>;

  // Setters
  setInterestQuery: (q: string) => void;
  setDateFrom: (d: string) => void;
  setDateTo: (d: string) => void;
  setLocalFilePath: (path: string) => void;
  setUseLocalFile: (use: boolean) => void;
  setShowRecommendations: (show: boolean) => void;
  togglePaperSelection: (id: string) => void;
  selectAllPapers: () => void;
  clearSelection: () => void;

  // Actions
  startRecommend: () => Promise<void>;
  clearResults: () => void;
}

const RecommendContext = createContext<RecommendContextType | undefined>(undefined);

export const useRecommendContext = () => {
  const context = useContext(RecommendContext);
  if (!context) {
    throw new Error('useRecommendContext must be used within a RecommendProvider');
  }
  return context;
};

export const RecommendProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [loading, setLoading] = useState(false);
  const [recommendations, setRecommendations] = useState<RecommendationGroup[]>([]);
  const [mergedPapers, setMergedPapers] = useState<Paper[]>([]);
  const [agentLogs, setAgentLogs] = useState<AgentLogEntry[]>([]);
  const [showRecommendations, setShowRecommendations] = useState(false);
  const [interestQuery, setInterestQuery] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [localFilePath, setLocalFilePath] = useState('');
  const [useLocalFile, setUseLocalFile] = useState(false);
  const [selectedPapers, setSelectedPapers] = useState<Set<string>>(new Set());
  
  const { toast } = useToast();

  // 监听后端流式日志
  useEffect(() => {
    const cancelLogListener = EventsOn("agent-log", (logEntry: AgentLogEntry) => {
      setAgentLogs(prev => [...prev, logEntry]);
    });

    return () => {
      EventsOff("agent-log");
    };
  }, []);

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

  const clearResults = () => {
    setRecommendations([]);
    setMergedPapers([]);
    setAgentLogs([]);
    setShowRecommendations(false);
    clearSelection();
  };

  const startRecommend = async () => {
    setLoading(true);
    // 清空之前的日志和结果
    setAgentLogs([]);
    setRecommendations([]);
    setMergedPapers([]);
    
    // 可以在这里跳转到日志页面，或者只在后台运行
    // window.location.hash = '#/logs'; // 可选：自动跳转到日志页查看进度

    try {
      const resultJson = await GetDailyRecommendations({
        interestQuery: interestQuery.trim() || '',
        platforms: ['arxiv'], // 专注于arXiv平台
        zoteroCollection: '',
        topK: 5,
        maxRecommendations: 20,
        forceCrawl: false,
        dateFrom: dateFrom.trim() || '',
        dateTo: dateTo.trim() || '',
        localFilePath: useLocalFile ? localFilePath : '',
        localFileAction: useLocalFile ? 'import_for_recommend' : '',
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
      
      // 转换数据格式
      const transformedRecommendations: RecommendationGroup[] = result.recommendations
        .filter((group: any) => group && group.papers && Array.isArray(group.papers) && group.papers.length > 0)
        .map((group: any, groupIdx: number) => {
          const papers = group.papers
            .map((similarPaper: any, paperIdx: number) => {
              const paper = similarPaper.paper || similarPaper.Paper || similarPaper;
              const similarity = similarPaper.similarity || similarPaper.Similarity || 0;
              
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
            .filter((p: any) => p.title && p.title.trim() !== '');
          
          return {
            seedPaper: {
              title: group.seedPaper?.title || group.seedPaper?.Title || group.zoteroPaper?.title || group.zoteroPaper?.Title || '',
              authors: Array.isArray(group.seedPaper?.authors)
                ? group.seedPaper.authors
                : (Array.isArray(group.seedPaper?.Authors)
                  ? group.seedPaper.Authors
                  : (Array.isArray(group.zoteroPaper?.authors)
                    ? group.zoteroPaper.authors
                    : (Array.isArray(group.zoteroPaper?.Authors) ? group.zoteroPaper.Authors : []))),
              abstract: group.seedPaper?.abstract || group.seedPaper?.Abstract || group.zoteroPaper?.abstract || group.zoteroPaper?.Abstract || '',
              source: group.seedPaper?.source || group.seedPaper?.Source || 'unknown',
              sourceId: group.seedPaper?.sourceId || group.seedPaper?.SourceID || '',
            },
            papers: papers,
          };
        })
        .filter((group: RecommendationGroup) => group.papers.length > 0);

      setRecommendations(transformedRecommendations);
      
      // 合并去重
      const papersMap = new Map<string, Paper>();
      transformedRecommendations.forEach((group, groupIdx) => {
        group.papers.forEach((paper, paperIdx) => {
          const uniqueKey = `${paper.source}:${paper.sourceId}`;
          if (!papersMap.has(uniqueKey)) {
            const paperWithId = {
              ...paper,
              id: paper.id || `${groupIdx}-${paperIdx}-${paper.source}-${paper.sourceId}`,
            };
            papersMap.set(uniqueKey, paperWithId);
          } else {
            const existing = papersMap.get(uniqueKey)!;
            if (paper.similarity > existing.similarity) {
              papersMap.set(uniqueKey, {
                ...paper,
                id: existing.id,
              });
            }
          }
        });
      });
      
      const merged = Array.from(papersMap.values());
      merged.sort((a, b) => b.similarity - a.similarity);
      setMergedPapers(merged);
      
      if (merged.length > 0) {
        setShowRecommendations(true);
      }
      
      // 更新完整日志（虽然流式已经更新了，但为了确保一致性）
      if (result.agentLogs && result.agentLogs.length > 0) {
        setAgentLogs(result.agentLogs);
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

  return (
    <RecommendContext.Provider value={{
      loading,
      recommendations,
      mergedPapers,
      agentLogs,
      showRecommendations,
      interestQuery,
      dateFrom,
      dateTo,
      localFilePath,
      useLocalFile,
      setInterestQuery,
      setDateFrom,
      setDateTo,
      setLocalFilePath,
      setUseLocalFile,
      setShowRecommendations,
      selectedPapers,
      togglePaperSelection,
      selectAllPapers,
      clearSelection,
      startRecommend,
      clearResults
    }}>
      {children}
    </RecommendContext.Provider>
  );
};

