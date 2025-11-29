import React from 'react';
import { Separator } from './ui/separator';
import { 
  Sparkles, 
  Search, 
  Library, 
  Download, 
  Activity, 
  Eraser, 
  Settings, 
  Info,
  FileText
} from 'lucide-react';

interface LayoutProps {
  children: React.ReactNode;
  onViewChange?: (view: ViewType) => void;
}

export type ViewType = 'logs' | 'search' | 'papers' | 'library' | 'export' | 'settings' | 'clean' | 'about' | 'recommend';

const Layout: React.FC<LayoutProps> = ({ children, onViewChange }) => {
  const [activeView, setActiveView] = React.useState<ViewType>('recommend');

  const handleViewChange = (view: ViewType) => {
    setActiveView(view);
    onViewChange?.(view);
    
    // Update URL
    window.location.hash = `#/${view}`;
  };

  // Top Navigation Items
  const topNavItems = [
    { id: 'recommend' as ViewType, icon: Sparkles, label: 'Recommendations' },
    { id: 'search' as ViewType, icon: Search, label: 'Crawl Papers' },
    { id: 'library' as ViewType, icon: Library, label: 'Library' },
    // { id: 'papers' as ViewType, icon: FileText, label: 'Semantic Search' }, // Merged into Library
    { id: 'export' as ViewType, icon: Download, label: 'Export' },
    { id: 'logs' as ViewType, icon: Activity, label: 'Logs' },
  ];

  // Bottom Navigation Items
  const bottomNavItems = [
    { id: 'clean' as ViewType, icon: Eraser, label: 'Clean' },
    { id: 'settings' as ViewType, icon: Settings, label: 'Settings' },
    { id: 'about' as ViewType, icon: Info, label: 'About & Help' },
  ];

  return (
    <div className="flex h-screen bg-background paper-texture overflow-hidden pt-12 rounded-xl text-foreground">
      {/* Sidebar */}
      <div className="w-20 flex flex-col items-center pb-6 gap-2 z-20 border-r border-border bg-background/50 backdrop-blur-sm">
        
        <Separator className="w-10 mt-4 mb-2 bg-border/60" />

        {/* Top Navigation */}
        <nav className="flex-1 flex flex-col gap-3 mt-2 px-2 w-full items-center">
          {topNavItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeView === item.id;
            return (
              <div key={item.id} className="relative group flex justify-center w-full">
                <button
                  onClick={() => handleViewChange(item.id)}
                  className={`
                    w-10 h-10 rounded-lg flex items-center justify-center relative
                    transition-all duration-200 ease-out
                    ${isActive
                      ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                      : 'text-muted-foreground hover:text-foreground hover:bg-secondary/30'
                    }
                  `}
                  title={item.label}
                >
                  <Icon className="w-5 h-5" strokeWidth={2} />
                </button>
                
                {/* Tooltip */}
                <div className="absolute left-full ml-4 px-3 py-1.5 bg-foreground text-background text-xs font-medium rounded opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity duration-200 whitespace-nowrap z-[100] shadow-xl">
                  {item.label}
                  {/* Triangle */}
                  <div className="absolute right-full top-1/2 -translate-y-1/2 -mr-1 border-4 border-transparent border-r-foreground" />
                </div>
              </div>
            );
          })}
        </nav>

        {/* Bottom Navigation */}
        <div className="flex flex-col gap-3 px-2 w-full items-center">
          {bottomNavItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeView === item.id;
            return (
              <div key={item.id} className="relative group flex justify-center w-full">
                <button
                  onClick={() => handleViewChange(item.id)}
                  className={`
                    w-10 h-10 rounded-lg flex items-center justify-center relative
                    transition-all duration-200 ease-out
                    ${isActive
                      ? 'bg-secondary text-secondary-foreground shadow-md shadow-secondary/20'
                      : 'text-muted-foreground hover:text-foreground hover:bg-secondary/30'
                    }
                  `}
                  title={item.label}
                >
                  <Icon className="w-5 h-5" strokeWidth={2} />
                </button>
                
                {/* Tooltip */}
                <div className="absolute left-full ml-4 px-3 py-1.5 bg-foreground text-background text-xs font-medium rounded opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity duration-200 whitespace-nowrap z-[100] shadow-xl">
                  {item.label}
                  <div className="absolute right-full top-1/2 -translate-y-1/2 -mr-1 border-4 border-transparent border-r-foreground" />
                </div>
              </div>
            );
          })}
        </div>

        {/* Version Badge - simplified */}
        <div className="mt-4">
          <span className="text-[10px] text-muted-foreground/60 font-mono">v1.0</span>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col overflow-hidden relative">
        <div className="relative z-10 flex-1 flex flex-col overflow-hidden">
          {children}
        </div>
      </div>
    </div>
  );
};

export default Layout;
