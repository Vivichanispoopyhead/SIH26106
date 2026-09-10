import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  fetchBackendHealth,
  HealthResponse,
  uploadEmail,
  startAnalysis,
  getEmail,
  getAnalysis,
  ApiError,
} from './services/api';
import { API_BASE_URL } from './config/env';
import { ParsedEmailResponse, EmailAnalysisResponse } from './types/api';
import { AppHeader } from './components/shell/AppHeader';
import { AppFooter } from './components/shell/AppFooter';
import { NavigationSidebar } from './components/shell/NavigationSidebar';
import { EmlUploadZone } from './components/ingestion/EmlUploadZone';
import { LoadingStage, WorkflowState } from './components/states/LoadingStage';
import { ErrorBanner } from './components/states/ErrorBanner';
import { ParsedEmailWorkspace } from './components/investigation/ParsedEmailWorkspace';
import { MailSearch } from 'lucide-react';
import './App.css';

type BackendStatus = 'checking' | 'online' | 'error';

export const App: React.FC = () => {
  // Backend health status
  const [backendStatus, setBackendStatus] = useState<BackendStatus>('checking');
  const [healthData, setHealthData] = useState<HealthResponse | null>(null);
  const [healthErrorMessage, setHealthErrorMessage] = useState<string | null>(null);
  const [lastChecked, setLastChecked] = useState<string | null>(null);

  // Workflow state for Vertical Slice #1
  const [workflowState, setWorkflowState] = useState<WorkflowState>('idle');
  const [caseId, setCaseId] = useState<string | undefined>(undefined);
  const [emailId, setEmailId] = useState<string | undefined>(undefined);
  const [filename, setFilename] = useState<string | undefined>(undefined);
  const [parsedEmail, setParsedEmail] = useState<ParsedEmailResponse | null>(null);
  const [analysisData, setAnalysisData] = useState<EmailAnalysisResponse | null>(null);
  const [analysisLoading, setAnalysisLoading] = useState<boolean>(false);
  const [analysisError, setAnalysisError] = useState<unknown | null>(null);
  const [currentFile, setCurrentFile] = useState<File | null>(null);
  const [error, setError] = useState<unknown | null>(null);

  const pollingRef = useRef<boolean>(false);
  const uploadInProgress = workflowState === 'uploading';

  // Health check handler
  const checkHealth = useCallback(async () => {
    setBackendStatus('checking');
    setHealthErrorMessage(null);

    try {
      const data = await fetchBackendHealth();
      setHealthData(data);
      setBackendStatus('online');
      setLastChecked(new Date().toLocaleTimeString());
    } catch (err) {
      setHealthData(null);
      setBackendStatus('error');
      setLastChecked(new Date().toLocaleTimeString());
      if (err instanceof ApiError) {
        setHealthErrorMessage(err.message);
      } else if (err instanceof Error) {
        setHealthErrorMessage(err.message);
      } else {
        setHealthErrorMessage('An unexpected error occurred while contacting the backend.');
      }
    }
  }, []);

  useEffect(() => {
    checkHealth();
  }, [checkHealth]);

  // Clean up any in-flight polling on unmount
  useEffect(() => {
    return () => {
      pollingRef.current = false;
    };
  }, []);

  // Poll for parsed email status
  const pollForParsedEmail = async (id: string): Promise<ParsedEmailResponse> => {
    pollingRef.current = true;
    const maxAttempts = 30; // 30 * 800ms ≈ 24 seconds
    const intervalMs = 800;

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      if (!pollingRef.current) {
        throw new Error('Analysis polling cancelled');
      }

      const emailData = await getEmail(id);

      if (emailData.status === 'parsed') {
        return emailData;
      }

      if (emailData.status === 'failed') {
        throw new ApiError('The email parser could not process this sample.', 'EMAIL_PARSE_FAILED', 422);
      }

      if (emailData.status === 'partial') {
        return emailData;
      }

      // If status is 'processing', 'uploaded', or 'started', wait and poll again
      await new Promise((res) => setTimeout(res, intervalMs));
    }

    throw new ApiError('Analysis timed out waiting for parser completion.', 'ANALYSIS_FAILED', 504);
  };

  // Main EML Ingestion & Analysis Workflow
  const handleFileSelected = async (file: File) => {
    setError(null);
    setCurrentFile(file);
    setFilename(file.name);
    setWorkflowState('uploading');

    try {
      // Step 1: Upload .EML -> POST /api/emails
      const uploadRes = await uploadEmail(file);
      setCaseId(uploadRes.case_id);
      setEmailId(uploadRes.email_id);
      setWorkflowState('uploaded');

      // Step 2: Start Analysis -> POST /api/emails/{email_id}/analysis
      setWorkflowState('starting_analysis');
      await startAnalysis(uploadRes.email_id);

      // Step 3: Wait for analysis & Fetch Parsed Email -> GET /api/emails/{email_id}
      setWorkflowState('processing');
      const emailResult = await pollForParsedEmail(uploadRes.email_id);
      setParsedEmail(emailResult);

      // Step 4: Fetch Canonical Analysis Result -> GET /api/emails/{email_id}/analysis
      setAnalysisLoading(true);
      try {
        const analysisResult = await getAnalysis(uploadRes.email_id);
        setAnalysisData(analysisResult);
      } catch (analysisErr) {
        setAnalysisError(analysisErr);
      } finally {
        setAnalysisLoading(false);
      }

      setWorkflowState('parsed');
    } catch (err) {
      pollingRef.current = false;
      setError(err);
      setWorkflowState('failed');
    }
  };

  const handleRetry = () => {
    if (currentFile) {
      handleFileSelected(currentFile);
    } else {
      handleReset();
    }
  };

  const handleReset = () => {
    pollingRef.current = false;
    setWorkflowState('idle');
    setCaseId(undefined);
    setEmailId(undefined);
    setFilename(undefined);
    setParsedEmail(null);
    setAnalysisData(null);
    setAnalysisLoading(false);
    setAnalysisError(null);
    setCurrentFile(null);
    setError(null);
  };

  return (
    <div className="app-shell" data-testid="app-shell">
      {/* Top Bar */}
      <AppHeader
        caseId={caseId}
        emailId={emailId}
        onResetCase={handleReset}
      />

      {/* Connectivity & Diagnostic Ribbon */}
      <section className="diagnostic-ribbon" data-testid="diagnostic-ribbon">
        <div className="diagnostic-left">
          <span className="app-title-mini">SIH26106 Email Threat Detection</span>
          <span className="status-pill online" data-testid="frontend-status-badge">
            <span className="status-dot" />
            Operational
          </span>
        </div>

        <div className="diagnostic-right">
          <div className="backend-conn-status" data-testid="backend-status-card">
            <span className="diagnostic-label">Backend:</span>
            {backendStatus === 'checking' && (
              <span className="status-pill checking" data-testid="backend-status-badge">
                <span className="status-dot pulsing" />
                Checking
              </span>
            )}
            {backendStatus === 'online' && (
              <span className="status-pill online" data-testid="backend-status-badge">
                <span className="status-dot" />
                Online
              </span>
            )}
            {backendStatus === 'error' && (
              <span className="status-pill error" data-testid="backend-status-badge">
                <span className="status-dot" />
                Unavailable
              </span>
            )}

            {backendStatus === 'online' && healthData && (
              <span className="health-status-text" data-testid="backend-health-status">
                ({healthData.status})
              </span>
            )}

            {backendStatus === 'error' && healthErrorMessage && (
              <span className="health-error-text" data-testid="backend-error-message">
                {healthErrorMessage}
              </span>
            )}

            {lastChecked && (
              <span className="diagnostic-label" style={{ fontSize: '10px' }}>
                {lastChecked}
              </span>
            )}

            <button
              type="button"
              className="btn-refresh-health"
              onClick={checkHealth}
              disabled={backendStatus === 'checking'}
              data-testid="refresh-health-btn"
              title="Re-check backend API health"
            >
              {backendStatus === 'checking' ? '⟳' : 'Check API'}
            </button>
          </div>
        </div>
      </section>

      <div className="application-body">
        <NavigationSidebar
          activeStage={workflowState === 'idle' ? 'ingest' : 'investigation'}
          hasEmail={Boolean(emailId)}
          onNewAnalysis={handleReset}
        />

        {/* Main Workspace */}
        <main className="main-content-viewport">
        {error != null && (
          <div className="error-container">
            <ErrorBanner
              error={error}
              onRetry={handleRetry}
              onDismiss={() => setError(null)}
            />
          </div>
        )}

        {workflowState === 'idle' && (
          <section className="ingestion-section" data-testid="ingestion-section">
            <div className="hero-icon-badge">
              <MailSearch size={30} strokeWidth={2} />
            </div>

            <div className="ingestion-hero">
              <h2 className="hero-heading">Submit an .eml sample to open an investigation</h2>
              <p className="hero-description">
                Upload an RFC 5322 .eml artifact. The sample will be immutably preserved, an investigation case
                will be automatically initialized, and the MIME structure, headers, and indicators will be extracted.
              </p>
            </div>

            <div className="upload-zone-wrapper">
              <EmlUploadZone
                onFileSelected={handleFileSelected}
                disabled={backendStatus === 'checking'}
                isUploading={uploadInProgress}
              />
            </div>
          </section>
        )}

        {workflowState !== 'idle' && workflowState !== 'parsed' && (
          <section className="loading-stage-section">
            <LoadingStage
              state={workflowState}
              caseId={caseId}
              emailId={emailId}
              filename={filename}
              errorMessage={
                error instanceof Error ? error.message : typeof error === 'string' ? error : undefined
              }
              onRetry={handleRetry}
              onCancel={handleReset}
            />
          </section>
        )}

        {workflowState === 'parsed' && parsedEmail && (
          <section className="workspace-stage-section">
            <ParsedEmailWorkspace
              emailData={parsedEmail}
              analysisData={analysisData}
              analysisLoading={analysisLoading}
              analysisError={analysisError}
            />
          </section>
        )}
        </main>
      </div>

      {/* Footer */}
      <AppFooter apiBaseUrl={API_BASE_URL} />
    </div>
  );
};

export default App;
