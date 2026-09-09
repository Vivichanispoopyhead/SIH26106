import React from 'react';
import './LoadingStage.css';

export type WorkflowState =
  | 'idle'
  | 'uploading'
  | 'uploaded'
  | 'starting_analysis'
  | 'processing'
  | 'parsed'
  | 'completed'
  | 'partial'
  | 'failed';

interface LoadingStageProps {
  state: WorkflowState;
  caseId?: string;
  emailId?: string;
  filename?: string;
  errorMessage?: string;
  onRetry?: () => void;
  onCancel?: () => void;
}

export const LoadingStage: React.FC<LoadingStageProps> = ({
  state,
  caseId,
  emailId,
  filename,
  errorMessage,
  onRetry,
  onCancel,
}) => {
  const getStepStatus = (stepIndex: number): 'done' | 'active' | 'pending' | 'failed' => {
    // Step 0: Upload / Ingest
    // Step 1: Case Creation & Preserve
    // Step 2: RFC 5322 Parse & Analysis
    if (state === 'failed') {
      return 'failed';
    }

    if (stepIndex === 0) {
      if (state === 'uploading') return 'active';
      return 'done';
    }

    if (stepIndex === 1) {
      if (state === 'uploading') return 'pending';
      if (state === 'uploaded' || state === 'starting_analysis') return 'active';
      return 'done';
    }

    if (stepIndex === 2) {
      if (state === 'uploading' || state === 'uploaded' || state === 'starting_analysis') {
        return 'pending';
      }
      if (state === 'processing') return 'active';
      return 'done';
    }

    return 'pending';
  };

  const getStatusText = (): string => {
    switch (state) {
      case 'uploading':
        return 'Ingesting RFC 5322 .eml sample to forensic backend...';
      case 'uploaded':
        return 'Sample uploaded. Initializing case & preserving raw artifact...';
      case 'starting_analysis':
        return 'Triggering automated parsing and indicator extraction pipeline...';
      case 'processing':
        return 'Parsing MIME structure, RFC 5322 headers, and indicators...';
      case 'partial':
        return 'Analysis completed with partial results.';
      case 'failed':
        return 'Forensic processing halted due to an error.';
      default:
        return 'Processing...';
    }
  };

  return (
    <div className="loading-stage-card" data-testid="loading-stage">
      <div className="loading-header">
        <div className="loading-title-group">
          <span className="analysis-phase-pill">Pipeline Stage 01: Ingest & Parse</span>
          <h3 className="loading-title">{getStatusText()}</h3>
        </div>
        <span className={`workflow-status-badge status-${state}`} data-testid="workflow-status-badge">
          {state.toUpperCase().replace('_', ' ')}
        </span>
      </div>

      {/* Forensic Step Tracker */}
      <div className="pipeline-tracker" data-testid="pipeline-tracker">
        <div className={`tracker-step ${getStepStatus(0)}`}>
          <span className="step-indicator">
            {getStepStatus(0) === 'done' ? '✓' : getStepStatus(0) === 'active' ? '⟳' : '1'}
          </span>
          <span className="step-label">Upload Sample</span>
        </div>

        <span className="tracker-divider">──</span>

        <div className={`tracker-step ${getStepStatus(1)}`}>
          <span className="step-indicator">
            {getStepStatus(1) === 'done' ? '✓' : getStepStatus(1) === 'active' ? '⟳' : '2'}
          </span>
          <span className="step-label">Case Ingestion</span>
        </div>

        <span className="tracker-divider">──</span>

        <div className={`tracker-step ${getStepStatus(2)}`}>
          <span className="step-indicator">
            {getStepStatus(2) === 'done' ? '✓' : getStepStatus(2) === 'active' ? '⟳' : '3'}
          </span>
          <span className="step-label">RFC 5322 Parsing</span>
        </div>
      </div>

      {/* Forensic identifiers */}
      {(caseId || emailId || filename) && (
        <div className="forensic-context-box">
          {filename && (
            <div className="context-item">
              <span className="context-label">Sample:</span>
              <code className="context-value">{filename}</code>
            </div>
          )}
          {caseId && (
            <div className="context-item">
              <span className="context-label">Case ID:</span>
              <code className="context-value" data-testid="loading-case-id">{caseId}</code>
            </div>
          )}
          {emailId && (
            <div className="context-item">
              <span className="context-label">Email ID:</span>
              <code className="context-value" data-testid="loading-email-id">{emailId}</code>
            </div>
          )}
        </div>
      )}

      {state === 'failed' && (
        <div className="loading-failure-box" role="alert">
          <p className="failure-message">{errorMessage || 'Analysis execution failed.'}</p>
          <div className="failure-actions">
            {onRetry && (
              <button
                type="button"
                className="btn-retry"
                onClick={onRetry}
                data-testid="retry-btn"
              >
                Retry Processing
              </button>
            )}
            {onCancel && (
              <button
                type="button"
                className="btn-cancel"
                onClick={onCancel}
                data-testid="cancel-btn"
              >
                Ingest Another Sample
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
