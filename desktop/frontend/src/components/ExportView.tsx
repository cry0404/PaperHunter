import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Separator } from './ui/separator';
import DownloadLineIcon from 'remixicon-react/DownloadLineIcon';
import BracesLineIcon from 'remixicon-react/BracesLineIcon';
import TableLineIcon from 'remixicon-react/TableLineIcon';
import DatabaseLineIcon from 'remixicon-react/DatabaseLineIcon';
import BookOpenLineIcon from 'remixicon-react/BookOpenLineIcon';
import { useToast } from './ui/use-toast';

const ExportView: React.FC = () => {
  const [format, setFormat] = useState<'csv' | 'json' | 'zotero' | 'feishu'>('csv');
  const [output, setOutput] = useState('');
  const [query, setQuery] = useState('');
  const [keywords, setKeywords] = useState<string>('');
  const [categories, setCategories] = useState<string>('');
  // 使用非空占位符值“all”，提交时再映射为空字符串
  const [source, setSource] = useState<'all' | 'arxiv' | 'openreview' | 'acl'>('all');
  const [collection, setCollection] = useState('');
  const [feishuName, setFeishuName] = useState('');
  const [limit, setLimit] = useState<number>(0);
  const [exporting, setExporting] = useState(false);
  const { toast } = useToast();

  const validate = (): boolean => {
    if ((format === 'csv' || format === 'json') && !output.trim()) {
      toast({ title: 'Invalid input', description: 'output is required for csv/json', variant: 'destructive' });
      return false;
    }
    if (format === 'feishu' && !feishuName.trim()) {
      toast({ title: 'Invalid input', description: 'feishuName is required for feishu export', variant: 'destructive' });
      return false;
    }
    // zotero：不强制 collection，允许为空
    return true;
  };

  const handleExport = async () => {
    if (!validate()) return;
    setExporting(true);
    try {
      const { ExportWithOptions } = await import('../../wailsjs/go/main/App');
      const kw = keywords.split(',').map(s => s.trim()).filter(Boolean);
      const cats = categories.split(',').map(s => s.trim()).filter(Boolean);
      const result = await ExportWithOptions({
        Format: format,
        Output: output,
        Query: query,
        Keywords: kw,
        Categories: cats,
        Source: source === 'all' ? '' : source,
        Collection: collection,
        FeishuName: feishuName,
        Limit: limit || 0,
      } as any);
      if (format === 'feishu' && result) {
        const { BrowserOpenURL } = await import('../../wailsjs/runtime/runtime');
        toast({
          title: '导出完成',
          description: (
            <div className="break-all">
              已上传到飞书: <a className="underline" onClick={()=>BrowserOpenURL(result)}>{result}</a>
            </div>
          )
        });
      } else if ((format==='csv'||format==='json') && result) {
        const { BrowserOpenURL } = await import('../../wailsjs/runtime/runtime');
        toast({
          title: '导出完成',
          description: (
            <div className="break-all">
              已保存到: <a className="underline" onClick={()=>BrowserOpenURL(`file://${result}`)}>{result}</a>
            </div>
          )
        });
      } else {
        toast({ title: '导出完成', description: '操作成功' });
      }
    } catch (error) {
      console.error('Export failed:', error);
      toast({ title: '导出失败', description: 'Export failed, please check settings', variant: 'destructive' });
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="space-y-2">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                <DownloadLineIcon className="w-5 h-5 text-primary" />
              </div>
              <CardTitle className="text-3xl font-display font-semibold">Export Papers</CardTitle>
            </div>
            <CardDescription className="text-base text-muted-foreground ml-13">
              将论文摘要导出为 CSV、JSON 文件或同步到飞书多维表格和 Zotero
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto px-8 py-12">
          <div className="max-w-4xl w-full space-y-8 mx-auto">
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>导出格式 (Format)</Label>
                  <Select value={format} onValueChange={(v:any)=>setFormat(v)}>
                    <SelectTrigger>
                      <SelectValue placeholder="选择导出格式" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="csv">
                        <div className="flex items-center gap-2"><TableLineIcon className="w-4 h-4"/> CSV</div>
                      </SelectItem>
                      <SelectItem value="json">
                        <div className="flex items-center gap-2"><BracesLineIcon className="w-4 h-4"/> JSON</div>
                      </SelectItem>
                      <SelectItem value="feishu">
                        <div className="flex items-center gap-2"><DatabaseLineIcon className="w-4 h-4"/> 飞书</div>
                      </SelectItem>
                      <SelectItem value="zotero">
                        <div className="flex items-center gap-2"><BookOpenLineIcon className="w-4 h-4"/> Zotero</div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {(format === 'csv' || format === 'json') && (
                  <div className="space-y-2">
                    <Label>输出路径 (Output)</Label>
                    <Input value={output} onChange={(e)=>setOutput(e.target.value)} placeholder="out/papers.csv 或 papers.json" />
                  </div>
                )}

                {format === 'feishu' && (
                  <div className="space-y-2">
                    <Label>飞书数据集名称</Label>
                    <Input value={feishuName} onChange={(e)=>setFeishuName(e.target.value)} placeholder="例如: 论文数据集" />
                  </div>
                )}

                {format === 'zotero' && (
                  <div className="space-y-2">
                    <Label>Zotero Collection Key (可选)</Label>
                    <Input value={collection} onChange={(e)=>setCollection(e.target.value)} placeholder="如 ABC123XY，不填则默认位置" />
                  </div>
                )}
              </div>

              <Separator className="bg-border/50" />

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>来源 (Source)</Label>
                  <Select value={source} onValueChange={(v:any)=>setSource(v)}>
                    <SelectTrigger>
                      <SelectValue placeholder="全部平台" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部</SelectItem>
                      <SelectItem value="arxiv">arXiv</SelectItem>
                      <SelectItem value="openreview">OpenReview</SelectItem>
                      <SelectItem value="acl">ACL Anthology</SelectItem>
                      <SelectItem value="ssrn">SSRN</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Limit</Label>
                  <Input type="number" value={limit} onChange={(e)=>setLimit(parseInt(e.target.value)||0)} placeholder="0 表示全部" />
                </div>

                <div className="space-y-2">
                  <Label>查询 (Query)</Label>
                  <Input value={query} onChange={(e)=>setQuery(e.target.value)} placeholder="在标题或摘要中查询" />
                </div>

                <div className="space-y-2">
                  <Label>关键词 (Keywords, 逗号分隔)</Label>
                  <Input value={keywords} onChange={(e)=>setKeywords(e.target.value)} placeholder="transformer, attention" />
                </div>

                <div className="space-y-2 col-span-2">
                  <Label>类别 (Categories, 逗号分隔)</Label>
                  <Input value={categories} onChange={(e)=>setCategories(e.target.value)} placeholder="cs.AI, cs.LG" />
                </div>
              </div>

              <div className="flex justify-end">
                <Button onClick={handleExport} disabled={exporting} className="px-8">
                  {exporting ? 'Exporting...' : 'Export'}
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default ExportView;