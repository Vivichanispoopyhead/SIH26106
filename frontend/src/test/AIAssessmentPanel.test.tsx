import { describe, it, expect } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import {
  AIAssessmentPanel,
  formatConfidencePercentage,
} from '../components/investigation/AIAssessmentStage/AIAssessmentPanel';
import { EmailAnalysisResponse } from '../types/api';

describe('formatConfidencePercentage helper', () => {
  it('formats decimal numbers between 0 and 1 correctly', () => {
    expect(formatConfidencePercentage(0.93)).toBe('93%');
    expect(formatConfidencePercentage(0.875)).toBe('88%');
    expect(formatConfidencePercentage(1)).toBe('100%');
    expect(formatConfidencePercentage(0)).toBe('0%');
  });

  it('formats percentage scale numbers correctly', () => {
    expect(formatConfidencePercentage(95)).toBe('95%');
    expect(formatConfidencePercentage(42.3)).toBe('42%');
  });

  it('returns null for missing or invalid values', () => {
    expect(formatConfidencePercentage(null)).toBeNull();
    expect(formatConfidencePercentage(undefined)).toBeNull();
    expect(formatConfidencePercentage(NaN)).toBeNull();
  });
});

describe('AIAssessmentPanel Component', () => {
  const successfulAnalysis: EmailAnalysisResponse = {
    analysis_id: 'analysis_01J_SUCCESS',
    email_id: 'email_01J_SUCCESS',
    case_id: 'case_01J_SUCCESS',
    status: 'completed',
    ai_assessment: {
      status: 'completed',
      classification: 'Phishing / Credential Harvesting',
      confidence: 0.94,
      supporting_signals: [
        'Urgent password expiration threat in Subject',
        'Sender domain adversary.org does not match corporate branding',
        'Hidden credential harvesting link in call to action',
      ],
      evidence_references: [
        'header:Subject',
        'header:From',
        'url:https://adversary.org/login',
      ],
      provider: 'anthropic',
      model: 'claude-3-5-sonnet',
      failure: null,
    },
    failure: null,
  };

  const notAvailableAnalysis: EmailAnalysisResponse = {
    analysis_id: 'analysis_01J_NOT_AVAIL',
    email_id: 'email_01J_NOT_AVAIL',
    case_id: 'case_01J_NOT_AVAIL',
    status: 'completed',
    ai_assessment: {
      status: 'not_available',
      classification: null,
      confidence: null,
      supporting_signals: [],
      evidence_references: [],
      provider: null,
      model: null,
      failure: {
        code: 'AI_NOT_CONFIGURED',
        message: 'No AI analyzer is configured for this deployment.',
      },
    },
    failure: null,
  };

  const partialFailedAnalysis: EmailAnalysisResponse = {
    analysis_id: 'analysis_01J_PARTIAL',
    email_id: 'email_01J_PARTIAL',
    case_id: 'case_01J_PARTIAL',
    status: 'partial',
    ai_assessment: {
      status: 'failed',
      classification: null,
      confidence: null,
      supporting_signals: [],
      evidence_references: [],
      provider: null,
      model: null,
      failure: {
        code: 'AI_ANALYSIS_FAILED',
        message: 'Upstream LLM provider timed out after 30s.',
      },
    },
    failure: null,
  };

  it('1. renders successful AI assessment with all required fields', () => {
    render(<AIAssessmentPanel analysisData={successfulAnalysis} />);

    // Panel is present with success styling
    const panel = screen.getByTestId('ai-assessment-panel');
    expect(panel).toBeInTheDocument();

    // Pipeline status & AI assessment status
    expect(screen.getByTestId('pipeline-status')).toHaveTextContent('pipeline: completed');
    expect(screen.getByTestId('ai-assessment-status')).toHaveTextContent('ai: completed');

    // Classification only when present
    expect(screen.getByTestId('ai-classification-metric')).toBeInTheDocument();
    expect(screen.getByTestId('ai-classification-value')).toHaveTextContent(
      'Phishing / Credential Harvesting'
    );

    // Confidence as percentage only when present
    expect(screen.getByTestId('ai-confidence-metric')).toBeInTheDocument();
    expect(screen.getByTestId('ai-confidence-value')).toHaveTextContent('94%');

    // Provider / Model when present
    expect(screen.getByTestId('ai-provider-model-metric')).toBeInTheDocument();
    expect(screen.getByTestId('ai-provider-model-value')).toHaveTextContent(
      'anthropic / claude-3-5-sonnet'
    );

    // Analysis ID
    expect(screen.getByText('analysis_01J_SUCCESS')).toBeInTheDocument();

    // Provenance badge
    const badges = screen.getAllByTestId('provenance-badge');
    expect(badges[0]).toHaveTextContent('AI-ASSESSED');
  });

  it('2. renders supporting signals list accurately', () => {
    render(<AIAssessmentPanel analysisData={successfulAnalysis} />);

    expect(screen.getByTestId('ai-signals-count')).toHaveTextContent('3');
    const signalItems = screen.getAllByTestId('ai-signal-item');
    expect(signalItems).toHaveLength(3);
    expect(signalItems[0]).toHaveTextContent('Urgent password expiration threat in Subject');
    expect(signalItems[1]).toHaveTextContent(
      'Sender domain adversary.org does not match corporate branding'
    );
    expect(signalItems[2]).toHaveTextContent('Hidden credential harvesting link in call to action');
  });

  it('3. renders evidence references with OBSERVED provenance tags', () => {
    render(<AIAssessmentPanel analysisData={successfulAnalysis} />);

    expect(screen.getByTestId('ai-evidence-count')).toHaveTextContent('3');
    const evidenceItems = screen.getAllByTestId('ai-evidence-item');
    expect(evidenceItems).toHaveLength(3);
    expect(evidenceItems[0]).toHaveTextContent('header:Subject');
    expect(evidenceItems[1]).toHaveTextContent('header:From');
    expect(evidenceItems[2]).toHaveTextContent('url:https://adversary.org/login');

    // Each evidence reference chip should have an OBSERVED badge
    evidenceItems.forEach((item) => {
      expect(within(item).getByText('OBSERVED')).toBeInTheDocument();
    });
  });

  it('4. renders neutral not_available state for AI_NOT_CONFIGURED without crashing', () => {
    render(<AIAssessmentPanel analysisData={notAvailableAnalysis} />);

    // Neutral unavailable state
    expect(screen.getByTestId('ai-assessment-unavailable')).toBeInTheDocument();
    expect(screen.getByText('AI Assessment Unavailable')).toBeInTheDocument();

    // Explains deterministic parsing is still available
    expect(
      screen.getByText(/Deterministic RFC 5322 parsing, header analysis, MIME boundary inspection/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/All observed email evidence remains preserved in the case repository/i)
    ).toBeInTheDocument();

    // Shows notice from failure object
    expect(screen.getByText('No AI analyzer is configured for this deployment.')).toBeInTheDocument();

    // Statuses
    expect(screen.getByTestId('pipeline-status')).toHaveTextContent('pipeline: completed');
    expect(screen.getByTestId('ai-assessment-status')).toHaveTextContent('ai: not_available');

    // Classification and confidence MUST NOT be rendered
    expect(screen.queryByTestId('ai-classification-metric')).not.toBeInTheDocument();
    expect(screen.queryByTestId('ai-confidence-metric')).not.toBeInTheDocument();
    expect(screen.queryByTestId('ai-provider-model-metric')).not.toBeInTheDocument();
  });

  it('5. renders warning panel for partial/failed state and explains evidence is retained', () => {
    render(<AIAssessmentPanel analysisData={partialFailedAnalysis} />);

    expect(screen.getByTestId('ai-assessment-warning')).toBeInTheDocument();
    expect(screen.getByText('AI Assessment Incomplete')).toBeInTheDocument();
    expect(screen.getByText('[AI_ANALYSIS_FAILED]')).toBeInTheDocument();
    expect(screen.getByText(/Upstream LLM provider timed out after 30s/i)).toBeInTheDocument();

    // Explains deterministic parsed email evidence is retained
    expect(
      screen.getByText(/Deterministic email parsing succeeded. RFC 5322 headers, MIME parts/i)
    ).toBeInTheDocument();

    // Status pills reflect partial pipeline and failed AI assessment
    expect(screen.getByTestId('pipeline-status')).toHaveTextContent('pipeline: partial');
    expect(screen.getByTestId('ai-assessment-status')).toHaveTextContent('ai: failed');

    // Classification and confidence MUST NOT be rendered
    expect(screen.queryByTestId('ai-classification-metric')).not.toBeInTheDocument();
    expect(screen.queryByTestId('ai-confidence-metric')).not.toBeInTheDocument();
  });

  it('6. handles direct API error prop gracefully', () => {
    const apiError = {
      code: 'CONNECTION_REFUSED',
      message: 'Failed to contact analysis service at :8080',
    };

    render(<AIAssessmentPanel error={apiError} />);

    expect(screen.getByTestId('ai-assessment-warning')).toBeInTheDocument();
    expect(screen.getByTestId('error-code-badge')).toHaveTextContent('[CONNECTION_REFUSED]');
    expect(screen.getByText(/Failed to contact analysis service at :8080/i)).toBeInTheDocument();
    expect(screen.getByText(/Deterministic email parsing succeeded/i)).toBeInTheDocument();
  });

  it('7. does NOT invent a risk score, keeping confidence separate', () => {
    render(<AIAssessmentPanel analysisData={successfulAnalysis} />);

    // Confidence metric exists and explicitly clarifies it is model confidence, not threat severity
    const confidenceTile = screen.getByTestId('ai-confidence-metric');
    expect(confidenceTile).toBeInTheDocument();
    expect(within(confidenceTile).getByText('94%')).toBeInTheDocument();
    expect(
      within(confidenceTile).getByText(/Model Confidence \(Not Threat Severity\)/i)
    ).toBeInTheDocument();

    // Integrity footer confirms risk scoring remains distinct
    expect(screen.getByTestId('ai-card-footer')).toHaveTextContent(
      /Risk scoring remains distinct from analytical confidence/i
    );

    // Must NOT contain synthetic risk labels
    expect(screen.queryByText(/Risk Score:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Threat Severity: High/i)).not.toBeInTheDocument();
  });

  it('8. handles empty supporting signals and evidence references gracefully', () => {
    const analysisWithEmptyLists: EmailAnalysisResponse = {
      ...successfulAnalysis,
      ai_assessment: {
        ...successfulAnalysis.ai_assessment,
        supporting_signals: [],
        evidence_references: [],
      },
    };

    render(<AIAssessmentPanel analysisData={analysisWithEmptyLists} />);

    expect(screen.getByTestId('ai-signals-empty')).toHaveTextContent(
      'No supporting signals reported.'
    );
    expect(screen.getByTestId('ai-evidence-empty')).toHaveTextContent(
      'No evidence references linked.'
    );
  });

  it('9. renders loading stage when isLoading is true', () => {
    render(<AIAssessmentPanel isLoading={true} />);

    expect(screen.getByTestId('ai-assessment-loading')).toBeInTheDocument();
    expect(
      screen.getByText(/Evaluating semantic signals, intent indicators, and RFC 5322 references/i)
    ).toBeInTheDocument();
  });
});
