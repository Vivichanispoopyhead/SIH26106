import React, { useState } from 'react';
import './MonoValue.css';

interface MonoValueProps {
  value: string;
  label?: string;
  copyable?: boolean;
  truncate?: boolean;
  className?: string;
}

export const MonoValue: React.FC<MonoValueProps> = ({
  value,
  label,
  copyable = true,
  truncate = false,
  className = '',
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      // Clipboard write not permitted
    }
  };

  return (
    <span
      className={`mono-value-container ${truncate ? 'truncate' : ''} ${className}`}
      title={label ? `${label}: ${value}` : value}
    >
      <code className="mono-text">{value}</code>
      {copyable && (
        <button
          type="button"
          className="copy-btn"
          onClick={handleCopy}
          aria-label={`Copy ${label || 'value'} to clipboard`}
          title={copied ? 'Copied!' : 'Copy to clipboard'}
        >
          {copied ? '✓' : '⧉'}
        </button>
      )}
    </span>
  );
};
