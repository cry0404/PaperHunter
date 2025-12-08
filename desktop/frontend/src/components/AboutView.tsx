import React from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Github } from 'lucide-react';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import Logo from './ui/logo';

const AboutView: React.FC = () => {
  const { t } = useTranslation();
  const open = (url: string) => { try { BrowserOpenURL(url) } catch { window.open(url, '_blank') } };
  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in bg-background">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-background/50 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="space-y-2">
            <CardTitle className="text-3xl font-sans font-medium tracking-tight">{t('about.title')}</CardTitle>
            <CardDescription className="text-base text-muted-foreground font-serif">{t('about.subtitle')}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-3xl mx-auto space-y-6">
            {/* Header with Logo */}
            <div className="p-6 rounded-xl border border-border/40 bg-card/30 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div>
                  <div className="text-2xl font-sans font-bold">PaperHunter</div>
                  <div className="text-sm text-muted-foreground font-serif">{t('about.desc')}</div>
                </div>
              </div>
              <Button variant="outline" onClick={()=>open('https://github.com/cry0404/PaperHunter')} className="font-sans">
                <Github className="w-4 h-4 mr-2" /> {t('about.github')}
              </Button>
            </div>
    

            <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-2">
              <div className="text-sm font-medium font-sans">{t('about.aboutTitle')}</div>
              <div className="text-sm text-muted-foreground font-serif leading-relaxed">{t('about.aboutContent')}</div>
            </div>

            <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-2">
              <div className="text-sm font-medium font-sans">{t('about.zoteroTitle')}</div>
              <div className="text-sm text-muted-foreground font-serif leading-relaxed">{t('about.zoteroContent')}</div>
            </div>

            <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-2">
              <div className="text-sm font-medium font-sans">{t('about.embeddingTitle')}</div>
              <div className="text-sm text-muted-foreground font-serif leading-relaxed">{t('about.embeddingContent')}</div>
            </div>

            <div className="p-6 rounded-xl border border-border/40 bg-card/30 space-y-2">
              <div className="text-sm font-medium font-sans">{t('about.feishuTitle')}</div>
              <div className="text-sm text-muted-foreground font-serif leading-relaxed">{t('about.feishuContent')}</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default AboutView;
