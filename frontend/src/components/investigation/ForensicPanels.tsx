import React, { useMemo } from 'react';
import { Download, RefreshCw } from 'lucide-react';
import {
  CaseGraphResponse,
  CaseTimelineResponse,
  EmailAnalysisResponse,
  ForensicReport,
  IPEnrichment,
  TimelineEvent,
} from '../../types/api';
import { EpistemicClass } from '../../types/provenance';
import { ProvenanceBadge } from '../common/ProvenanceBadge';
import { MonoValue } from '../common/MonoValue';
import { ErrorBanner } from '../states/ErrorBanner';
import './ForensicPanels.css';
import { normalizeAnalysis, normalizeGraph, normalizeReport, normalizeTimeline } from '../../services/normalize';

interface AsyncState {
  loading?: boolean;
  error?: unknown | null;
}

export interface TimelinePanelProps extends AsyncState {
  data?: CaseTimelineResponse | null;
}

const timestampValue = (event: TimelineEvent): number => {
  if (!event.timestamp) return Number.POSITIVE_INFINITY;
  const parsed = Date.parse(event.timestamp);
  return Number.isNaN(parsed) ? Number.POSITIVE_INFINITY : parsed;
};

export const TimelinePanel: React.FC<TimelinePanelProps> = ({ data, loading, error }) => {
  const normalizedTimeline = useMemo(() => normalizeTimeline(data), [data]);
  const events = useMemo(() => [...normalizedTimeline.events].sort((a, b) => {
    const timestampOrder = timestampValue(a) - timestampValue(b);
    if (timestampOrder !== 0) return timestampOrder;
    if (a.sequence !== b.sequence) return a.sequence - b.sequence;
    return a.id.localeCompare(b.id);
  }), [normalizedTimeline]);

  return (
    <section className="forensic-panel" data-testid="timeline-panel">
      <div className="forensic-panel-header">
        <div><span className="section-eyebrow">Infrastructure chronology</span><h3>Relay timeline</h3></div>
        <span className="panel-count">{events.length} events</span>
      </div>
      {loading && <p className="panel-state">Loading evidence-backed timeline…</p>}
      {!loading && error !== null && error !== undefined && <ErrorBanner error={error} />}
      {!loading && !error && events.length === 0 && <p className="panel-state">No timeline events are available for this analysis.</p>}
      {!loading && !error && events.length > 0 && (
        <ol className="timeline-list">
          {events.map((event) => (
            <li key={event.id} className="timeline-event" data-testid="timeline-event">
              <span className="timeline-marker" aria-hidden="true" />
              <div className="timeline-event-body">
                <div className="timeline-event-topline">
                  <strong>{event.title}</strong>
                  <ProvenanceBadge classification={event.provenance as EpistemicClass} showIcon={false} />
                </div>
                <p>{event.description}</p>
                <div className="timeline-event-meta">
                  <span>Sequence <code>{event.sequence}</code></span>
                  {event.timestamp ? <code>{event.timestamp}</code> : <span className="muted">Timestamp unavailable</span>}
                  {event.hostname && <code>{event.hostname}</code>}
                  {event.ip_address && <code>{event.ip_address}</code>}
                  {event.source_header_order !== null && <span>Header <code>#{event.source_header_order}</code></span>}
                  <span>Confidence <strong>{event.confidence}</strong></span>
                </div>
                {event.evidence_ids.length > 0 && <div className="evidence-ref-row">Evidence: {event.evidence_ids.map((id) => <code key={id}>{id}</code>)}</div>}
              </div>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
};

export interface GraphPanelProps extends AsyncState {
  data?: CaseGraphResponse | null;
}

export const GraphPanel: React.FC<GraphPanelProps> = ({ data, loading, error }) => {
  const normalizedGraph = useMemo(() => normalizeGraph(data), [data]);
  const nodes = normalizedGraph.nodes;
  const edges = normalizedGraph.edges;
  const positions = useMemo(() => {
    const width = 760;
    const height = Math.max(220, Math.ceil(nodes.length / 3) * 110);
    return new Map(nodes.map((node, index) => [node.id, {
      x: 130 + (index % 3) * 250,
      y: 55 + Math.floor(index / 3) * 100,
      width,
      height,
    }]));
  }, [nodes]);

  return (
    <section className="forensic-panel" data-testid="graph-panel">
      <div className="forensic-panel-header">
        <div><span className="section-eyebrow">Evidence-backed relationships</span><h3>Entity graph</h3></div>
        <span className="panel-count">{nodes.length} nodes · {edges.length} edges</span>
      </div>
      {loading && <p className="panel-state">Loading graph entities…</p>}
      {!loading && error !== null && error !== undefined && <ErrorBanner error={error} />}
      {!loading && !error && nodes.length === 0 && <p className="panel-state">No graph is available until an analysis completes.</p>}
      {!loading && !error && nodes.length > 0 && (
        <>
          <div className="graph-canvas" data-testid="graph-canvas">
            <svg viewBox={`0 0 760 ${Math.max(220, Math.ceil(nodes.length / 3) * 110)}`} role="img" aria-label="Evidence-backed entity graph">
              {edges.map((edge) => {
                const from = positions.get(edge.source_node_id);
                const to = positions.get(edge.target_node_id);
                if (!from || !to) return null;
                return <line key={edge.id} x1={from.x} y1={from.y} x2={to.x} y2={to.y} className="graph-edge" />;
              })}
              {nodes.map((node) => {
                const position = positions.get(node.id);
                if (!position) return null;
                return (
                  <g key={node.id} transform={`translate(${position.x}, ${position.y})`} data-testid="graph-node">
                    <circle r="22" className="graph-node-circle" />
                    <text y="42" textAnchor="middle" className="graph-node-label">{node.label.slice(0, 28)}</text>
                  </g>
                );
              })}
            </svg>
          </div>
          <div className="graph-node-list">
            {nodes.map((node) => (
              <article className="graph-node-card" key={node.id}>
                <div className="graph-node-card-top"><code>{node.type}</code><ProvenanceBadge classification={node.provenance as EpistemicClass} showIcon={false} /></div>
                <strong>{node.label}</strong>
                <MonoValue value={node.value} label={node.type} />
                <span className="muted">{node.evidence_ids.length} evidence reference{node.evidence_ids.length === 1 ? '' : 's'}</span>
              </article>
            ))}
          </div>
        </>
      )}
    </section>
  );
};

export interface EnrichmentPanelProps extends AsyncState {
  values?: IPEnrichment[] | null;
}

export const EnrichmentPanel: React.FC<EnrichmentPanelProps> = ({ values, loading, error }) => {
  const normalizedValues = normalizeAnalysis({ ip_enrichment: values }).ip_enrichment ?? [];
  return <section className="forensic-panel" data-testid="enrichment-panel">
    <div className="forensic-panel-header">
      <div><span className="section-eyebrow">Passive infrastructure context</span><h3>IP enrichment</h3></div>
      <span className="panel-count">{normalizedValues.length} addresses</span>
    </div>
    <p className="panel-disclaimer">Estimated IP location; does not prove physical actor presence.</p>
    {loading && <p className="panel-state">Loading passive enrichment…</p>}
    {!loading && error !== null && error !== undefined && <ErrorBanner error={error} />}
    {!loading && !error && normalizedValues.length === 0 && <p className="panel-state">No IP enrichment records are available.</p>}
    {!loading && !error && normalizedValues.length > 0 && <div className="enrichment-table-wrap"><table className="forensic-table"><thead><tr><th>IP</th><th>Status</th><th>Location</th><th>ASN / organization</th><th>Provider</th></tr></thead><tbody>{normalizedValues.map((item) => <tr key={item.ip_address}><td><code>{item.ip_address}</code></td><td><span className={`enrichment-status status-${item.status}`}>{item.status}</span></td><td>{[item.city, item.region, item.country].filter(Boolean).join(', ') || '—'}</td><td>{[item.asn, item.organization].filter(Boolean).join(' · ') || '—'}</td><td>{item.provider ?? '—'}{item.confidence !== null && <span className="muted"> ({Math.round(item.confidence * 100)}%)</span>}</td></tr>)}</tbody></table></div>}
  </section>
  ;
};

export interface ReportPanelProps extends AsyncState {
  data?: ForensicReport | null;
  onGenerate: () => void;
}

export const ReportPanel: React.FC<ReportPanelProps> = ({ data, loading, error, onGenerate }) => {
  const normalizedReport = useMemo(() => data ? normalizeReport(data) : null, [data]);
  const downloadReport = () => {
    if (!normalizedReport) return;
    const blob = new Blob([JSON.stringify(normalizedReport, null, 2)], { type: 'application/json' });
    const objectUrl = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = objectUrl;
    anchor.download = `${normalizedReport.report_id.replace(/[^a-zA-Z0-9._-]/g, '_')}.json`;
    anchor.click();
    URL.revokeObjectURL(objectUrl);
  };
  return (
    <section className="forensic-panel" data-testid="report-panel">
      <div className="forensic-panel-header"><div><span className="section-eyebrow">Case conclusion</span><h3>Forensic report</h3></div><div className="report-actions"><button type="button" className="forensic-button" onClick={onGenerate} disabled={loading}><RefreshCw size={14} /> {normalizedReport ? 'Refresh report' : 'Generate report'}</button>{normalizedReport && <button type="button" className="forensic-button secondary" onClick={downloadReport}><Download size={14} /> Download JSON</button>}</div></div>
      {loading && <p className="panel-state">Generating the safe structured report…</p>}
      {!loading && error !== null && error !== undefined && <ErrorBanner error={error} />}
      {!loading && !error && !normalizedReport && <p className="panel-state">Generate a report after analysis completes.</p>}
      {!loading && !error && normalizedReport && <div className="report-summary"><div className="report-status-row"><span className={`report-status status-${normalizedReport.status}`}>{normalizedReport.status}</span><code>schema {normalizedReport.schema_version}</code><code>{normalizedReport.generated_at}</code></div><div className="report-metrics"><div><span>Risk</span><strong>{normalizedReport.analyses[0]?.risk.score ?? '—'}</strong></div><div><span>Verdict</span><strong>{normalizedReport.analyses[0]?.risk.verdict ?? 'unknown'}</strong></div><div><span>Evidence</span><strong>{normalizedReport.evidence.length}</strong></div><div><span>Timeline</span><strong>{normalizedReport.timeline.length}</strong></div><div><span>Graph nodes</span><strong>{normalizedReport.graph.node_count}</strong></div></div>{normalizedReport.limitations.length > 0 && <div className="report-limitations"><strong>Limitations</strong><ul>{normalizedReport.limitations.map((limitation) => <li key={limitation}>{limitation}</li>)}</ul></div>}<p className="panel-disclaimer">AI output is an evaluated assessment, not ground truth. URLs were not automatically visited and attachments were not executed.</p></div>}
    </section>
  );
};

export const AnalysisContextPanels: React.FC<{ analysis?: EmailAnalysisResponse | null }> = ({ analysis }) => (
  <EnrichmentPanel values={analysis?.ip_enrichment} />
);
