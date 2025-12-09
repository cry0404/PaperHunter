import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Checkbox } from './ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Separator } from './ui/separator';
import { useToast } from './ui/use-toast';
import { Trash2, FileOutput, Filter, Loader2 } from 'lucide-react';
import { CleanWithOptions } from '../../wailsjs/go/main/App';

const CleanView: React.FC = () => {
  const { t } = useTranslation();
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
      toast({ title: t('common.error'), description: 'Please specify export filename', variant: 'destructive' });
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

      const result = typeof res === 'string' ? JSON.parse(res) : res;
      
      const matched = result?.matched ?? result?.Matched ?? 0;
      const deleted = result?.deleted ?? result?.Deleted ?? 0;

      toast({ 
        title: t('common.success'), 
        description: `Matched: ${matched}, Deleted: ${deleted}`,
        variant: 'default'
      });
    } catch (e) {
      console.error(e);
      toast({ title: t('common.error'), description: 'Check your conditions and try again', variant: 'destructive' });
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="flex items-center gap-3">
            <div>
              <CardTitle className="text-3xl font-sans font-medium tracking-tight">{t('clean.title')}</CardTitle>
              <CardDescription className="text-base text-muted-foreground mt-1 font-serif">
                {t('clean.subtitle')}
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-4xl mx-auto space-y-8">
            
            {/* 筛选条件卡片 */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 mb-2">
                <Filter className="w-5 h-5 text-anthropic-orange" />
                <h3 className="text-lg font-sans font-medium text-foreground">{t('clean.filters')}</h3>
              </div>
              
              <div className="p-6 rounded-xl border border-border/60 bg-card/50 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <div className="space-y-2">
                  <Label className="flex items-center gap-2 font-sans text-foreground">
                    {t('clean.source')}
                  </Label>
                  <Select value={source} onValueChange={(v:any)=>setSource(v)}>
                    <SelectTrigger className="bg-background border-border">
                      <SelectValue placeholder={t('clean.allSources')} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{t('clean.allSources')}</SelectItem>
                      <SelectItem value="arxiv">arXiv</SelectItem>
                      <SelectItem value="openreview">OpenReview</SelectItem>
                      <SelectItem value="acl">ACL Anthology</SelectItem>
                      <SelectItem value="ssrn">SSRN</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label className="flex items-center gap-2 font-sans text-foreground">
                    {t('clean.startDate')}
                  </Label>
                  <Input type="date" value={from} onChange={(e)=>setFrom(e.target.value)} className="bg-background border-border font-sans" />
                </div>

                <div className="space-y-2">
                  <Label className="flex items-center gap-2 font-sans text-foreground">
                     {t('clean.endDate')}
                  </Label>
                  <Input type="date" value={until} onChange={(e)=>setUntil(e.target.value)} className="bg-background border-border font-sans" />
                </div>

                <div className="space-y-2">
                  <Label className="flex items-center gap-2 font-sans text-foreground">
                     {t('clean.options')}
                  </Label>
                  <div className="h-10 flex items-center">
                    <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-anthropic-orange transition-colors font-sans text-foreground">
                      <Checkbox checked={withoutEmbed} onCheckedChange={(v:boolean)=>setWithoutEmbed(v)} />
                      <span>{t('clean.missingEmbeddings')}</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <Separator className="bg-border/60" />

            {/* 导出选项卡片 */}
            <div className="space-y-4">
               <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <FileOutput className="w-5 h-5 text-anthropic-orange" />
                  <h3 className="text-lg font-sans font-medium text-foreground">{t('clean.backup')}</h3>
                </div>
                <label className="flex items-center gap-2 text-sm cursor-pointer hover:text-anthropic-orange transition-colors font-sans text-foreground">
                  <Checkbox checked={exportBefore} onCheckedChange={(v:boolean)=>setExportBefore(v)} />
                  <span className="font-medium">{t('clean.backupCheck')}</span>
                </label>
              </div>

              <div className={`p-6 rounded-xl border border-border/60 bg-card/50 transition-all duration-300 ${!exportBefore ? 'opacity-90 pointer-events-none' : ''}`}>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label className="font-sans text-foreground">{t('clean.format')}</Label>
                    <Select value={exportFormat} onValueChange={(v:any)=>setExportFormat(v)} disabled={!exportBefore}>
                      <SelectTrigger className="bg-background border-border">
                        <SelectValue placeholder="Select Format" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="csv">CSV</SelectItem>
                        <SelectItem value="json">JSON</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label className="font-sans text-foreground">{t('clean.filename')}</Label>
                    <Input 
                      value={exportOutput} 
                      onChange={(e)=>setExportOutput(e.target.value)} 
                      placeholder="backup/clean_backup.csv" 
                      disabled={!exportBefore}
                      className="bg-background border-border font-mono text-sm placeholder:text-gray-400 dark:placeholder:text-gray-500"
                    />
                    <p className="text-xs text-muted-foreground font-sans">Saved relative to app directory</p>
                  </div>
                </div>
              </div>
            </div>

            {/* 操作按钮 */}
            <div className="flex justify-end pt-4">
              <Button 
                onClick={handleClean} 
                disabled={running} 
                className="px-8 h-12 bg-destructive hover:bg-destructive/90 text-destructive-foreground font-sans text-base transition-all shadow-md hover:shadow-lg"
              >
                {running ? (
                  <>
                    <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                    {t('clean.cleaning')}
                  </>
                ) : (
                  <>
                    <Trash2 className="mr-2 h-5 w-5" />
                    {t('clean.startCleanup')}
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
