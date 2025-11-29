import React, { createContext, useContext, useState, ReactNode } from 'react';

// 定义爬取参数类型（与 SearchView.tsx 中的一致）
export interface CrawlParams {
  platform: 'arxiv' | 'acl' | 'openreview' | 'ssrn';
  keywords: string[];
  categories: string[];
  dateFrom: string;
  dateUntil: string;
  limit: number;
  update: boolean;
  useAPI: boolean;
  venueId: string;
  useRSS: boolean;
  useBibTeX: boolean;
}

// 定义 Context 的数据结构
interface CrawlContextType {
  // 当前任务状态
  currentTaskId: string | null;
  isCrawling: boolean;
  
  // 爬取参数状态
  crawlParams: CrawlParams;
  keywordInput: string;
  categoryInput: string;
  
  // 更新方法
  setCurrentTaskId: (id: string | null) => void;
  setIsCrawling: (isCrawling: boolean) => void;
  setCrawlParams: (params: CrawlParams | ((prev: CrawlParams) => CrawlParams)) => void;
  setKeywordInput: (input: string) => void;
  setCategoryInput: (input: string) => void;
}

// 默认初始状态
const initialCrawlParams: CrawlParams = {
  platform: 'arxiv',
  keywords: [],
  categories: [],
  dateFrom: '',
  dateUntil: '',
  limit: 100,
  update: false,
  useAPI: false,
  venueId: '',
  useRSS: true,
  useBibTeX: false
};

const CrawlContext = createContext<CrawlContextType | undefined>(undefined);

export const CrawlProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [currentTaskId, setCurrentTaskId] = useState<string | null>(null);
  const [isCrawling, setIsCrawling] = useState(false);
  
  const [crawlParams, setCrawlParams] = useState<CrawlParams>(initialCrawlParams);
  const [keywordInput, setKeywordInput] = useState('');
  const [categoryInput, setCategoryInput] = useState('');

  return (
    <CrawlContext.Provider value={{
      currentTaskId,
      isCrawling,
      crawlParams,
      keywordInput,
      categoryInput,
      setCurrentTaskId,
      setIsCrawling,
      setCrawlParams,
      setKeywordInput,
      setCategoryInput
    }}>
      {children}
    </CrawlContext.Provider>
  );
};

export const useCrawlContext = () => {
  const context = useContext(CrawlContext);
  if (context === undefined) {
    throw new Error('useCrawlContext must be used within a CrawlProvider');
  }
  return context;
};

