import React from 'react';
import { MonoValue } from '../common/MonoValue';
import './AppHeader.css';
import { AISettingsDialog } from './AISettingsDialog';
import { useState } from 'react';

interface AppHeaderProps {
  caseId?: string;
  emailId?: string;
  onResetCase?: () => void;
}

export const AppHeader: React.FC<AppHeaderProps> = ({
  caseId,
  emailId,
  onResetCase,
}) => {
  const [settingsOpen, setSettingsOpen] = useState(false);
  return (
    <>
    <header className="app-topbar" data-testid="app-header">
      <div className="topbar-brand">
        <span className="brand-logo-badge" aria-hidden="true">
          <span className="brand-logo-mark">S</span>
          <span>SIH26106</span>
        </span>
        <div className="brand-titles">
          <h1 className="brand-name">Forensic Threat Intelligence</h1>
          <span className="brand-sub">Forensic Threat Intelligence Console</span>
        </div>
      </div>

      <div className="topbar-context">
        <button type="button" className="btn-ai-settings" onClick={() => setSettingsOpen(true)}>AI provider</button>
        {caseId ? (
          <div className="case-id-pill" data-testid="active-case-pill">
            <span className="case-label">ACTIVE CASE:</span>
            <MonoValue value={caseId} label="Case ID" copyable={true} />
            {emailId && (
              <>
                <span className="case-label" style={{ marginLeft: '8px' }}>EMAIL:</span>
                <MonoValue value={emailId} label="Email ID" copyable={true} />
              </>
            )}
          </div>
        ) : (
          <span className="no-case-label">No Active Case</span>
        )}

        {onResetCase && caseId && (
          <button
            type="button"
            className="btn-new-sample"
            onClick={onResetCase}
            data-testid="btn-new-sample"
            title="Ingest a new .EML file"
          >
            + Ingest New Sample
          </button>
        )}
      </div>
    </header>
    <AISettingsDialog isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </>
  );
};
