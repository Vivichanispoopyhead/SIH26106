import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { EnrichmentPanel, GraphPanel, ReportPanel, TimelinePanel } from '../components/investigation/ForensicPanels';

const timeline = {
  case_id: 'case-1', email_ids: ['email-1'], events: [
    { id: 'timeline:email-1:relay:1', type: 'relay_observed', sequence: 1, timestamp: null, title: 'Observed relay', description: 'Relay was present in the message path.', hostname: 'mx.example.test', ip_address: '192.0.2.1', source_header_order: 4, provenance: 'OBSERVED', confidence: 'medium', evidence_ids: ['evidence-1'], related_node_id: 'relay:email-1:4', metadata: {} },
  ],
};

describe('forensic investigation panels', () => {
  it('renders timeline provenance and explicitly handles missing timestamps', () => {
    render(<TimelinePanel data={timeline} />);
    expect(screen.getByTestId('timeline-event')).toHaveTextContent('OBSERVED');
    expect(screen.getByText('Timestamp unavailable')).toBeInTheDocument();
    expect(screen.getByText('192.0.2.1')).toBeInTheDocument();
  });

  it('renders graph nodes without inventing entities', () => {
    render(<GraphPanel data={{ case_id: 'case-1', email_ids: ['email-1'], analysis_ids: ['analysis-1'], nodes: [{ id: 'ip:192.0.2.1', type: 'ip', label: '192.0.2.1', value: '192.0.2.1', provenance: 'OBSERVED', evidence_ids: ['evidence-1'], metadata: {} }], edges: [] }} />);
    expect(screen.getByTestId('graph-node')).toBeInTheDocument();
    expect(screen.getByText('1 evidence reference')).toBeInTheDocument();
  });

  it('distinguishes unavailable enrichment from enriched data', () => {
    render(<EnrichmentPanel values={[{ ip_address: '203.0.113.5', status: 'not_configured', country: null, region: null, city: null, latitude: null, longitude: null, asn: null, organization: null, provider: null, confidence: null, retrieved_at: null, provenance: 'INFERRED', failure: null }]} />);
    expect(screen.getByTestId('enrichment-panel')).toHaveTextContent('not_configured');
    expect(screen.getByText('Estimated IP location; does not prove physical actor presence.')).toBeInTheDocument();
  });

  it('renders report limitations and safe report controls', () => {
    render(<ReportPanel data={{ report_id: 'report:case-1:analysis-1', schema_version: '1.0', case: { id: 'case-1', status: 'created', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }, generated_at: '2026-01-01T00:00:00Z', status: 'partial', emails: [], analyses: [], evidence: [], graph: { node_count: 0, edge_count: 0, node_types: {}, node_ids: [], edge_ids: [] }, timeline: [], limitations: ['AI output is an evaluated assessment, not ground truth.'] }} onGenerate={() => undefined} onDownloadPDF={vi.fn()} />);
    expect(screen.getByTestId('report-panel')).toHaveTextContent('AI output is an evaluated assessment, not ground truth.');
    expect(screen.getByRole('button', { name: /Download JSON/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Download PDF/i })).toBeInTheDocument();
  });
});
