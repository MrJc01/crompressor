import React from 'react';

interface ProgressBarProps {
  label: string;
  progress: number;
  total?: number;
}

export function ProgressBar({ label, progress, total }: ProgressBarProps) {
  const percentage = total && total > 0 ? Math.min(100, (progress / total) * 100) : progress;
  
  return (
    <div className="flex items-center gap-2 text-xs" style={{ minWidth: '200px' }}>
      <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
      <div className="flex-1 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--bg-hover)' }}>
        <div 
          className="h-full transition-all duration-300 ease-out" 
          style={{ width: `${percentage}%`, background: 'var(--accent)' }}
        />
      </div>
      <span className="font-mono w-10 text-right" style={{ color: 'var(--accent)' }}>
        {percentage.toFixed(0)}%
      </span>
    </div>
  );
}
