import './styles/globals.css';
import React, { useState, useEffect } from 'react';
import TitleBar from './components/TitleBar';
import Layout, { ViewType } from './components/Layout';
import LogsView from './components/LogsView';
import SearchView from './components/SearchView';
import PapersView from './components/PapersView';
import LibraryView from './components/LibraryView';
import ExportView from './components/ExportView';
import SettingsView from './components/SettingsView';
import CleanView from './components/CleanView';
import AboutView from './components/AboutView';
import RecommendView from './components/RecommendView';
import { Toaster } from './components/ui/toaster';
import { CrawlProvider } from './context/CrawlContext';

function App() {
    const [currentView, setCurrentView] = useState<ViewType>('recommend');

    // 监听URL变化
    useEffect(() => {
        const handleHashChange = () => {
            const hash = window.location.hash;
            if (hash.startsWith('#/recommend')) {
                setCurrentView('recommend');
            } else if (hash.startsWith('#/logs')) {
                setCurrentView('logs');
            } else if (hash.startsWith('#/search')) {
                setCurrentView('search');
            } else if (hash.startsWith('#/papers')) {
                setCurrentView('papers');
            } else if (hash.startsWith('#/library')) {
                setCurrentView('library');
            } else if (hash.startsWith('#/export')) {
                setCurrentView('export');
            } else if (hash.startsWith('#/settings')) {
                setCurrentView('settings');
            } else if (hash.startsWith('#/clean')) {
                setCurrentView('clean');
            } else if (hash.startsWith('#/about')) {
                setCurrentView('about');
            } else {
                // 默认显示推荐页
                setCurrentView('recommend');
            }
        };

        // 初始检查：如果 hash 指向 export，就先设置为 export，避免先渲染 SearchView 触发 Wails API 调用
        handleHashChange();

        // 监听hash变化
        window.addEventListener('hashchange', handleHashChange);
        return () => window.removeEventListener('hashchange', handleHashChange);
    }, []);

    const renderView = () => {
        switch (currentView) {
            case 'recommend':
                return <RecommendView />;
            case 'logs':
                return <LogsView />;
            case 'search':
                return <SearchView />;
            case 'papers':
                return <PapersView />;
            case 'library':
                return <LibraryView />;
            case 'export':
                return <ExportView />;
            case 'settings':
                return <SettingsView />;
            case 'clean':
                return <CleanView />;
            case 'about':
                return <AboutView />;
            default:
                return <RecommendView />;
        }
    };

    return (
        <CrawlProvider>
            <TitleBar />
            <Layout onViewChange={setCurrentView}>
                {renderView()}
            </Layout>
            <Toaster />
        </CrawlProvider>
    );
}

export default App;
