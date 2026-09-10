import React from 'react';
import { EmailAnalysisResponse, AIAssessmentStatus, AnalysisStatus } from '../../../types/api';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import './AIAssessmentPanel.css';

export interface AIAssessmentPanelProps {
  analysisData?: EmailAnalysisResponse | null;
  isLoading?: boolean;
  error?: unknown | null;
}

/**
 * Formats assessment confidence into an integer percentage.
 * Returns null if confidence is missing, null, or undefined.
 */
export function formatConfidencePercentage(confidence: number | null | undefined): string | null {
  if (confidence == null || isNaN(confidence)) {
    return null;
  }
  const pct = confidence >= 0 && confidence <= 1 ? confidence * 100 : confidence;
  return `${Math.round(pct)}%`;
}

export const AIAssessmentPanel: React.FC<AIAssessmentPanelProps> = ({
  analysisData,
  isLoading = false,
  error = null,
}) => {
  // 1. Loading State
  if (isLoading) {
    return (
      <div className="ai-assessment-card ai-loading-state" data-testid="ai-assessment-loading">
        <div className="ai-card-header">
          <div className="header-left">
            <span className="ai-card-eyebrow">Forensic Pipeline Stage</span>
            <h3 className="ai-card-title">AI Threat Assessment</h3>
          </div>
          <ProvenanceBadge classification="AI-ASSESSED" />
        </div>
        <div className="loading-body">
          <span className="status-dot pulsing" aria-hidden="true" />
          <span className="loading-text">
            Evaluating semantic signals, intent indicators, and RFC 5322 references...
          </span>
        </div>
      </div>
    );
  }

  // 2. Direct API / Client Error State
  if (error) {
    const errorMessage: string =
      error instanceof Error
        ? error.message
        : typeof error === 'string'
          ? error
          : typeof error === 'object' && error !== null && 'message' in error
            ? String((error as { message: unknown }).message)
            : 'Failed to retrieve analysis';
    const errorCode =
      typeof error === 'object' && error !== null && 'code' in error
        ? String((error as { code: unknown }).code)
        : 'ANALYSIS_ERROR';

    return (
      <div className="ai-assessment-card ai-warning-state" data-testid="ai-assessment-warning">
        <div className="ai-card-header">
          <div className="header-left">
            <span className="ai-card-eyebrow">Analysis Pipeline Notice</span>
            <h3 className="ai-card-title">AI Assessment Unavailable</h3>
          </div>
          <div className="header-right">
            <span className="status-pill warning" data-testid="pipeline-status">
              pipeline: partial
            </span>
            <ProvenanceBadge classification="AI-ASSESSED" />
          </div>
        </div>
        <div className="warning-body">
          <div className="warning-banner">
            <span className="warning-icon" aria-hidden="true">⚠</span>
            <div className="warning-content">
              <h4 className="warning-title">Analysis Retrieval Incomplete</h4>
              <p className="warning-message">
                <code className="error-code-badge" data-testid="error-code-badge">[{errorCode}]</code> {errorMessage}
              </p>
              <p className="evidence-retention-note">
                Deterministic email parsing succeeded. RFC 5322 headers, MIME boundaries, and network indicators
                remain preserved and available for forensic review.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 3. No data available yet
  if (!analysisData) {
    return (
      <div className="ai-assessment-card ai-neutral-state" data-testid="ai-assessment-empty">
        <div className="ai-card-header">
          <div className="header-left">
            <span className="ai-card-eyebrow">Forensic Pipeline Stage</span>
            <h3 className="ai-card-title">AI Threat Assessment</h3>
          </div>
          <ProvenanceBadge classification="AI-ASSESSED" />
        </div>
        <div className="neutral-body">
          <p className="neutral-text">
            No canonical analysis record is currently available for this email artifact.
          </p>
        </div>
      </div>
    );
  }

  const pipelineStatus: AnalysisStatus = analysisData.status;
  const assessment = analysisData.ai_assessment;
  const aiStatus: AIAssessmentStatus = assessment?.status ?? 'not_available';
  const failure = assessment?.failure || analysisData.failure;
  const isNotConfigured =
    aiStatus === 'not_available' || failure?.code === 'AI_NOT_CONFIGURED';
  const isFailedOrPartial =
    pipelineStatus === 'partial' ||
    pipelineStatus === 'failed' ||
    aiStatus === 'failed' ||
    aiStatus === 'partial';

  // 4. Status "not_available" and code "AI_NOT_CONFIGURED" (Requirement 6)
  if (isNotConfigured) {
    return (
      <div className="ai-assessment-card ai-neutral-state" data-testid="ai-assessment-unavailable">
        <div className="ai-card-header">
          <div className="header-left">
            <span className="ai-card-eyebrow">Evaluated Assessment</span>
            <h3 className="ai-card-title">AI Assessment Unavailable</h3>
          </div>
          <div className="header-right">
            <span className="status-pill neutral" data-testid="pipeline-status">
              pipeline: {pipelineStatus}
            </span>
            <span className="status-pill neutral" data-testid="ai-assessment-status">
              ai: {aiStatus}
            </span>
            <ProvenanceBadge classification="AI-ASSESSED" />
          </div>
        </div>

        <div className="neutral-body">
          <div className="neutral-banner">
            <span className="neutral-icon" aria-hidden="true">ℹ</span>
            <div className="neutral-content">
              <h4 className="neutral-title">AI Analyzer Not Configured</h4>
              <p className="neutral-description">
                No AI or machine learning analyzer is configured for this environment.
                Deterministic RFC 5322 parsing, header analysis, MIME boundary inspection, and network
                indicator extraction remain fully operational and verified.
              </p>
              {failure?.message && (
                <div className="neutral-detail">
                  <span className="detail-label">Provider Notice:</span>
                  <span className="detail-value">{failure.message}</span>
                </div>
              )}
              <p className="retention-assurance">
                All observed email evidence remains preserved in the case repository.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 5. Status "failed" or overall status "partial" (Requirement 7)
  if (isFailedOrPartial && aiStatus !== 'completed') {
    return (
      <div className="ai-assessment-card ai-warning-state" data-testid="ai-assessment-warning">
        <div className="ai-card-header">
          <div className="header-left">
            <span className="ai-card-eyebrow">Evaluated Assessment · Non-Fatal Status</span>
            <h3 className="ai-card-title">AI Assessment Incomplete</h3>
          </div>
          <div className="header-right">
            <span className="status-pill warning" data-testid="pipeline-status">
              pipeline: {pipelineStatus}
            </span>
            <span className="status-pill error" data-testid="ai-assessment-status">
              ai: {aiStatus}
            </span>
            <ProvenanceBadge classification="AI-ASSESSED" />
          </div>
        </div>

        <div className="warning-body">
          <div className="warning-banner">
            <span className="warning-icon" aria-hidden="true">⚠</span>
            <div className="warning-content">
              <h4 className="warning-title">Non-Fatal Analyzer Failure</h4>
              <p className="warning-description">
                The AI assessment stage could not complete evaluation.
                {failure?.message ? ` ${failure.message}` : ''}
              </p>
              {failure?.code && (
                <div className="warning-detail">
                  <span className="detail-label">Failure Code:</span>
                  <code className="error-code-badge">[{failure.code}]</code>
                </div>
              )}
              <p className="evidence-retention-note">
                Deterministic email parsing succeeded. RFC 5322 headers, MIME parts, and network indicators
                remain preserved and available for forensic review.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 6. Completed AI Assessment (Requirements 5, 8, 9)
  const classification = assessment?.classification;
  const confidencePct = formatConfidencePercentage(assessment?.confidence);
  const supportingSignals = assessment?.supporting_signals ?? [];
  const evidenceReferences = assessment?.evidence_references ?? [];
  const provider = assessment?.provider;
  const model = assessment?.model;

  return (
    <div className="ai-assessment-card ai-success-state" data-testid="ai-assessment-panel">
      {/* Card Header */}
      <div className="ai-card-header">
        <div className="header-left">
          <span className="ai-card-eyebrow">Intent & Semantic Assessment</span>
          <h3 className="ai-card-title">AI Threat Assessment</h3>
          <span className="ai-card-disclaimer">
            Evaluated Assessment · Traceable to Observed Evidence
          </span>
        </div>
        <div className="header-right">
          <span className="status-pill success" data-testid="pipeline-status">
            pipeline: {pipelineStatus}
          </span>
          <span className="status-pill success" data-testid="ai-assessment-status">
            ai: {aiStatus}
          </span>
          <ProvenanceBadge classification="AI-ASSESSED" />
        </div>
      </div>

      {/* Primary Metrics Grid */}
      <div className="ai-metrics-grid">
        {/* Classification - only when present */}
        {classification && (
          <div className="ai-metric-tile" data-testid="ai-classification-metric">
            <span className="tile-label">CLASSIFICATION</span>
            <span className="tile-value classification-highlight" data-testid="ai-classification-value">
              {classification}
            </span>
            <span className="tile-sublabel">Evaluated Assessment</span>
          </div>
        )}

        {/* Confidence as percentage - only when present (Requirement 5 & 8) */}
        {confidencePct && (
          <div className="ai-metric-tile" data-testid="ai-confidence-metric">
            <span className="tile-label">ASSESSMENT CONFIDENCE</span>
            <div className="confidence-value-container">
              <span className="tile-value confidence-highlight" data-testid="ai-confidence-value">
                {confidencePct}
              </span>
            </div>
            <div className="confidence-meter-bar" aria-hidden="true">
              <div
                className="confidence-fill"
                style={{ width: confidencePct }}
              />
            </div>
            <span className="tile-sublabel">Model Confidence (Not Threat Severity)</span>
          </div>
        )}

        {/* Provider / Model - only when present (Requirement 5) */}
        {(provider || model) && (
          <div className="ai-metric-tile" data-testid="ai-provider-model-metric">
            <span className="tile-label">INFERENCE ENGINE</span>
            <span className="tile-value mono-compact" data-testid="ai-provider-model-value">
              {provider && model ? `${provider} / ${model}` : (provider || model)}
            </span>
            <span className="tile-sublabel">Provider & Model</span>
          </div>
        )}

        {/* Analysis Identifier */}
        <div className="ai-metric-tile" data-testid="ai-pipeline-metric">
          <span className="tile-label">ANALYSIS ID</span>
          <span className="tile-value mono-compact" data-testid="ai-analysis-id">
            <MonoValue value={analysisData.analysis_id} label="Analysis ID" copyable={true} />
          </span>
          <span className="tile-sublabel">Canonical Record</span>
        </div>
      </div>

      {/* Supporting Signals */}
      <div className="ai-signals-section" data-testid="ai-signals-section">
        <div className="section-subtitle-bar">
          <h4 className="section-subtitle">
            Supporting Signals
            <span className="count-tag" data-testid="ai-signals-count">{supportingSignals.length}</span>
          </h4>
          <span className="section-hint">Semantic, syntactic, and structural signals extracted during assessment</span>
        </div>
        {supportingSignals.length > 0 ? (
          <ul className="signals-list" data-testid="ai-signals-list">
            {supportingSignals.map((signal, idx) => (
              <li key={idx} className="signal-item" data-testid="ai-signal-item">
                <span className="signal-bullet" aria-hidden="true">✦</span>
                <span className="signal-text">{signal}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="empty-signals-text" data-testid="ai-signals-empty">
            No supporting signals reported.
          </p>
        )}
      </div>

      {/* Evidence References */}
      <div className="ai-evidence-section" data-testid="ai-evidence-section">
        <div className="section-subtitle-bar">
          <h4 className="section-subtitle">
            Evidence References
            <span className="count-tag" data-testid="ai-evidence-count">{evidenceReferences.length}</span>
          </h4>
          <span className="section-hint">Observed RFC 5322 artifacts corroborating this assessment</span>
        </div>
        {evidenceReferences.length > 0 ? (
          <div className="evidence-chips-grid" data-testid="ai-evidence-list">
            {evidenceReferences.map((ref, idx) => (
              <div key={idx} className="evidence-chip" data-testid="ai-evidence-item">
                <span className="evidence-dot" aria-hidden="true">●</span>
                <code className="evidence-ref-name">{ref}</code>
                <ProvenanceBadge classification="OBSERVED" showIcon={false} className="mini-badge" />
              </div>
            ))}
          </div>
        ) : (
          <p className="empty-evidence-text" data-testid="ai-evidence-empty">
            No evidence references linked.
          </p>
        )}
      </div>

      {/* Epistemic Integrity Footer */}
      <div className="ai-card-footer" data-testid="ai-card-footer">
        <span className="footer-glyph" aria-hidden="true">⚖</span>
        <span className="footer-disclaimer">
          <strong>Forensic Integrity Notice:</strong> AI assessment describes evaluated model intent; it does not constitute confirmed ground truth or legal attribution. Corroborate all findings against observed header and cryptographic evidence. Risk scoring remains distinct from analytical confidence.
        </span>
      </div>
    </div>
  );
};
