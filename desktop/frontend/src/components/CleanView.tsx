import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Checkbox } from './ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Separator } from './ui/separator';
import { useToast } from './ui/use-toast';

const CleanView: React.FC = () => {
  const [source, setSource] = useState<'all'|'arxiv'|'openreview'|'acl'>('all');
  const [from, setFrom] = useState('');
  const [until, setUntil] = useState('');
  const [withoutEmbed, setWithoutEmbed] = useState(false);
  const [exportBefore, setExportBefore] = useState(false);
  const [exportFormat, setExportFormat] = useState<'csv'|'json'>('csv');
  const [exportOutput, setExportOutput] = useState('');
  const [running, setRunning] = useState(false);
  const { toast } = useToast();

  const handleClean = async () => {
    if (!exportBefore || exportOutput.trim()) {

    } else {
      toast({ title: 'Invalid input', description: '请填写导出文件名', variant: 'destructive' });
      return;
    }

    setRunning(true);
    try {
      const { CleanWithOptions } = await import('../../wailsjs/go/main/App');
      const res = await CleanWithOptions({
        Source: source === 'all' ? '' : source,
        From: from,
        Until: until,
        WithoutEmbed: withoutEmbed,
        ExportBefore: exportBefore,
        ExportFormat: exportFormat,
        ExportOutput: exportOutput
      } as any);
      const result = typeof res === 'string' ? JSON.parse(res) : res;
      toast({ title: '清理完成', description: `匹配 ${result?.Matched ?? 0}，删除 ${result?.Deleted ?? 0}` });
    } catch (e) {
      console.error(e);
      toast({ title: '清理失败', description: '请检查条件与配置', variant: 'destructive' });
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="space-y-2">
            <CardTitle className="text-3xl font-display font-semibold">Clean Papers</CardTitle>
            <CardDescription className="text-base text-muted-foreground ml-13">按条件删除冗余数据，支持删除前导出</CardDescription>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-4xl mx-auto space-y-6">
            <div className="glass-card p-6 rounded-2xl space-y-6">
              <div className="grid grid-cols-4 gap-4">
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
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>From</Label>
                  <Input type="date" value={from} onChange={(e)=>setFrom(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>Until</Label>
                  <Input type="date" value={until} onChange={(e)=>setUntil(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>Options</Label>
                  <label className="flex items-center gap-2 text-sm">
                    <Checkbox checked={withoutEmbed} onCheckedChange={(v:boolean)=>setWithoutEmbed(v)} />
                    <span>仅无向量 (embedding IS NULL)</span>
                  </label>
                </div>
              </div>

              <Separator className="bg-border/50" />

              <div className="grid grid-cols-3 gap-4">
                <label className="flex items-center gap-2 text-sm">
                  <Checkbox checked={exportBefore} onCheckedChange={(v:boolean)=>setExportBefore(v)} />
                  <span>删除前导出</span>
                </label>
                <div className="space-y-2">
                  <Label>导出格式</Label>
                  <Select value={exportFormat} onValueChange={(v:any)=>setExportFormat(v)}>
                    <SelectTrigger>
                      <SelectValue placeholder="选择格式" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="csv">CSV</SelectItem>
                      <SelectItem value="json">JSON</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>输出文件</Label>
                  <Input value={exportOutput} onChange={(e)=>setExportOutput(e.target.value)} placeholder="backup/clean_backup.csv" />
                </div>
              </div>

              <div className="flex justify-end">
                <Button onClick={handleClean} disabled={running} className="px-8">
                  {running ? 'Cleaning...' : 'Start Clean'}
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default CleanView;



