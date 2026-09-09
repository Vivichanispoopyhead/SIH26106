import React from 'react';
import { getFriendlyErrorMessage } from '../../services/api';
import './ErrorBanner.css';

interface ErrorBannerProps {
  error: unknown;
  onRetry?: () => void;
  onDismiss?: () => void;
}

export const ErrorBanner: React.FC<ErrorBannerProps> = ({
  error,
  onRetry,
  onDismiss,
}) => {
  if (!error) return null;

  const { title, message, code } = getFriendlyErrorMessage(error);

  return (
    <div className="error-banner-card" role="alert" data-testid="error-banner">
      <div className="error-banner-header">
        <div className="error-title-row">
          <span className="error-icon" aria-hidden="true">⚠</span>
          <h4 className="error-title">{title}</h4>
          <span className="error-code-badge" data-testid="error-code-badge">
            {code}
          </span>
        </div>

        {onDismiss && (
          <button
            type="button"
            className="error-dismiss-btn"
            onClick={onDismiss}
            aria-label="Dismiss error"
          >
            ×
          </button>
        )}
      </div>

      <p className="error-banner-message">{message}</p>

      {onRetry && (
        <div className="error-actions">
          <button
            type="button"
            className="btn-error-retry"
            onClick={onRetry}
            data-testid="error-retry-btn"
          >
            Retry Action
          </button>
        </div>
      )}
    </div>
  );
};
