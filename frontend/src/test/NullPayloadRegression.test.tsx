import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { EvidencePanel } from '../components/investigation/EvidenceStage/EvidencePanel';
import { GraphPanel, TimelinePanel } from '../components/investigation/ForensicPanels';
import { WorkspaceErrorBoundary } from '../components/states/WorkspaceErrorBoundary';
import { normalizeEvidence, normalizeGraph, normalizeTimeline } from '../services/normalize';

describe('backend-shaped nullable payload regression', () => {
  it('normalizes null evidence arrays and optional values before rendering', () => {
    const normalized = normalizeEvidence({ email_id: 'email-null', analysis_id: null, evidence: [{ evidence_id: 'e-1', type: 'url', source: null, value: null, snippet: null, provenance: 'OBSERVED', header_order: null, related_signal_codes: null, hash: null }] });
    render(<EvidencePanel evidenceData={normalized} />);
    expect(screen.getByTestId('evidence-item')).toBeInTheDocument();
    expect(screen.getByText('unknown source')).toBeInTheDocument();
  });

  it('normalizes null graph metadata, edge evidence and timeline evidence', () => {
    const graph = normalizeGraph({ case_id: 'case-null', email_ids: null, analysis_ids: null, nodes: [{ id: 'node-1', type: 'ip', label: null, value: null, provenance: null, evidence_ids: null, metadata: null }], edges: [{ id: 'edge-1', source_node_id: 'node-1', target_node_id: null, relationship: null, provenance: null, evidence_ids: null, confidence: null }] });
    const timeline = normalizeTimeline({ case_id: 'case-null', email_ids: null, events: [{ id: 'event-1', title: null, description: null, timestamp: null, evidence_ids: null, metadata: null, provenance: null, confidence: null }] });
    render(<><GraphPanel data={graph} /><TimelinePanel data={timeline} /></>);
    expect(screen.getByTestId('graph-node')).toBeInTheDocument();
    expect(screen.getByTestId('timeline-event')).toBeInTheDocument();
  });

  it('shows a visible diagnostic instead of a blank workspace on render failure', () => {
    const Broken = () => { throw new Error('safe render diagnostic'); };
    render(<WorkspaceErrorBoundary><Broken /></WorkspaceErrorBoundary>);
    expect(screen.getByTestId('workspace-render-error')).toHaveTextContent('safe render diagnostic');
  });
});
