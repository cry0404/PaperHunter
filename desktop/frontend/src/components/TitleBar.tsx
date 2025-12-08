import React, { useEffect, useState } from 'react';
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime';
import { X, Minus, Square } from 'lucide-react';

// Wails 拖动区域样式
const draggableStyle: React.CSSProperties = {
  // @ts-ignore - Wails 自定义 CSS 属性
  '--wails-draggable': 'drag'
};

const noDragStyle: React.CSSProperties = {
  // @ts-ignore - Wails 自定义 CSS 属性
  '--wails-draggable': 'no-drag'
};

const TitleBar: React.FC = () => {
  const [isMac, setIsMac] = useState(false);

  useEffect(() => {
    // 检测操作系统
    const userAgent = window.navigator.userAgent.toLowerCase();
    setIsMac(userAgent.includes('mac'));
  }, []);

  const handleMinimize = () => {
    WindowMinimise();
  };

  const handleMaximize = () => {
    WindowToggleMaximise();
  };

  const handleClose = () => {
    Quit();
  };

  return (
    <div
      className="fixed top-0 left-0 right-0 h-12 bg-card/80 backdrop-blur-xl border-b border-border/30 z-50 flex items-center overflow-hidden shadow-sm"
      style={{...draggableStyle, borderTopLeftRadius: '12px', borderTopRightRadius: '12px'}}
    >
      {/* macOS - 给系统红绿灯按钮留出空间 */}
      {isMac && <div className="w-[80px]" />}

      {/* 中间标题区域 */}
      <div className="flex-1 flex items-center justify-center">
        <span className="text-base font-sans font-bold text-foreground tracking-wide">PaperHunter</span>
      </div>

      {/* Windows 风格 - 按钮在右边 */}
      {!isMac && (
        <div className="flex items-center h-full" style={noDragStyle}>
          <button
            onClick={handleMinimize}
            className="h-full px-4 hover:bg-secondary/80 transition-all duration-200 flex items-center justify-center group"
            title="Minimize"
          >
            <Minus className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
          </button>
          <button
            onClick={handleMaximize}
            className="h-full px-4 hover:bg-secondary/80 transition-all duration-200 flex items-center justify-center group"
            title="Maximize"
          >
            <Square className="w-3.5 h-3.5 text-muted-foreground group-hover:text-foreground transition-colors" />
          </button>
          <button
            onClick={handleClose}
            className="h-full px-4 hover:bg-destructive hover:text-destructive-foreground transition-all duration-200 flex items-center justify-center"
            title="Close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* macOS - 右侧留空保持视觉平衡 */}
      {isMac && <div className="w-[80px]" />}
    </div>
  );
};

export default TitleBar;

