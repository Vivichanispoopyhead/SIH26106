import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { getAnalysis, getFriendlyErrorMessage, ApiError } from '../services/api';

describe('Analysis API Client (getAnalysis)', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetches canonical analysis result successfully on 200 OK', async () => {
    const mockAnalysisPayload = {
      analysis_id: 'analysis_01J_API_TEST',
      email_id: 'email_01J_API_TEST',
      case_id: 'case_01J_API_TEST',
      status: 'completed',
      ai_assessment: {
        status: 'completed',
        classification: 'Phishing',
        confidence: 0.91,
        supporting_signals: ['Suspicious sender'],
        evidence_references: ['header:From'],
        provider: 'anthropic',
        model: 'claude-3-5-sonnet',
        failure: null,
      },
      failure: null,
    };

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => mockAnalysisPayload,
    });
    vi.stubGlobal('fetch', mockFetch);

    const result = await getAnalysis('email_01J_API_TEST');

    expect(result).toEqual(mockAnalysisPayload);
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/emails/email_01J_API_TEST/analysis'),
      expect.objectContaining({
        method: 'GET',
        headers: { Accept: 'application/json' },
      })
    );
  });

  it('throws ApiError with ANALYSIS_NOT_FOUND on 404', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: async () => ({
        error: {
          code: 'ANALYSIS_NOT_FOUND',
          message: 'No analysis has been started for this email.',
        },
      }),
    });
    vi.stubGlobal('fetch', mockFetch);

    await expect(getAnalysis('non_existent_email')).rejects.toThrow(ApiError);
    await expect(getAnalysis('non_existent_email')).rejects.toMatchObject({
      code: 'ANALYSIS_NOT_FOUND',
      status: 404,
    });
  });

  it('throws ApiError with INTERNAL_ERROR on 500', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({
        error: {
          code: 'INTERNAL_ERROR',
          message: 'Failed to load analysis record from store.',
        },
      }),
    });
    vi.stubGlobal('fetch', mockFetch);

    await expect(getAnalysis('email_err')).rejects.toMatchObject({
      code: 'INTERNAL_ERROR',
      status: 500,
    });
  });

  it('throws ApiError with MALFORMED_RESPONSE when response is invalid non-JSON', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => {
        throw new Error('Unexpected token < in JSON at position 0');
      },
    });
    vi.stubGlobal('fetch', mockFetch);

    await expect(getAnalysis('email_corrupt')).rejects.toMatchObject({
      code: 'MALFORMED_RESPONSE',
    });
  });

  it('throws ApiError with MALFORMED_RESPONSE when response lacks ai_assessment', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        analysis_id: 'analysis_bad',
        // missing ai_assessment
      }),
    });
    vi.stubGlobal('fetch', mockFetch);

    await expect(getAnalysis('email_bad')).rejects.toMatchObject({
      code: 'MALFORMED_RESPONSE',
    });
  });

  it('throws ApiError with CONNECTION_REFUSED on network failure', async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error('Failed to fetch'));
    vi.stubGlobal('fetch', mockFetch);

    await expect(getAnalysis('email_offline')).rejects.toMatchObject({
      code: 'CONNECTION_REFUSED',
    });
  });
});

describe('getFriendlyErrorMessage for Analysis Codes', () => {
  it('maps ANALYSIS_NOT_FOUND accurately', () => {
    const err = new ApiError('Analysis not found', 'ANALYSIS_NOT_FOUND', 404);
    const friendly = getFriendlyErrorMessage(err);
    expect(friendly.code).toBe('ANALYSIS_NOT_FOUND');
    expect(friendly.title).toBe('Analysis Not Found');
    expect(friendly.message).toMatch(/No analysis has been initialized/i);
  });

  it('maps AI_NOT_CONFIGURED accurately', () => {
    const err = new ApiError('No AI analyzer', 'AI_NOT_CONFIGURED', 200);
    const friendly = getFriendlyErrorMessage(err);
    expect(friendly.code).toBe('AI_NOT_CONFIGURED');
    expect(friendly.title).toBe('AI Analyzer Unavailable');
    expect(friendly.message).toMatch(/Deterministic parsing remains active/i);
  });

  it('maps AI_ANALYSIS_FAILED accurately', () => {
    const err = new ApiError('Model error', 'AI_ANALYSIS_FAILED', 500);
    const friendly = getFriendlyErrorMessage(err);
    expect(friendly.code).toBe('AI_ANALYSIS_FAILED');
    expect(friendly.title).toBe('AI Assessment Failed');
  });
});
