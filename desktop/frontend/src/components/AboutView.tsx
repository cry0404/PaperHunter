import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import GithubLineIcon from 'remixicon-react/GithubLineIcon';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import Logo from './ui/logo';

const AboutView: React.FC = () => {
  const open = (url: string) => { try { BrowserOpenURL(url) } catch { window.open(url, '_blank') } };
  return (
    <div className="flex flex-col h-full overflow-hidden animate-fade-in">
      <Card className="flex-1 flex flex-col border-0 rounded-none shadow-none bg-transparent overflow-hidden">
        <CardHeader className="border-b border-border/30 bg-card/30 backdrop-blur-sm px-8 py-8 flex-shrink-0">
          <div className="space-y-2">
            <CardTitle className="text-3xl font-display font-semibold">About & Help</CardTitle>
            <CardDescription className="text-base text-muted-foreground ml-13">项目地址与常用配置说明</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="flex-1 overflow-y-auto px-8 py-8">
          <div className="max-w-3xl mx-auto space-y-6">
            {/* Header with Logo */}
            <div className="glass-card p-6 rounded-2xl flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center border border-primary/20">
                  <Logo size={32} className="text-primary" />
                </div>
                <div>
                  <div className="text-2xl font-display font-bold">PaperHunter</div>
                  <div className="text-sm text-muted-foreground">多平台学术论文爬取与本地语义检索工具</div>
                </div>
              </div>
              <Button variant="outline" onClick={()=>open('https://github.com/cry0404/PaperHunter')}>
                <GithubLineIcon className="w-4 h-4 mr-2" /> GitHub
              </Button>
            </div>
    

            <div className="glass-card p-6 rounded-2xl space-y-2">
              <div className="text-sm font-medium">关于本项目</div>
              <div className="text-sm text-muted-foreground">QuickSearchPaper 是一个多平台学术论文爬取与本地语义检索工具，支持 arXiv/OpenReview/ACL，并可导出到 CSV/JSON、Zotero、飞书多维表格。</div>
            </div>

            <div className="glass-card p-6 rounded-2xl space-y-2">
              <div className="text-sm font-medium">获取 Zotero API Key</div>
              <div className="text-sm text-muted-foreground">登录 Zotero 官网，依次进入 Settings → Feeds/API → Create new private key，复制 User ID 与 API Key，填入 Settings 页面对应字段。</div>
            </div>

            <div className="glass-card p-6 rounded-2xl space-y-2">
              <div className="text-sm font-medium">配置向量服务 API Key</div>
              <div className="text-sm text-muted-foreground">在 Settings → Embedding Service 填写 BaseURL、API Key、ModelName、Dim。推荐国内可用的 BaseURL（示例）: https://api.siliconflow.cn/v1。Dim 需与你的模型维度一致。</div>
            </div>

            <div className="glass-card p-6 rounded-2xl space-y-2">
              <div className="text-sm font-medium">飞书说明</div>
              <div className="text-sm text-muted-foreground">导出到飞书多维表格需要在 Settings 配置 AppID 与 AppSecret。详细指引待补充。</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default AboutView;
