import React, { useState, useEffect, useCallback } from 'react';
import { fetchBackendHealth, HealthResponse, ApiError } from './services/api';
import { API_BASE_URL } from './config/env';
import './App.css';

type BackendStatus = 'checking' | 'online' | 'error';

export const App: React.FC = () => {
  const [status, setStatus] = useState<BackendStatus>('checking');
  const [healthData, setHealthData] = useState<HealthResponse | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [lastChecked, setLastChecked] = useState<string | null>(null);

  const checkHealth = useCallback(async () => {
    setStatus('checking');
    setErrorMessage(null);

    try {
      const data = await fetchBackendHealth();
      setHealthData(data);
      setStatus('online');
      setLastChecked(new Date().toLocaleTimeString());
    } catch (err) {
      setHealthData(null);
      setStatus('error');
      setLastChecked(new Date().toLocaleTimeString());
      if (err instanceof ApiError) {
        setErrorMessage(err.message);
      } else if (err instanceof Error) {
        setErrorMessage(err.message);
      } else {
        setErrorMessage('An unexpected error occurred while contacting the backend.');
      }
    }
  }, []);

  useEffect(() => {
    checkHealth();
  }, [checkHealth]);

  return (
    <div className="app-container">
      <header className="app-header">
        <span className="brand-badge">Forensic Platform MVP</span>
        <h1 className="app-title">SIH26106 Email Threat Detection</h1>
        <p className="app-subtitle">
          AI-Powered Email Threat Intelligence & Forensic Analysis Platform
        </p>
      </header>

      <main className="grid-cards">
        {/* Frontend Status Card */}
        <section className="status-card" data-testid="frontend-status-card">
          <div className="card-header">
            <h2 className="card-title">Frontend Shell</h2>
            <span className="status-pill online" data-testid="frontend-status-badge">
              <span className="status-dot" />
              Operational
            </span>
          </div>

          <div className="info-rows">
            <div className="info-row">
              <span className="info-label">Framework</span>
              <span className="info-value">React 19 + TypeScript</span>
            </div>
            <div className="info-row">
              <span className="info-label">Dev Server</span>
              <span className="info-value mono">http://localhost:5173</span>
            </div>
            <div className="info-row">
              <span className="info-label">Role</span>
              <span className="info-value">Forensic UI & Visualization</span>
            </div>
          </div>
        </section>

        {/* Backend Connection Card */}
        <section className="status-card" data-testid="backend-status-card">
          <div className="card-header">
            <h2 className="card-title">Backend Connection</h2>
            {status === 'checking' && (
              <span className="status-pill checking" data-testid="backend-status-badge">
                <span className="status-dot pulsing" />
                Checking
              </span>
            )}
            {status === 'online' && (
              <span className="status-pill online" data-testid="backend-status-badge">
                <span className="status-dot" />
                Online
              </span>
            )}
            {status === 'error' && (
              <span className="status-pill error" data-testid="backend-status-badge">
                <span className="status-dot" />
                Unavailable
              </span>
            )}
          </div>

          <div className="info-rows">
            <div className="info-row">
              <span className="info-label">Target API URL</span>
              <span className="info-value mono">{API_BASE_URL}</span>
            </div>
            <div className="info-row">
              <span className="info-label">Health Endpoint</span>
              <span className="info-value mono">GET /api/health</span>
            </div>

            {status === 'online' && healthData && (
              <div className="info-row" data-testid="backend-health-status">
                <span className="info-label">Reported Status</span>
                <span className="info-value mono">{healthData.status}</span>
              </div>
            )}

            {lastChecked && (
              <div className="info-row">
                <span className="info-label">Last Checked</span>
                <span className="info-value">{lastChecked}</span>
              </div>
            )}
          </div>

          {status === 'error' && errorMessage && (
            <div className="error-banner" data-testid="backend-error-message" role="alert">
              <strong>Connection Error:</strong> {errorMessage}
            </div>
          )}

          <div className="card-actions">
            <button
              className="btn-refresh"
              onClick={checkHealth}
              disabled={status === 'checking'}
              data-testid="refresh-health-btn"
            >
              {status === 'checking' ? 'Checking Health...' : 'Check Connection'}
            </button>
          </div>
        </section>
      </main>

      <footer className="app-footer">
        <span>SIH26106 Architecture Compliance: REST /api contract</span>
        <span>Agy-CLI Frontend &bull; Codex-CLI Backend</span>
      </footer>
    </div>
  );
};

export default App;
