import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Checkbox } from './ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { useToast } from './ui/use-toast';
import { Eraser, Trash2, FileOutput, Calendar, Database, Filter } from 'lucide-react';
import { CleanWithOptions } from '../../wailsjs/go/main/App';

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
    if (exportBefore && !exportOutput.trim()) {
      toast({ title: 'Invalid input', description: '请填写导出文件名', variant: 'destructive' });
      return;
    }

    setRunning(true);
    try {
      // 调用 Wails 后端方法
      const res = await CleanWithOptions({
        source: source === 'all' ? '' : source,
        from: from,
        until: until,
        withoutEmbed: withoutEmbed,
        exportBefore: exportBefore,
        exportFormat: exportFormat,
        exportOutput: exportOutput
      } as any);

      // Wails 返回的对象通常属性是小写的（基于 JSON tag）
      // 使用类型断言或 any 来访问，或者假设它符合 CleanResult 接口
      const result = typeof res === 'string' ? JSON.parse(res) : res;
      
      // 修复：使用小写属性访问，因为 Wails 生成的 JS 对象对应 Go struct 的 json tags
      const matched = result?.matched ?? result?.Matched ?? 0;
      const deleted = result?.deleted ?? result?.Deleted ?? 0;

      toast({ 
        title: '清理完成', 
        description: `匹配 ${matched} 篇，删除 ${deleted} 篇`,
        variant: 'default'
      });
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
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <Eraser className="w-5 h-5 text-primary" />
            </div>
            <div>
              <CardTitle className="text-3xl font-display font-semibold">Data Cleaning</CardTitle>
              <CardDescription className="text-base text-muted-foreground mt-1">
                管理和清理数据库中的论文数据，支持条件筛选和导出备份
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-4xl mx-auto space-y-6">
            
            {/* 筛选条件卡片 */}
            <div className="glass-card p-6 rounded-2xl space-y-6 border border-border/50 bg-card/50">
              <div className="flex items-center gap-2 mb-4">
                <Filter className="w-5 h-5 text-primary" />
                <h3 className="text-lg font-semibold">筛选条件</h3>
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <div className="space-y-2">
                  <Label className="flex items-center gap-2">
                    <Filter className="w-4 h-4" /> 来源 (Source)
                  </Label>
                  <Select value={source} onValueChange={(v:any)=>setSource(v)}>
                    <SelectTrigger>
                      <SelectValue placeholder="全部平台" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部平台</SelectItem>
                      <SelectItem value="arxiv">arXiv</SelectItem>
                      <SelectItem value="openreview">OpenReview</SelectItem>
                      <SelectItem value="acl">ACL Anthology</SelectItem>
                      <SelectItem value="ssrn">SSRN</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label className="flex items-center gap-2">
                    <Calendar className="w-4 h-4" /> Start Date
                  </Label>
                  <Input type="date" value={from} onChange={(e)=>setFrom(e.target.value)} />
                </div>

                <div className="space-y-2">
                  <Label className="flex items-center gap-2">
                    <Calendar className="w-4 h-4" /> End Date
                  </Label>
                  <Input type="date" value={until} onChange={(e)=>setUntil(e.target.value)} />
                </div>

                <div className="space-y-2">
                  <Label className="flex items-center gap-2">
                    <Database className="w-4 h-4" /> 数据选项
                  </Label>
                  <div className="h-10 flex items-center">
                    <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-primary transition-colors">
                      <Checkbox checked={withoutEmbed} onCheckedChange={(v:boolean)=>setWithoutEmbed(v)} />
                      <span>仅清理无向量数据</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            {/* 导出选项卡片 */}
            <div className="glass-card p-6 rounded-2xl space-y-6 border border-border/50 bg-card/50">
               <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <FileOutput className="w-5 h-5 text-primary" />
                  <h3 className="text-lg font-semibold">备份选项</h3>
                </div>
                <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-primary transition-colors">
                  <Checkbox checked={exportBefore} onCheckedChange={(v:boolean)=>setExportBefore(v)} />
                  <span className="font-medium">删除前导出备份</span>
                </label>
              </div>

              {exportBefore && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 animate-in fade-in slide-in-from-top-2 duration-200">
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
                    <Label>输出文件名</Label>
                    <Input 
                      value={exportOutput} 
                      onChange={(e)=>setExportOutput(e.target.value)} 
                      placeholder="例如: backup/clean_backup.csv" 
                    />
                    <p className="text-xs text-muted-foreground">文件将保存在应用运行目录</p>
                  </div>
                </div>
              )}
            </div>

            {/* 操作按钮 */}
            <div className="flex justify-end pt-4">
              <Button 
                onClick={handleClean} 
                disabled={running} 
                className="px-8 bg-destructive hover:bg-destructive/90 text-destructive-foreground transition-all hover:scale-105 shadow-lg shadow-destructive/20"
                size="lg"
              >
                {running ? (
                  <>
                    <Database className="mr-2 h-4 w-4 animate-pulse" />
                    Cleaning...
                  </>
                ) : (
                  <>
                    <Trash2 className="mr-2 h-4 w-4" />
                    Start Clean
                  </>
                )}
              </Button>
            </div>

          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default CleanView;
