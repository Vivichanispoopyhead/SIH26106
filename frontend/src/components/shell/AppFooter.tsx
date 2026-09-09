import React from 'react';
import './AppFooter.css';

interface AppFooterProps {
  apiBaseUrl: string;
}

export const AppFooter: React.FC<AppFooterProps> = ({ apiBaseUrl }) => {
  return (
    <footer className="app-bottom-bar" data-testid="app-footer">
      <div className="footer-left">
        <span className="footer-indicator">
          <span className="footer-dot" />
          API Target: <code className="footer-mono">{apiBaseUrl}</code>
        </span>
        <span className="footer-divider">•</span>
        <span>Contract: <code className="footer-mono">docs/api-contract.md</code></span>
      </div>

      <div className="footer-right">
        <span>Epistemic Model: STRICT OBSERVATION ONLY</span>
        <span className="footer-divider">•</span>
        <span>SIH26106 Monolith MVP</span>
      </div>
    </footer>
  );
};
