import React, { useState } from 'react';
import { ParsedEmailResponse, EmailAnalysisResponse } from '../../types/api';
import { EmailHeaderCard } from './EmailStage/EmailHeaderCard';
import { MimeSummaryCard } from './EmailStage/MimeSummaryCard';
import { HeaderTable } from './HeadersStage/HeaderTable';
import { IndicatorTable } from './IndicatorsStage/IndicatorTable';
import { AttachmentList } from './EmailStage/AttachmentList';
import { AIAssessmentPanel } from './AIAssessmentStage/AIAssessmentPanel';
import { ProvenanceBadge } from '../common/ProvenanceBadge';
import { MonoValue } from '../common/MonoValue';
import './ParsedEmailWorkspace.css';

interface ParsedEmailWorkspaceProps {
  emailData: ParsedEmailResponse;
  analysisData?: EmailAnalysisResponse | null;
  analysisLoading?: boolean;
  analysisError?: unknown | null;
}

type WorkspaceView = 'all' | 'assessment' | 'metadata' | 'indicators' | 'headers' | 'attachments';

export const ParsedEmailWorkspace: React.FC<ParsedEmailWorkspaceProps> = ({
  emailData,
  analysisData,
  analysisLoading,
  analysisError,
}) => {
  const [activeView, setActiveView] = useState<WorkspaceView>('all');

  const headerCount = emailData.headers?.length || 0;
  const indicatorCount =
    (emailData.indicators?.ips?.length || 0) +
    (emailData.indicators?.domains?.length || 0) +
    (emailData.indicators?.urls?.length || 0);
  const attachmentCount = emailData.attachments?.length || 0;

  const hasAnalysis = analysisData !== undefined || analysisLoading || Boolean(analysisError);
  const messageCount = (emailData.message?.from?.length || 0) + (emailData.message?.to?.length || 0);

  return (
    <div className="parsed-workspace-container" data-testid="parsed-email-workspace">
      {/* Forensic Triage Ribbon */}
      <div className="case-triage-ribbon" data-testid="case-triage-ribbon">
        <div className="triage-left">
          <span className="triage-eyebrow">Case Ingestion Complete</span>
          <h2 className="triage-subject" data-testid="triage-subject">
            {emailData.message?.subject || emailData.filename || 'Parsed Email Sample'}
          </h2>
          <div className="triage-identifiers">
            <span className="identifier-chip">
              <span className="id-label">Email ID:</span>
              <MonoValue value={emailData.email_id} label="Email ID" copyable={true} />
            </span>
            <span className="identifier-chip">
              <span className="id-label">Case ID:</span>
              <MonoValue value={emailData.case_id} label="Case ID" copyable={true} />
            </span>
            {emailData.filename && (
              <span className="identifier-chip">
                <span className="id-label">Artifact:</span>
                <code className="id-file">{emailData.filename}</code>
              </span>
            )}
          </div>

          <div className="workspace-summary-grid" aria-label="Observed artifact summary">
            <div className="workspace-summary-item">
              <span className="summary-label">Observed headers</span>
              <strong>{headerCount}</strong>
            </div>
            <div className="workspace-summary-item">
              <span className="summary-label">Network indicators</span>
              <strong>{indicatorCount}</strong>
            </div>
            <div className="workspace-summary-item">
              <span className="summary-label">Attachments</span>
              <strong>{attachmentCount}</strong>
            </div>
            <div className="workspace-summary-item">
              <span className="summary-label">Envelope addresses</span>
              <strong>{messageCount}</strong>
            </div>
          </div>
        </div>

        <div className="triage-right">
          <div className="status-group">
            <span className="status-label">Ingestion State</span>
            <span className="status-parsed-pill" data-testid="parsed-status-badge">
              ✓ PARSED
            </span>
          </div>
          <div className="provenance-group">
            <span className="status-label">Epistemic Class</span>
            <ProvenanceBadge classification="OBSERVED" />
          </div>
        </div>
      </div>

      {/* Forensic Workspace Tabs */}
      <div className="workspace-tabs-bar" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={activeView === 'all'}
          className={`workspace-tab ${activeView === 'all' ? 'active' : ''}`}
          onClick={() => setActiveView('all')}
          data-testid="view-tab-all"
        >
          Comprehensive View
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeView === 'assessment'}
          className={`workspace-tab ${activeView === 'assessment' ? 'active' : ''}`}
          onClick={() => setActiveView('assessment')}
          data-testid="view-tab-assessment"
        >
          AI Assessment
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeView === 'metadata'}
          className={`workspace-tab ${activeView === 'metadata' ? 'active' : ''}`}
          onClick={() => setActiveView('metadata')}
          data-testid="view-tab-metadata"
        >
          Envelope & MIME
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeView === 'indicators'}
          className={`workspace-tab ${activeView === 'indicators' ? 'active' : ''}`}
          onClick={() => setActiveView('indicators')}
          data-testid="view-tab-indicators"
        >
          Indicators ({indicatorCount})
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeView === 'headers'}
          className={`workspace-tab ${activeView === 'headers' ? 'active' : ''}`}
          onClick={() => setActiveView('headers')}
          data-testid="view-tab-headers"
        >
          Headers ({headerCount})
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeView === 'attachments'}
          className={`workspace-tab ${activeView === 'attachments' ? 'active' : ''}`}
          onClick={() => setActiveView('attachments')}
          data-testid="view-tab-attachments"
        >
          Attachments ({attachmentCount})
        </button>
      </div>

      {/* Structured Sections */}
      <div className="workspace-content">
        {(activeView === 'assessment' || (activeView === 'all' && hasAnalysis)) && (
          <section className="workspace-section" data-testid="section-assessment">
            <AIAssessmentPanel
              analysisData={analysisData}
              isLoading={analysisLoading}
              error={analysisError}
            />
          </section>
        )}

        {(activeView === 'all' || activeView === 'metadata') && (
          <section className="workspace-section" data-testid="section-envelope">
            <EmailHeaderCard message={emailData.message} filename={emailData.filename} />
            <MimeSummaryCard mime={emailData.mime} />
          </section>
        )}

        {(activeView === 'all' || activeView === 'indicators') && (
          <section className="workspace-section" data-testid="section-indicators">
            <IndicatorTable indicators={emailData.indicators} />
          </section>
        )}

        {(activeView === 'all' || activeView === 'headers') && (
          <section className="workspace-section" data-testid="section-headers">
            <HeaderTable headers={emailData.headers} />
          </section>
        )}

        {(activeView === 'all' || activeView === 'attachments') && (
          <section className="workspace-section" data-testid="section-attachments">
            <AttachmentList attachments={emailData.attachments} />
          </section>
        )}

        {/* Pipeline Progression Notice */}
        <section className="pipeline-notice-box" data-testid="pipeline-notice">
          <div className="notice-icon" aria-hidden="true">ℹ</div>
          <div className="notice-content">
            <h4 className="notice-title">Stage 01: Ingestion & Parsing Complete</h4>
            <p className="notice-text">
              The RFC 5322 structure, headers, MIME boundaries, extracted network indicators,
              and attachment digests have been recorded in the case repository as <strong>OBSERVED</strong> facts.
              Subsequent forensic stages (SPF/DKIM alignment, relay chain timeline, IP geolocation,
              reputation enrichment, and entity graph) execute in subsequent pipeline slices.
            </p>
          </div>
        </section>
      </div>
    </div>
  );
};
