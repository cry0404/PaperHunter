import React from 'react';

interface LogoProps {
  className?: string;
  size?: number;
}

export const Logo: React.FC<LogoProps> = ({ className = "", size = 32 }) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* 背景形状：抽象的书页/纸张，略微倾斜 */}
      <path
        d="M6 4C6 2.89543 6.89543 2 8 2H24C26.2091 2 28 3.79086 28 6V26C28 28.2091 26.2091 30 24 30H8C6.89543 30 6 29.1046 6 28V4Z"
        fill="currentColor"
        className="text-secondary"
        fillOpacity="0.2"
      />
      
      {/* 主要图形：衬线体 Q 的变体，结合放大镜/搜索意象 */}
      <path
        d="M16.5 6C11.8056 6 8 9.80558 8 14.5C8 19.1944 11.8056 23 16.5 23C18.4677 23 20.2794 22.3311 21.7148 21.2002L24.2574 23.7426C24.6479 24.1331 25.281 24.1331 25.6716 23.7426C26.0621 23.3521 26.0621 22.7189 25.6716 22.3284L23.1292 19.786C24.3095 18.3105 25 16.4802 25 14.5C25 9.80558 21.1944 6 16.5 6ZM10 14.5C10 10.9101 12.9101 8 16.5 8C20.0899 8 23 10.9101 23 14.5C23 18.0899 20.0899 21 16.5 21C12.9101 21 10 18.0899 10 14.5Z"
        fill="currentColor"
        className="text-primary"
      />
      
      {/* 装饰细节：书页线条 */}
      <rect x="10" y="12" width="2" height="5" rx="1" fill="currentColor" className="text-primary" fillOpacity="0.2" />
    </svg>
  );
};

export default Logo;






