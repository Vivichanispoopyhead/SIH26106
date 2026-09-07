import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import App from '../App';

describe('App Component', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders application title and frontend operational status', () => {
    // Keep request pending
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => new Promise(() => {}))
    );

    render(<App />);

    expect(screen.getByText('SIH26106 Email Threat Detection')).toBeInTheDocument();
    expect(screen.getByTestId('frontend-status-badge')).toHaveTextContent('Operational');
  });

  it('displays checking status while backend health check is in flight', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => new Promise(() => {}))
    );

    render(<App />);

    expect(screen.getByTestId('backend-status-badge')).toHaveTextContent('Checking');
  });

  it('displays backend online status and payload when GET /api/health succeeds', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ status: 'ok' }),
      })
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('backend-status-badge')).toHaveTextContent('Online');
    });

    expect(screen.getByTestId('backend-health-status')).toHaveTextContent('ok');
  });

  it('displays unavailable error state when backend responds with non-200 error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({
          error: { code: 'INTERNAL_ERROR', message: 'Database connection failed' },
        }),
      })
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('backend-status-badge')).toHaveTextContent('Unavailable');
    });

    expect(screen.getByTestId('backend-error-message')).toHaveTextContent('Database connection failed');
  });

  it('displays unavailable error state on connection refused or network failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new Error('Connection refused'))
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('backend-status-badge')).toHaveTextContent('Unavailable');
    });

    expect(screen.getByTestId('backend-error-message')).toHaveTextContent('Connection refused');
  });

  it('re-checks backend health when user clicks the retry button', async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error('Connection refused'))
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ status: 'ok' }),
      });

    vi.stubGlobal('fetch', fetchMock);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('backend-status-badge')).toHaveTextContent('Unavailable');
    });

    const refreshBtn = screen.getByTestId('refresh-health-btn');
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(screen.getByTestId('backend-status-badge')).toHaveTextContent('Online');
    });

    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
