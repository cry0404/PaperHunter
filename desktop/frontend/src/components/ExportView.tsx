import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Separator } from './ui/separator';
import { useToast } from './ui/use-toast';

const ExportView: React.FC = () => {
  const { t } = useTranslation();
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
      toast({ title: t('exportView.validation.invalid'), description: t('exportView.validation.outputRequired'), variant: 'destructive' });
      return false;
    }
    if (format === 'feishu' && !feishuName.trim()) {
      toast({ title: t('exportView.validation.invalid'), description: t('exportView.validation.feishuNameRequired'), variant: 'destructive' });
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
          title: t('exportView.success.title'),
          description: (
            <div className="break-all">
              {t('exportView.success.feishu')} <a className="underline cursor-pointer" onClick={()=>BrowserOpenURL(result)}>{result}</a>
            </div>
          )
        });
      } else if ((format==='csv'||format==='json') && result) {
        const { BrowserOpenURL } = await import('../../wailsjs/runtime/runtime');
        toast({
          title: t('exportView.success.title'),
          description: (
            <div className="break-all">
              {t('exportView.success.saved')} <a className="underline cursor-pointer" onClick={()=>BrowserOpenURL(`file://${result}`)}>{result}</a>
            </div>
          )
        });
      } else {
        toast({ title: t('exportView.success.title'), description: t('exportView.success.operation') });
      }
    } catch (error) {
      console.error('Export failed:', error);
      toast({ title: t('exportView.failure.title'), description: t('exportView.failure.desc'), variant: 'destructive' });
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <CardTitle className="text-3xl font-sans font-medium tracking-tight">{t('export.title')}</CardTitle>
          <CardDescription className="text-base text-muted-foreground font-serif">
            {t('exportView.subtitle')}
          </CardDescription>
        </CardHeader>

        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-4xl w-full space-y-8 mx-auto">
            <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label className="font-sans">{t('exportView.selectFormat')}</Label>
                  <Select value={format} onValueChange={(v:any)=>setFormat(v)}>
                    <SelectTrigger className="font-sans bg-background border border-border text-foreground">
                      <SelectValue placeholder={t('exportView.selectFormat')} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="csv">CSV</SelectItem>
                      <SelectItem value="json">JSON</SelectItem>
                      <SelectItem value="feishu">{t('exportView.feishu')}</SelectItem>
                      <SelectItem value="zotero">Zotero</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {(format === 'csv' || format === 'json') && (
                  <div className="space-y-2">
                    <Label className="font-sans">{t('export.outputPath')}</Label>
                    <Input value={output} onChange={(e)=>setOutput(e.target.value)} placeholder={t('exportView.outputPathPlaceholder')} className="bg-background border border-border text-foreground font-mono text-sm" />
                  </div>
                )}

                {format === 'feishu' && (
                  <div className="space-y-2">
                    <Label className="font-sans">{t('export.feishuName')}</Label>
                    <Input value={feishuName} onChange={(e)=>setFeishuName(e.target.value)} placeholder={t('exportView.feishuNamePlaceholder')} className="bg-background border border-border text-foreground font-sans" />
                  </div>
                )}

                {format === 'zotero' && (
                  <div className="space-y-2">
                    <Label className="font-sans">{t('export.collectionKey')}</Label>
                    <Input value={collection} onChange={(e)=>setCollection(e.target.value)} placeholder={t('exportView.zoteroPlaceholder')} className="bg-background border border-border text-foreground font-mono text-sm" />
                  </div>
                )}
              </div>

              <Separator className="bg-border/40" />

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label className="font-sans">{t('exportView.source')}</Label>
                  <Select value={source} onValueChange={(v:any)=>setSource(v)}>
                    <SelectTrigger className="font-sans bg-background border border-border text-foreground">
                      <SelectValue placeholder={t('exportView.allPlatforms')} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{t('exportView.all')}</SelectItem>
                      <SelectItem value="arxiv">arXiv</SelectItem>
                      <SelectItem value="openreview">OpenReview</SelectItem>
                      <SelectItem value="acl">ACL Anthology</SelectItem>
                      <SelectItem value="ssrn">SSRN</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label className="font-sans">{t('exportView.limit')}</Label>
                  <Input type="number" value={limit} onChange={(e)=>setLimit(parseInt(e.target.value)||0)} placeholder={t('exportView.limitPlaceholder')} className="bg-background border border-border text-foreground font-sans" />
                </div>

                <div className="space-y-2">
                  <Label className="font-sans">{t('exportView.query')}</Label>
                  <Input value={query} onChange={(e)=>setQuery(e.target.value)} placeholder={t('exportView.queryPlaceholder')} className="bg-background border border-border text-foreground font-sans" />
                </div>

                <div className="space-y-2">
                  <Label className="font-sans">{t('exportView.keywords')}</Label>
                  <Input value={keywords} onChange={(e)=>setKeywords(e.target.value)} placeholder="transformer, attention" className="bg-background border border-border text-foreground font-sans" />
                </div>

                <div className="space-y-2 col-span-2">
                  <Label className="font-sans">{t('exportView.categories')}</Label>
                  <Input value={categories} onChange={(e)=>setCategories(e.target.value)} placeholder="cs.AI, cs.LG" className="bg-background border border-border text-foreground font-mono text-sm" />
                </div>
              </div>

              <div className="flex justify-end">
                <Button onClick={handleExport} disabled={exporting} className="px-8 font-sans bg-anthropic-dark text-anthropic-light hover:bg-anthropic-dark/90">
                  {exporting ? t('exportView.exporting') : t('export.confirm')}
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