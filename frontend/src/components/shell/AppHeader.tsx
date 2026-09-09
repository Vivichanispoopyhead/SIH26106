import React from 'react';
import { MonoValue } from '../common/MonoValue';
import './AppHeader.css';

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
  return (
    <header className="app-topbar" data-testid="app-header">
      <div className="topbar-brand">
        <span className="brand-logo-badge">SIH26106</span>
        <div className="brand-titles">
          <h1 className="brand-name">Forensic Threat Intelligence</h1>
          <span className="brand-sub">Vertical Slice #1 • EML Ingestion & Parsing</span>
        </div>
      </div>

      <div className="topbar-context">
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
  );
};
