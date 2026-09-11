import {
  AnalysisFailure,
  AttachmentMetadata,
  AuthenticationResults,
  CaseGraphResponse,
  CaseTimelineResponse,
  EmailAnalysisResponse,
  EmailEvidenceResponse,
  EvidenceItem,
  ForensicReport,
  GraphEdge,
  GraphNode,
  HeaderEntry,
  IndicatorSet,
  IPEnrichment,
  MessageMetadata,
  MimeInfo,
  ParsedEmailResponse,
  ReceivedRelay,
  RiskAssessment,
  TimelineEvent,
} from '../types/api';

const stringValue = (value: unknown, fallback = ''): string => typeof value === 'string' ? value : fallback;
const nullableString = (value: unknown): string | null => typeof value === 'string' ? value : null;
const stringArray = (value: unknown): string[] => Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
const recordValue = (value: unknown): Record<string, string> => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  return Object.fromEntries(Object.entries(value).filter(([, item]) => typeof item === 'string')) as Record<string, string>;
};
const numberRecord = (value: unknown): Record<string, number> => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  return Object.fromEntries(Object.entries(value).filter(([, item]) => typeof item === 'number' && Number.isFinite(item))) as Record<string, number>;
};
const failure = (value: unknown): AnalysisFailure | null => {
  if (!value || typeof value !== 'object') return null;
  const item = value as Record<string, unknown>;
  const code = stringValue(item.code);
  const message = stringValue(item.message);
  return code || message ? { code, message } : null;
};
const numberOrNull = (value: unknown): number | null => typeof value === 'number' && Number.isFinite(value) ? value : null;

export function normalizeMessage(value: unknown): MessageMetadata {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return {
    message_id: nullableString(item.message_id) ?? undefined,
    from: stringArray(item.from), to: stringArray(item.to), cc: stringArray(item.cc), reply_to: stringArray(item.reply_to),
    subject: nullableString(item.subject) ?? undefined, date: nullableString(item.date) ?? undefined, return_path: nullableString(item.return_path) ?? undefined,
  };
}

export function normalizeMime(value: unknown): MimeInfo {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return { content_type: stringValue(item.content_type, 'unknown/unknown'), has_plain_text: item.has_plain_text === true, has_html: item.has_html === true, attachment_count: typeof item.attachment_count === 'number' ? item.attachment_count : 0 };
}

export function normalizeIndicators(value: unknown): IndicatorSet {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return { ips: stringArray(item.ips), domains: stringArray(item.domains), urls: stringArray(item.urls) };
}

export function normalizeAttachments(value: unknown): AttachmentMetadata[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object').map((item) => ({ filename: stringValue(item.filename, 'unnamed attachment'), mime_type: stringValue(item.mime_type, 'application/octet-stream'), size_bytes: typeof item.size_bytes === 'number' ? item.size_bytes : 0, sha256: stringValue(item.sha256) }));
}

export function normalizeEmail(value: unknown): ParsedEmailResponse {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const headers: HeaderEntry[] = Array.isArray(item.headers) ? item.headers.filter((entry): entry is Record<string, unknown> => Boolean(entry) && typeof entry === 'object').map((entry, index) => ({ name: stringValue(entry.name, 'Unknown'), value: stringValue(entry.value), order: typeof entry.order === 'number' ? entry.order : index + 1 })) : [];
  return { email_id: stringValue(item.email_id), case_id: stringValue(item.case_id), status: stringValue(item.status, 'uploaded'), filename: nullableString(item.filename) ?? undefined, message: normalizeMessage(item.message), mime: normalizeMime(item.mime), headers, indicators: normalizeIndicators(item.indicators), attachments: normalizeAttachments(item.attachments) };
}

function normalizeAuthCheck(value: unknown) {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return { status: stringValue(item.status, 'unknown'), explanation: stringValue(item.explanation), evidence_references: Array.isArray(item.evidence_references) ? item.evidence_references.filter((ref) => ref && typeof ref === 'object') : [] };
}

function normalizeAuthentication(value: unknown): AuthenticationResults {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return { spf: normalizeAuthCheck(item.spf), dkim: normalizeAuthCheck(item.dkim), dmarc: normalizeAuthCheck(item.dmarc) };
}

function normalizeRelay(value: unknown): ReceivedRelay {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return { sequence: typeof item.sequence === 'number' ? item.sequence : 0, hostname: nullableString(item.hostname), ip_address: nullableString(item.ip_address), timestamp: nullableString(item.timestamp), source_header_order: typeof item.source_header_order === 'number' ? item.source_header_order : 0, confidence: stringValue(item.confidence, 'unknown'), provenance: stringValue(item.provenance, 'INFERRED') };
}

function normalizeEnrichment(value: unknown): IPEnrichment {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  return { ip_address: stringValue(item.ip_address), status: stringValue(item.status, 'not_configured'), country: nullableString(item.country), region: nullableString(item.region), city: nullableString(item.city), latitude: numberOrNull(item.latitude), longitude: numberOrNull(item.longitude), asn: nullableString(item.asn), organization: nullableString(item.organization), provider: nullableString(item.provider), confidence: numberOrNull(item.confidence), retrieved_at: nullableString(item.retrieved_at), provenance: stringValue(item.provenance, 'INFERRED'), failure: failure(item.failure) };
}

function normalizeRisk(value: unknown): RiskAssessment {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const signals = Array.isArray(item.contributing_signals) ? item.contributing_signals.filter((signal): signal is Record<string, unknown> => Boolean(signal) && typeof signal === 'object').map((signal) => ({ code: stringValue(signal.code), description: stringValue(signal.description), points: typeof signal.points === 'number' ? signal.points : 0, category: stringValue(signal.category), provenance: stringValue(signal.provenance, 'INFERRED'), evidence_ids: stringArray(signal.evidence_ids), evidence_references: Array.isArray(signal.evidence_references) ? signal.evidence_references.filter((ref) => ref && typeof ref === 'object') : [] })) : [];
  return { score: typeof item.score === 'number' ? item.score : 0, level: stringValue(item.level, 'low'), verdict: stringValue(item.verdict, 'unknown'), confidence: numberOrNull(item.confidence), contributing_signals: signals, evidence_references: Array.isArray(item.evidence_references) ? item.evidence_references.filter((ref) => ref && typeof ref === 'object') : [] };
}

export function normalizeAnalysis(value: unknown): EmailAnalysisResponse {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const ai = item.ai_assessment && typeof item.ai_assessment === 'object' ? item.ai_assessment as Record<string, unknown> : {};
  return { analysis_id: stringValue(item.analysis_id), email_id: stringValue(item.email_id), case_id: stringValue(item.case_id), status: stringValue(item.status, 'failed') as EmailAnalysisResponse['status'], ai_assessment: { status: stringValue(ai.status, 'not_available') as EmailAnalysisResponse['ai_assessment']['status'], classification: nullableString(ai.classification), confidence: numberOrNull(ai.confidence), supporting_signals: stringArray(ai.supporting_signals), evidence_references: stringArray(ai.evidence_references), provider: nullableString(ai.provider), model: nullableString(ai.model), failure: failure(ai.failure) }, risk: normalizeRisk(item.risk), authentication: normalizeAuthentication(item.authentication), received_chain: Array.isArray(item.received_chain) ? item.received_chain.map(normalizeRelay) : [], ip_enrichment: Array.isArray(item.ip_enrichment) ? item.ip_enrichment.map(normalizeEnrichment) : [], failure: failure(item.failure) };
}

export function normalizeEvidence(value: unknown): EmailEvidenceResponse {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const evidence: EvidenceItem[] = Array.isArray(item.evidence) ? item.evidence.filter((entry): entry is Record<string, unknown> => Boolean(entry) && typeof entry === 'object').map((entry) => ({ evidence_id: stringValue(entry.evidence_id), type: stringValue(entry.type, 'unknown'), source: stringValue(entry.source, 'unknown source'), value: stringValue(entry.value), snippet: nullableString(entry.snippet), provenance: stringValue(entry.provenance, 'OBSERVED') as EvidenceItem['provenance'], header_order: typeof entry.header_order === 'number' ? entry.header_order : null, related_signal_codes: stringArray(entry.related_signal_codes), hash: nullableString(entry.hash) })) : [];
  return { email_id: stringValue(item.email_id), analysis_id: nullableString(item.analysis_id), evidence };
}

export function normalizeGraph(value: unknown): CaseGraphResponse {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const nodes: GraphNode[] = Array.isArray(item.nodes) ? item.nodes.filter((node): node is Record<string, unknown> => Boolean(node) && typeof node === 'object').map((node) => ({ id: stringValue(node.id), type: stringValue(node.type, 'unknown'), label: stringValue(node.label, 'Unnamed entity'), value: stringValue(node.value), provenance: stringValue(node.provenance, 'OBSERVED'), evidence_ids: stringArray(node.evidence_ids), metadata: recordValue(node.metadata) })) : [];
  const edges: GraphEdge[] = Array.isArray(item.edges) ? item.edges.filter((edge): edge is Record<string, unknown> => Boolean(edge) && typeof edge === 'object').map((edge) => ({ id: stringValue(edge.id), source_node_id: stringValue(edge.source_node_id), target_node_id: stringValue(edge.target_node_id), relationship: stringValue(edge.relationship, 'related_to'), provenance: stringValue(edge.provenance, 'INFERRED'), evidence_ids: stringArray(edge.evidence_ids), confidence: numberOrNull(edge.confidence) })) : [];
  return { case_id: stringValue(item.case_id), email_ids: stringArray(item.email_ids), analysis_ids: stringArray(item.analysis_ids), nodes, edges };
}

export function normalizeTimeline(value: unknown): CaseTimelineResponse {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const events: TimelineEvent[] = Array.isArray(item.events) ? item.events.filter((event): event is Record<string, unknown> => Boolean(event) && typeof event === 'object').map((event) => ({ id: stringValue(event.id), type: stringValue(event.type, 'indicator_observed'), sequence: typeof event.sequence === 'number' ? event.sequence : 0, timestamp: nullableString(event.timestamp), title: stringValue(event.title, 'Forensic event'), description: stringValue(event.description), hostname: nullableString(event.hostname), ip_address: nullableString(event.ip_address), source_header_order: typeof event.source_header_order === 'number' ? event.source_header_order : null, provenance: stringValue(event.provenance, 'OBSERVED'), confidence: stringValue(event.confidence, 'unknown'), evidence_ids: stringArray(event.evidence_ids), related_node_id: nullableString(event.related_node_id), metadata: recordValue(event.metadata) })) : [];
  return { case_id: stringValue(item.case_id), email_ids: stringArray(item.email_ids), events };
}

export function normalizeReport(value: unknown): ForensicReport {
  const item = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const reportCase = item.case && typeof item.case === 'object' ? item.case as Record<string, unknown> : {};
  return { report_id: stringValue(item.report_id), schema_version: stringValue(item.schema_version, '1.0'), case: { id: stringValue(reportCase.id), status: stringValue(reportCase.status), created_at: stringValue(reportCase.created_at), updated_at: stringValue(reportCase.updated_at) }, generated_at: stringValue(item.generated_at), status: stringValue(item.status, 'partial'), emails: Array.isArray(item.emails) ? item.emails.filter(Boolean).map((email) => { const entry = email as Record<string, unknown>; return { email_id: stringValue(entry.email_id), filename: stringValue(entry.filename), created_at: stringValue(entry.created_at), message: normalizeMessage(entry.message), mime: normalizeMime(entry.mime), indicators: normalizeIndicators(entry.indicators), attachments: normalizeAttachments(entry.attachments) }; }) : [], analyses: Array.isArray(item.analyses) ? item.analyses.filter(Boolean).map((analysis) => normalizeAnalysis(analysis) as ForensicReport['analyses'][number]) : [], evidence: normalizeEvidence({ email_id: '', evidence: item.evidence }).evidence, graph: { node_count: typeof (item.graph as Record<string, unknown> | undefined)?.node_count === 'number' ? (item.graph as Record<string, unknown>).node_count as number : 0, edge_count: typeof (item.graph as Record<string, unknown> | undefined)?.edge_count === 'number' ? (item.graph as Record<string, unknown>).edge_count as number : 0, node_types: numberRecord((item.graph as Record<string, unknown> | undefined)?.node_types), node_ids: stringArray((item.graph as Record<string, unknown> | undefined)?.node_ids), edge_ids: stringArray((item.graph as Record<string, unknown> | undefined)?.edge_ids) }, timeline: normalizeTimeline({ events: item.timeline }).events, limitations: stringArray(item.limitations) };
}
