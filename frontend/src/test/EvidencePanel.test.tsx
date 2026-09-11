import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { EvidencePanel } from '../components/investigation/EvidenceStage/EvidencePanel';
import { EvidenceDrawer } from '../components/investigation/EvidenceStage/EvidenceDrawer';
import { EmailEvidenceResponse, EvidenceItem } from '../types/api';

// ----------------------------
// Shared test fixture data
// ----------------------------

const observedAuthItem: EvidenceItem = {
  evidence_id: 'evidence_AUTH_001',
  type: 'authentication',
  source: 'Authentication-Results',
  value: 'spf=fail',
  snippet: 'Authentication-Results: mx.target.com; spf=fail smtp.mailfrom=attacker.net',
  provenance: 'OBSERVED',
  header_order: 7,
  related_signal_codes: ['SPF_FAIL', 'DKIM_FAIL'],
  hash: null,
};

const enrichedIpItem: EvidenceItem = {
  evidence_id: 'evidence_IP_001',
  type: 'ip_enrichment',
  source: 'ipinfo.io',
  value: '198.51.100.22',
  snippet: null,
  provenance: 'ENRICHED',
  header_order: null,
  related_signal_codes: ['SUSPICIOUS_IP'],
  hash: null,
};

const inferredItem: EvidenceItem = {
  evidence_id: 'evidence_INF_001',
  type: 'relay',
  source: 'Relay Chain',
  value: 'anomalous hop gap > 3 hours',
  snippet: null,
  provenance: 'INFERRED',
  header_order: null,
  related_signal_codes: [],
  hash: null,
};

const aiAssessedItem: EvidenceItem = {
  evidence_id: 'evidence_AI_001',
  type: 'ai_finding',
  source: 'Gemini',
  value: 'Phishing attempt detected with 94% model confidence',
  snippet: null,
  provenance: 'AI-ASSESSED',
  header_order: null,
  related_signal_codes: ['AI_PHISHING'],
  hash: null,
};

const attachmentItem: EvidenceItem = {
  evidence_id: 'evidence_ATT_001',
  type: 'attachment',
  source: 'MIME payload',
  value: 'invoice.pdf.exe',
  snippet: null,
  provenance: 'OBSERVED',
  header_order: null,
  related_signal_codes: [],
  hash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
};

const fullEvidenceResponse: EmailEvidenceResponse = {
  email_id: 'email_EVIDENCE_TEST',
  analysis_id: 'analysis_EVIDENCE_TEST',
  evidence: [observedAuthItem, enrichedIpItem, inferredItem, aiAssessedItem, attachmentItem],
};

// =====================================================
// EvidencePanel Tests
// =====================================================

describe('EvidencePanel Component', () => {
  describe('1. Loading state', () => {
    it('renders loading indicator when isLoading=true', () => {
      render(<EvidencePanel isLoading={true} />);
      expect(screen.getByTestId('evidence-panel')).toBeInTheDocument();
      expect(screen.getByTestId('evidence-loading')).toBeInTheDocument();
      expect(screen.getByText(/Retrieving cryptographically verifiable evidence/i)).toBeInTheDocument();
    });
  });

  describe('2. Empty state', () => {
    it('renders empty state when evidence array is empty', () => {
      render(
        <EvidencePanel
          evidenceData={{ email_id: 'email_TEST', evidence: [] }}
        />
      );
      expect(screen.getByTestId('evidence-empty')).toBeInTheDocument();
      expect(screen.getByText('No Evidence Available')).toBeInTheDocument();
    });
  });

  describe('3. Non-fatal error state', () => {
    it('renders error banner non-fatally when error is set, keeping disclaimers visible', () => {
      const mockError = { code: 'EVIDENCE_NOT_FOUND', message: 'No evidence records found' };
      const mockRetry = vi.fn();

      render(
        <EvidencePanel
          evidenceData={{ email_id: 'email_TEST', evidence: [] }}
          error={mockError}
          onRetry={mockRetry}
        />
      );

      // Error is shown
      expect(screen.getByTestId('evidence-error')).toBeInTheDocument();

      // But disclaimers remain visible - non-fatal
      expect(screen.getByTestId('evidence-disclaimers')).toBeInTheDocument();
    });
  });

  describe('4. Evidence records rendering', () => {
    it('renders all evidence items from response', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);

      const items = screen.getAllByTestId('evidence-item');
      expect(items).toHaveLength(5);

      // Count badge
      expect(screen.getByTestId('evidence-count-badge')).toHaveTextContent('5 records');
    });

    it('renders OBSERVED provenance badge for observed items', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }} />);
      const badges = screen.getAllByTestId('provenance-badge');
      const observedBadges = badges.filter(b => b.textContent?.includes('OBSERVED'));
      expect(observedBadges.length).toBeGreaterThanOrEqual(1);
    });

    it('renders ENRICHED provenance badge for enriched items', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [enrichedIpItem] }} />);
      const badges = screen.getAllByTestId('provenance-badge');
      const enrichedBadge = badges.find(b => b.textContent?.includes('ENRICHED'));
      expect(enrichedBadge).toBeInTheDocument();
    });

    it('renders INFERRED provenance badge for inferred items', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [inferredItem] }} />);
      const badges = screen.getAllByTestId('provenance-badge');
      const inferredBadge = badges.find(b => b.textContent?.includes('INFERRED'));
      expect(inferredBadge).toBeInTheDocument();
    });

    it('renders AI-ASSESSED provenance badge for AI-assessed items', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [aiAssessedItem] }} />);
      const badges = screen.getAllByTestId('provenance-badge');
      const aiBadge = badges.find(b => b.textContent?.includes('AI-ASSESSED'));
      expect(aiBadge).toBeInTheDocument();
    });

    it('renders snippet when present', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }} />);
      expect(screen.getByTestId('evidence-snippet')).toBeInTheDocument();
      expect(screen.getByText('VERIFIABLE ARTIFACT SNIPPET')).toBeInTheDocument();
      expect(screen.getByText(/spf=fail smtp.mailfrom=attacker.net/)).toBeInTheDocument();
    });

    it('renders value when no snippet', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [enrichedIpItem] }} />);
      expect(screen.getByTestId('evidence-value')).toBeInTheDocument();
      expect(screen.getByText('EXTRACTED VALUE')).toBeInTheDocument();
    });

    it('renders SHA-256 hash when present', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [attachmentItem] }} />);
      expect(screen.getByTestId('evidence-hash')).toBeInTheDocument();
      expect(screen.getByText('ARTIFACT SHA-256:')).toBeInTheDocument();
    });

    it('renders header order pill when header_order is present', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }} />);
      expect(screen.getByTestId('header-order-pill')).toHaveTextContent('Order #7');
    });

    it('does NOT render header order pill when header_order is null', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [enrichedIpItem] }} />);
      expect(screen.queryByTestId('header-order-pill')).not.toBeInTheDocument();
    });
  });

  describe('5. Related signal codes', () => {
    it('renders related signal code buttons for items with signals', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }} />);
      expect(screen.getByTestId('evidence-panel')).toBeInTheDocument();
      // Signal buttons should be rendered (SPF_FAIL, DKIM_FAIL)
      const signalContainer = screen.getByTestId('related-signals-container');
      expect(within(signalContainer).getByTestId('signal-code-SPF_FAIL')).toBeInTheDocument();
      expect(within(signalContainer).getByTestId('signal-code-DKIM_FAIL')).toBeInTheDocument();
    });

    it('calls onSelectSignalCode when signal button is clicked', () => {
      const mockSelectSignal = vi.fn();
      render(
        <EvidencePanel
          evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }}
          onSelectSignalCode={mockSelectSignal}
        />
      );
      fireEvent.click(screen.getByTestId('signal-code-SPF_FAIL'));
      expect(mockSelectSignal).toHaveBeenCalledWith('SPF_FAIL');
    });
  });

  describe('6. Filtering by signal code', () => {
    it('filters evidence by selectedSignalCode prop', () => {
      render(
        <EvidencePanel
          evidenceData={fullEvidenceResponse}
          selectedSignalCode="SPF_FAIL"
        />
      );

      // Active filter ribbon should appear
      expect(screen.getByTestId('active-filter-ribbon')).toBeInTheDocument();
      expect(screen.getByTestId('active-signal-pill')).toHaveTextContent('SPF_FAIL');

      // Only 1 item has SPF_FAIL signal
      const items = screen.getAllByTestId('evidence-item');
      expect(items).toHaveLength(1);
    });

    it('shows no-matches state when filter matches 0 items', () => {
      render(
        <EvidencePanel
          evidenceData={fullEvidenceResponse}
          selectedSignalCode="NONEXISTENT_SIGNAL"
        />
      );
      expect(screen.getByTestId('evidence-no-matches')).toBeInTheDocument();
    });

    it('clears filter when clear button is clicked', () => {
      const mockClear = vi.fn();
      render(
        <EvidencePanel
          evidenceData={fullEvidenceResponse}
          selectedSignalCode="SPF_FAIL"
          onClearFilter={mockClear}
        />
      );

      fireEvent.click(screen.getByTestId('clear-evidence-filter-btn'));
      expect(mockClear).toHaveBeenCalled();
    });
  });

  describe('7. Text search filtering', () => {
    it('filters evidence by search term matching value field', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);

      const searchInput = screen.getByTestId('evidence-search-input');
      fireEvent.change(searchInput, { target: { value: '198.51.100.22' } });

      const items = screen.getAllByTestId('evidence-item');
      expect(items).toHaveLength(1);
    });

    it('filters evidence by search term matching source field', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);

      const searchInput = screen.getByTestId('evidence-search-input');
      fireEvent.change(searchInput, { target: { value: 'ipinfo.io' } });

      const items = screen.getAllByTestId('evidence-item');
      expect(items).toHaveLength(1);
    });
  });

  describe('8. Type filter', () => {
    it('filters evidence by type selector', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);

      const typeFilter = screen.getByTestId('evidence-type-filter');
      fireEvent.change(typeFilter, { target: { value: 'authentication' } });

      const items = screen.getAllByTestId('evidence-item');
      expect(items).toHaveLength(1);
    });
  });

  describe('9. Epistemic disclaimer notices (required)', () => {
    it('renders AI Assessment disclaimer', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);
      expect(screen.getByTestId('evidence-disclaimers')).toHaveTextContent(
        'AI findings are evaluated assessments, not ground truth.'
      );
    });

    it('renders Geolocation disclaimer with exact required text', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);
      expect(screen.getByTestId('evidence-disclaimers')).toHaveTextContent(
        'Estimated IP location; does not prove physical actor presence.'
      );
    });

    it('renders Enrichment disclaimer with exact required text', () => {
      render(<EvidencePanel evidenceData={fullEvidenceResponse} />);
      expect(screen.getByTestId('evidence-disclaimers')).toHaveTextContent(
        'Enriched data is not proof of identity or maliciousness.'
      );
    });
  });

  describe('10. Safe rendering - no clickable raw URLs', () => {
    it('does NOT render raw URLs as <a> anchor tags (URL safety)', () => {
      const urlItem: EvidenceItem = {
        evidence_id: 'evidence_URL_001',
        type: 'indicator',
        source: 'Message-Body',
        value: 'https://attacker.org/phish-payload',
        snippet: null,
        provenance: 'OBSERVED',
        header_order: null,
        related_signal_codes: [],
        hash: null,
      };

      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [urlItem] }} />);

      // Must render as code/text, not an anchor link
      const anchors = document.querySelectorAll('a[href]');
      const externalLinks = Array.from(anchors).filter(a =>
        (a as HTMLAnchorElement).href.includes('attacker.org')
      );
      expect(externalLinks).toHaveLength(0);
    });
  });

  describe('11. Inspect evidence button', () => {
    it('calls onSelectEvidenceItem when inspect button is clicked', () => {
      const mockInspect = vi.fn();
      render(
        <EvidencePanel
          evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }}
          onSelectEvidenceItem={mockInspect}
        />
      );
      const btn = screen.getByTestId(`inspect-evidence-${observedAuthItem.evidence_id}`);
      fireEvent.click(btn);
      expect(mockInspect).toHaveBeenCalledWith(observedAuthItem);
    });

    it('does NOT render inspect button when no onSelectEvidenceItem provided', () => {
      render(<EvidencePanel evidenceData={{ email_id: 'e', evidence: [observedAuthItem] }} />);
      expect(screen.queryByTestId(`inspect-evidence-${observedAuthItem.evidence_id}`)).not.toBeInTheDocument();
    });
  });
});

// =====================================================
// EvidenceDrawer Tests
// =====================================================

describe('EvidenceDrawer Component', () => {
  it('renders nothing when isOpen=false', () => {
    render(<EvidenceDrawer isOpen={false} onClose={vi.fn()} />);
    expect(screen.queryByTestId('evidence-drawer')).not.toBeInTheDocument();
  });

  it('renders nothing when evidenceItem is null', () => {
    render(<EvidenceDrawer isOpen={true} onClose={vi.fn()} evidenceItem={null} />);
    expect(screen.queryByTestId('evidence-drawer')).not.toBeInTheDocument();
  });

  it('renders drawer with evidence ID when open with item', () => {
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={observedAuthItem}
      />
    );
    expect(screen.getByTestId('evidence-drawer')).toBeInTheDocument();
    expect(screen.getByTestId('drawer-evidence-id')).toHaveTextContent('evidence_AUTH_001');
  });

  it('renders provenance badge in drawer header', () => {
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={observedAuthItem}
      />
    );
    const badges = screen.getAllByTestId('provenance-badge');
    const observedBadge = badges.find(b => b.textContent?.includes('OBSERVED'));
    expect(observedBadge).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', () => {
    const mockClose = vi.fn();
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={mockClose}
        evidenceItem={observedAuthItem}
      />
    );
    fireEvent.click(screen.getByTestId('close-evidence-drawer-btn'));
    expect(mockClose).toHaveBeenCalled();
  });

  it('calls onClose when Escape key is pressed', () => {
    const mockClose = vi.fn();
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={mockClose}
        evidenceItem={observedAuthItem}
      />
    );
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(mockClose).toHaveBeenCalled();
  });

  it('renders snippet when evidenceItem has snippet', () => {
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={observedAuthItem}
      />
    );
    expect(screen.getByTestId('drawer-snippet')).toBeInTheDocument();
    expect(screen.getByText(/spf=fail smtp.mailfrom=attacker.net/)).toBeInTheDocument();
  });

  it('renders value block when no snippet', () => {
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={enrichedIpItem}
      />
    );
    expect(screen.getByTestId('drawer-value')).toBeInTheDocument();
  });

  it('renders corroborated signal buttons in drawer', () => {
    const mockSelectSignal = vi.fn();
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={observedAuthItem}
        onSelectSignalCode={mockSelectSignal}
      />
    );
    const spfBtn = screen.getByTestId('drawer-signal-SPF_FAIL');
    expect(spfBtn).toBeInTheDocument();
    fireEvent.click(spfBtn);
    expect(mockSelectSignal).toHaveBeenCalledWith('SPF_FAIL');
  });

  it('renders header order when present', () => {
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={observedAuthItem}
      />
    );
    expect(screen.getByText('#7')).toBeInTheDocument();
  });

  it('renders SHA-256 hash when present', () => {
    render(
      <EvidenceDrawer
        isOpen={true}
        onClose={vi.fn()}
        evidenceItem={attachmentItem}
      />
    );
    expect(screen.getByText('SHA-256 HASH:')).toBeInTheDocument();
  });
});
