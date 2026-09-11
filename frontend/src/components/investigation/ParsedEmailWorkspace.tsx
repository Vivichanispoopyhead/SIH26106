import React, { useState } from 'react';
import {
  ParsedEmailResponse,
  EmailAnalysisResponse,
  EmailEvidenceResponse,
  CaseGraphResponse,
  CaseTimelineResponse,
  ForensicReport,
  EvidenceItem,
  RiskSignal,
} from '../../types/api';
import { EpistemicClass } from '../../types/provenance';
import { EmailHeaderCard } from './EmailStage/EmailHeaderCard';
import { MimeSummaryCard } from './EmailStage/MimeSummaryCard';
import { HeaderTable } from './HeadersStage/HeaderTable';
import { IndicatorTable } from './IndicatorsStage/IndicatorTable';
import { AttachmentList } from './EmailStage/AttachmentList';
import { AIAssessmentPanel } from './AIAssessmentStage/AIAssessmentPanel';
import { EvidencePanel } from './EvidenceStage/EvidencePanel';
import { EvidenceDrawer } from './EvidenceStage/EvidenceDrawer';
import { EnrichmentPanel, GraphPanel, ReportPanel, TimelinePanel } from './ForensicPanels';
import { ProvenanceBadge } from '../common/ProvenanceBadge';
import { MonoValue } from '../common/MonoValue';
import './ParsedEmailWorkspace.css';

export interface ParsedEmailWorkspaceProps {
  emailData: ParsedEmailResponse;
  analysisData?: EmailAnalysisResponse | null;
  analysisLoading?: boolean;
  analysisError?: unknown | null;
  evidenceData?: EmailEvidenceResponse | null;
  evidenceLoading?: boolean;
  evidenceError?: unknown | null;
  onRetryEvidence?: () => void;
  graphData?: CaseGraphResponse | null;
  graphLoading?: boolean;
  graphError?: unknown | null;
  timelineData?: CaseTimelineResponse | null;
  timelineLoading?: boolean;
  timelineError?: unknown | null;
  reportData?: ForensicReport | null;
  reportLoading?: boolean;
  reportError?: unknown | null;
  onGenerateReport?: () => void;
  onDownloadReportPDF?: () => Promise<{ blob: Blob; filename: string }>;
  requestedView?: WorkspaceView;
}

export type WorkspaceView =
  | 'all'
  | 'assessment'
  | 'evidence'
  | 'metadata'
  | 'indicators'
  | 'headers'
  | 'attachments'
  | 'timeline'
  | 'graph'
  | 'enrichment'
  | 'report';

export const ParsedEmailWorkspace: React.FC<ParsedEmailWorkspaceProps> = ({
  emailData,
  analysisData,
  analysisLoading,
  analysisError,
  evidenceData,
  evidenceLoading,
  evidenceError,
  onRetryEvidence,
  graphData,
  graphLoading,
  graphError,
  timelineData,
  timelineLoading,
  timelineError,
  reportData,
  reportLoading,
  reportError,
  onGenerateReport,
  onDownloadReportPDF,
  requestedView,
}) => {
  const [activeView, setActiveView] = useState<WorkspaceView>('all');
  const [selectedSignalCode, setSelectedSignalCode] = useState<string | null>(null);
  const [selectedEvidenceQuery, setSelectedEvidenceQuery] = useState<string | null>(null);
  const [drawerItem, setDrawerItem] = useState<EvidenceItem | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState<boolean>(false);

  React.useEffect(() => {
    if (requestedView) setActiveView(requestedView);
  }, [requestedView]);

  const headerCount = emailData.headers?.length || 0;
  const indicatorCount =
    (emailData.indicators?.ips?.length || 0) +
    (emailData.indicators?.domains?.length || 0) +
    (emailData.indicators?.urls?.length || 0);
  const attachmentCount = emailData.attachments?.length || 0;
  const evidenceCount = evidenceData?.evidence?.length || 0;
  const messageCount = (emailData.message?.from?.length || 0) + (emailData.message?.to?.length || 0);

  const hasAnalysis = analysisData !== undefined || analysisLoading || Boolean(analysisError);
  const hasEvidence = evidenceData !== undefined || evidenceLoading || Boolean(evidenceError);
  const riskSignals: RiskSignal[] = analysisData?.risk?.contributing_signals ?? [];

  // Cross-navigation handlers
  const handleSelectSignalCode = (code: string) => {
    setSelectedSignalCode(code);
    setSelectedEvidenceQuery(null);
    if (activeView !== 'evidence' && activeView !== 'all') {
      setActiveView('evidence');
    }
  };

  const handleSelectEvidenceReference = (ref: string) => {
    setSelectedEvidenceQuery(ref);
    setSelectedSignalCode(null);
    if (activeView !== 'evidence' && activeView !== 'all') {
      setActiveView('evidence');
    }
    if (evidenceData?.evidence) {
      const match = evidenceData.evidence.find(
        (e) =>
          e.evidence_id.toLowerCase() === ref.toLowerCase() ||
          e.source.toLowerCase() === ref.toLowerCase()
      );
      if (match) {
        setDrawerItem(match);
        setIsDrawerOpen(true);
      }
    }
  };

  const handleSelectIndicator = (val: string) => {
    setSelectedEvidenceQuery(val);
    setSelectedSignalCode(null);
    if (activeView !== 'evidence' && activeView !== 'all') {
      setActiveView('evidence');
    }
  };

  const handleSelectHeader = (headerName: string) => {
    setSelectedEvidenceQuery(headerName);
    setSelectedSignalCode(null);
    if (activeView !== 'evidence' && activeView !== 'all') {
      setActiveView('evidence');
    }
  };

  const handleClearEvidenceFilter = () => {
    setSelectedSignalCode(null);
    setSelectedEvidenceQuery(null);
  };

  const handleOpenDrawer = (item: EvidenceItem) => {
    setDrawerItem(item);
    setIsDrawerOpen(true);
  };

  const handleCloseDrawer = () => {
    setIsDrawerOpen(false);
    setDrawerItem(null);
  };

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
            {evidenceData && (
              <div className="workspace-summary-item" data-testid="summary-evidence-count">
                <span className="summary-label">Evidence records</span>
                <strong>{evidenceCount}</strong>
              </div>
            )}
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

      {/* Risk Assessment & Contributing Signals (Rendered when risk is provided) */}
      {analysisData?.risk && (
        <section className="risk-signals-card" data-testid="risk-signals-section">
          <div className="risk-card-header">
            <div className="risk-title-group">
              <span className="section-eyebrow">Risk Engine Assessment</span>
              <div className="risk-score-row">
                <h3 className="risk-card-title">Threat Assessment</h3>
                <span
                  className={`risk-score-badge level-${analysisData.risk.level?.toLowerCase()}`}
                  data-testid="risk-score-badge"
                >
                  Score: {analysisData.risk.score}/100 ({analysisData.risk.level?.toUpperCase()})
                </span>
                <span className="risk-verdict-pill" data-testid="risk-verdict-pill">
                  Verdict: {analysisData.risk.verdict}
                </span>
              </div>
            </div>
            <ProvenanceBadge classification="INFERRED" />
          </div>

          {riskSignals.length > 0 ? (
            <div className="contributing-signals-grid" data-testid="risk-signals-grid">
              {riskSignals.map((sig) => (
                <div key={sig.code} className="signal-card" data-testid={`risk-signal-${sig.code}`}>
                  <div className="signal-top">
                    <div className="signal-id-group">
                      <code className="signal-code">⚡ {sig.code}</code>
                      <span className="signal-points">+{sig.points} pts</span>
                      <span className="signal-category-tag">{sig.category}</span>
                    </div>
                    <ProvenanceBadge
                      classification={sig.provenance as EpistemicClass}
                      showIcon={false}
                      className="mini-badge"
                    />
                  </div>
                  <p className="signal-description">{sig.description}</p>
                  <button
                    type="button"
                    className="btn-signal-evidence"
                    onClick={() => handleSelectSignalCode(sig.code)}
                    data-testid={`signal-evidence-btn-${sig.code}`}
                    title={`View evidence corroborating ${sig.code}`}
                  >
                    View Supporting Evidence →
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <p className="no-signals-text" data-testid="no-signals-text">
              No elevated risk signals triggered.
            </p>
          )}
        </section>
      )}

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
          aria-selected={activeView === 'evidence'}
          className={`workspace-tab ${activeView === 'evidence' ? 'active' : ''}`}
          onClick={() => setActiveView('evidence')}
          data-testid="view-tab-evidence"
        >
          Evidence Vault ({evidenceCount})
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
        <button type="button" role="tab" aria-selected={activeView === 'timeline'} className={`workspace-tab ${activeView === 'timeline' ? 'active' : ''}`} onClick={() => setActiveView('timeline')} data-testid="view-tab-timeline">Relay Timeline</button>
        <button type="button" role="tab" aria-selected={activeView === 'graph'} className={`workspace-tab ${activeView === 'graph' ? 'active' : ''}`} onClick={() => setActiveView('graph')} data-testid="view-tab-graph">Entity Graph</button>
        <button type="button" role="tab" aria-selected={activeView === 'enrichment'} className={`workspace-tab ${activeView === 'enrichment' ? 'active' : ''}`} onClick={() => setActiveView('enrichment')} data-testid="view-tab-enrichment">Infrastructure</button>
        <button type="button" role="tab" aria-selected={activeView === 'report'} className={`workspace-tab ${activeView === 'report' ? 'active' : ''}`} onClick={() => setActiveView('report')} data-testid="view-tab-report">Report</button>
      </div>

      {/* Structured Sections */}
      <div className="workspace-content">
        {(activeView === 'assessment' || (activeView === 'all' && hasAnalysis)) && (
          <section className="workspace-section" data-testid="section-assessment">
            <AIAssessmentPanel
              analysisData={analysisData}
              isLoading={analysisLoading}
              error={analysisError}
              onSelectEvidenceReference={handleSelectEvidenceReference}
            />
          </section>
        )}

        {(activeView === 'evidence' || (activeView === 'all' && hasEvidence)) && (
          <section className="workspace-section" data-testid="section-evidence">
            <EvidencePanel
              evidenceData={evidenceData}
              isLoading={evidenceLoading}
              error={evidenceError}
              selectedSignalCode={selectedSignalCode}
              selectedQuery={selectedEvidenceQuery}
              onClearFilter={handleClearEvidenceFilter}
              onSelectSignalCode={handleSelectSignalCode}
              onSelectEvidenceItem={handleOpenDrawer}
              onRetry={onRetryEvidence}
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
            <IndicatorTable
              indicators={emailData.indicators}
              onSelectIndicator={handleSelectIndicator}
            />
          </section>
        )}

        {(activeView === 'all' || activeView === 'headers') && (
          <section className="workspace-section" data-testid="section-headers">
            <HeaderTable
              headers={emailData.headers}
              onSelectHeader={handleSelectHeader}
            />
          </section>
        )}

        {(activeView === 'all' || activeView === 'attachments') && (
          <section className="workspace-section" data-testid="section-attachments">
            <AttachmentList attachments={emailData.attachments} />
          </section>
        )}

        {(activeView === 'all' || activeView === 'timeline') && (
          <TimelinePanel data={timelineData} loading={timelineLoading} error={timelineError} />
        )}

        {(activeView === 'all' || activeView === 'graph') && (
          <GraphPanel data={graphData} loading={graphLoading} error={graphError} />
        )}

        {(activeView === 'all' || activeView === 'enrichment') && (
          <EnrichmentPanel values={analysisData?.ip_enrichment} loading={analysisLoading} error={analysisError} />
        )}

        {(activeView === 'all' || activeView === 'report') && (
          <ReportPanel data={reportData} loading={reportLoading} error={reportError} onGenerate={onGenerateReport ?? (() => undefined)} onDownloadPDF={onDownloadReportPDF} />
        )}

        {/* Pipeline Progression Notice */}
        <section className="pipeline-notice-box" data-testid="pipeline-notice">
          <div className="notice-icon" aria-hidden="true">ℹ</div>
          <div className="notice-content">
            <h4 className="notice-title">Stage 01: Ingestion & Parsing Complete</h4>
            <p className="notice-text">
              The RFC 5322 structure, headers, MIME boundaries, extracted network indicators,
              and attachment digests have been recorded in the case repository as <strong>OBSERVED</strong> facts.
              Authentication, relay reconstruction, enrichment, evidence, graph, timeline and report views are
              derived from the persisted analysis. External URLs were not visited and attachments were not executed.
            </p>
          </div>
        </section>
      </div>

      {/* Contextual Evidence Drawer */}
      <EvidenceDrawer
        isOpen={isDrawerOpen}
        onClose={handleCloseDrawer}
        evidenceItem={drawerItem}
        onSelectSignalCode={handleSelectSignalCode}
      />
    </div>
  );
};
