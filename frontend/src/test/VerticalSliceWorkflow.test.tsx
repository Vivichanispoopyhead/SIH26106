import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import App from '../App';

describe('Vertical Slice #1 End-to-End Workflow', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('F, G, H: executes full happy path from EML upload to parsed email display', async () => {
    // Mock health check first
    const mockFetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith('/api/health')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({ status: 'ok' }),
        });
      }

      // POST /api/emails
      if (url.endsWith('/api/emails') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            case_id: 'case_01J_WORKFLOW_TEST',
            email_id: 'email_01J_WORKFLOW_TEST',
            status: 'uploaded',
          }),
        });
      }

      // POST /api/emails/{email_id}/analysis
      if (url.includes('/analysis') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            analysis_id: 'analysis_01J_TEST',
            email_id: 'email_01J_WORKFLOW_TEST',
            case_id: 'case_01J_WORKFLOW_TEST',
            status: 'started',
          }),
        });
      }

      // GET /api/emails/{email_id}/analysis
      if (url.endsWith('/analysis') && (!init?.method || init?.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            analysis_id: 'analysis_01J_WORKFLOW_TEST',
            email_id: 'email_01J_WORKFLOW_TEST',
            case_id: 'case_01J_WORKFLOW_TEST',
            status: 'completed',
            ai_assessment: {
              status: 'completed',
              classification: 'Phishing',
              confidence: 0.95,
              supporting_signals: ['Suspicious sender domain', 'Account urgency pretext'],
              evidence_references: ['header:From', 'header:Subject'],
              provider: 'anthropic',
              model: 'claude-3-5-sonnet',
              failure: null,
            },
            failure: null,
          }),
        });
      }

      // GET /api/emails/{email_id}
      if (url.includes('/api/emails/email_01J_WORKFLOW_TEST')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            email_id: 'email_01J_WORKFLOW_TEST',
            case_id: 'case_01J_WORKFLOW_TEST',
            status: 'parsed',
            filename: 'threat_sample.eml',
            message: {
              message_id: '<threat@origin.com>',
              from: ['origin@attacker.net'],
              to: ['security@target.org'],
              cc: [],
              reply_to: ['bounce@attacker.net'],
              subject: 'Urgent Account Suspension Notice',
              date: '2026-09-08T14:30:00Z',
              return_path: 'bounce@attacker.net',
            },
            mime: {
              content_type: 'multipart/alternative',
              has_plain_text: true,
              has_html: true,
              attachment_count: 0,
            },
            headers: [
              { name: 'From', value: 'origin@attacker.net', order: 1 },
              { name: 'To', value: 'security@target.org', order: 2 },
              { name: 'Subject', value: 'Urgent Account Suspension Notice', order: 3 },
            ],
            indicators: {
              ips: ['198.51.100.22'],
              domains: ['attacker.net'],
              urls: ['https://attacker.net/login'],
            },
            attachments: [],
          }),
        });
      }

      return Promise.reject(new Error(`Unhandled URL: ${url}`));
    });

    vi.stubGlobal('fetch', mockFetch);

    render(<App />);

    // Initial state: Dropzone rendered
    expect(screen.getByTestId('eml-dropzone')).toBeInTheDocument();

    // Select valid .eml file
    const file = new File(['mock eml content'], 'threat_sample.eml', {
      type: 'message/rfc822',
    });
    const fileInput = screen.getByTestId('file-picker-input');
    fireEvent.change(fileInput, { target: { files: [file] } });

    // Step progression & Parsed email display
    await waitFor(() => {
      expect(screen.getByTestId('parsed-email-workspace')).toBeInTheDocument();
    });

    // Check parsed metadata rendered
    expect(screen.getByTestId('triage-subject')).toHaveTextContent('Urgent Account Suspension Notice');
    expect(screen.getAllByText('case_01J_WORKFLOW_TEST').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('email_01J_WORKFLOW_TEST').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('origin@attacker.net').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('198.51.100.22')).toBeInTheDocument();
    expect(screen.getByText('attacker.net')).toBeInTheDocument();

    // Canonical AI assessment panel is displayed with classification and confidence
    expect(screen.getByTestId('ai-assessment-panel')).toBeInTheDocument();
    expect(screen.getByTestId('ai-classification-value')).toHaveTextContent('Phishing');
    expect(screen.getByTestId('ai-confidence-value')).toHaveTextContent('95%');
  });

  it('E, G: renders analysis state progression during polling before parsed', async () => {
    let pollCount = 0;

    const mockFetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith('/api/health')) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ status: 'ok' }) });
      }

      if (url.endsWith('/api/emails') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            case_id: 'case_poll_1',
            email_id: 'email_poll_1',
            status: 'uploaded',
          }),
        });
      }

      if (url.includes('/analysis') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            email_id: 'email_poll_1',
            case_id: 'case_poll_1',
            status: 'started',
          }),
        });
      }

      if (url.endsWith('/analysis') && (!init?.method || init?.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            analysis_id: 'analysis_poll_1',
            email_id: 'email_poll_1',
            case_id: 'case_poll_1',
            status: 'completed',
            ai_assessment: {
              status: 'not_available',
              classification: null,
              confidence: null,
              supporting_signals: [],
              evidence_references: [],
              provider: null,
              model: null,
              failure: { code: 'AI_NOT_CONFIGURED', message: 'No AI analyzer configured' },
            },
            failure: null,
          }),
        });
      }

      if (url.includes('/api/emails/email_poll_1')) {
        pollCount++;
        if (pollCount === 1) {
          // First poll: still processing
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({
              email_id: 'email_poll_1',
              case_id: 'case_poll_1',
              status: 'processing',
            }),
          });
        }
        // Second poll: parsed
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            email_id: 'email_poll_1',
            case_id: 'case_poll_1',
            status: 'parsed',
            filename: 'poll.eml',
            message: { subject: 'Polled Email Subject' },
            mime: { content_type: 'text/plain', has_plain_text: true, has_html: false, attachment_count: 0 },
            headers: [],
            indicators: { ips: [], domains: [], urls: [] },
            attachments: [],
          }),
        });
      }

      return Promise.reject(new Error('Unknown endpoint'));
    });

    vi.stubGlobal('fetch', mockFetch);

    render(<App />);

    const file = new File(['content'], 'poll.eml', { type: 'message/rfc822' });
    fireEvent.change(screen.getByTestId('file-picker-input'), { target: { files: [file] } });

    // Eventually reaches parsed
    await waitFor(() => {
      expect(screen.getByTestId('parsed-email-workspace')).toBeInTheDocument();
    });

    expect(screen.getByTestId('triage-subject')).toHaveTextContent('Polled Email Subject');
    expect(pollCount).toBeGreaterThanOrEqual(2);
  });

  it('I: handles backend upload error (500 internal server error)', async () => {
    const mockFetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith('/api/health')) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ status: 'ok' }) });
      }

      if (url.endsWith('/api/emails') && init?.method === 'POST') {
        return Promise.resolve({
          ok: false,
          status: 500,
          json: async () => ({
            error: {
              code: 'INTERNAL_ERROR',
              message: 'Failed to write artifact to disk.',
            },
          }),
        });
      }

      return Promise.reject(new Error('Unknown'));
    });

    vi.stubGlobal('fetch', mockFetch);

    render(<App />);

    const file = new File(['content'], 'sample.eml', { type: 'message/rfc822' });
    fireEvent.change(screen.getByTestId('file-picker-input'), { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByTestId('error-banner')).toBeInTheDocument();
    });

    expect(screen.getByTestId('error-code-badge')).toHaveTextContent('INTERNAL_ERROR');
  });

  it('J: handles parse failure (422 EMAIL_PARSE_FAILED) gracefully', async () => {
    const mockFetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith('/api/health')) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ status: 'ok' }) });
      }

      if (url.endsWith('/api/emails') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            case_id: 'case_parse_fail',
            email_id: 'email_parse_fail',
            status: 'uploaded',
          }),
        });
      }

      if (url.includes('/analysis') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            email_id: 'email_parse_fail',
            case_id: 'case_parse_fail',
            status: 'started',
          }),
        });
      }

      if (url.includes('/api/emails/email_parse_fail')) {
        return Promise.resolve({
          ok: false,
          status: 422,
          json: async () => ({
            error: {
              code: 'EMAIL_PARSE_FAILED',
              message: 'RFC 5322 syntax malformed.',
            },
          }),
        });
      }

      return Promise.reject(new Error('Unknown'));
    });

    vi.stubGlobal('fetch', mockFetch);

    render(<App />);

    const file = new File(['corrupt data'], 'corrupt.eml', { type: 'message/rfc822' });
    fireEvent.change(screen.getByTestId('file-picker-input'), { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByTestId('error-banner')).toBeInTheDocument();
    });

    expect(screen.getByTestId('error-code-badge')).toHaveTextContent('EMAIL_PARSE_FAILED');
    expect(screen.getByText(/The forensic engine could not parse/i)).toBeInTheDocument();
  });

  it('K: handles retry behavior on failure', async () => {
    let uploadAttempts = 0;

    const mockFetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.endsWith('/api/health')) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ status: 'ok' }) });
      }

      if (url.endsWith('/api/emails') && init?.method === 'POST') {
        uploadAttempts++;
        if (uploadAttempts === 1) {
          return Promise.resolve({
            ok: false,
            status: 500,
            json: async () => ({
              error: { code: 'INTERNAL_ERROR', message: 'Transient storage failure' },
            }),
          });
        }
        // Second attempt succeeds
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            case_id: 'case_retry_success',
            email_id: 'email_retry_success',
            status: 'uploaded',
          }),
        });
      }

      if (url.includes('/analysis') && init?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            email_id: 'email_retry_success',
            case_id: 'case_retry_success',
            status: 'started',
          }),
        });
      }

      if (url.endsWith('/analysis') && (!init?.method || init?.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            analysis_id: 'analysis_retry_1',
            email_id: 'email_retry_success',
            case_id: 'case_retry_success',
            status: 'completed',
            ai_assessment: {
              status: 'completed',
              classification: 'Suspicious',
              confidence: 0.88,
              supporting_signals: ['Domain age indicator'],
              evidence_references: ['header:From'],
              provider: 'anthropic',
              model: 'claude-3-5-sonnet',
              failure: null,
            },
            failure: null,
          }),
        });
      }

      if (url.includes('/api/emails/email_retry_success')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            email_id: 'email_retry_success',
            case_id: 'case_retry_success',
            status: 'parsed',
            filename: 'retry.eml',
            message: { subject: 'Retry Success Subject' },
            mime: { content_type: 'text/plain', has_plain_text: true, has_html: false, attachment_count: 0 },
            headers: [],
            indicators: { ips: [], domains: [], urls: [] },
            attachments: [],
          }),
        });
      }

      return Promise.reject(new Error('Unknown'));
    });

    vi.stubGlobal('fetch', mockFetch);

    render(<App />);

    const file = new File(['content'], 'retry.eml', { type: 'message/rfc822' });
    fireEvent.change(screen.getByTestId('file-picker-input'), { target: { files: [file] } });

    // First attempt fails
    await waitFor(() => {
      expect(screen.getByTestId('error-banner')).toBeInTheDocument();
    });

    // Click retry
    const retryBtn = screen.getByTestId('error-retry-btn');
    fireEvent.click(retryBtn);

    // Second attempt succeeds and reaches parsed
    await waitFor(() => {
      expect(screen.getByTestId('parsed-email-workspace')).toBeInTheDocument();
    });

    expect(screen.getByTestId('triage-subject')).toHaveTextContent('Retry Success Subject');
    expect(uploadAttempts).toBe(2);
  });
});
