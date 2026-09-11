import { EpistemicClass } from './provenance';

/**
 * API Contract Types matching docs/api-contract.md exactly.
 */

export type UploadStatus = 'uploaded';

export type AnalysisStatus =
  | 'started'
  | 'processing'
  | 'completed'
  | 'partial'
  | 'failed';

export type EmailStatus =
  | 'uploaded'
  | 'processing'
  | 'parsed'
  | 'partial'
  | 'failed';

export type ApiErrorCode =
  | 'INVALID_REQUEST'
  | 'MISSING_FILE'
  | 'UNSUPPORTED_FILE_TYPE'
  | 'FILE_TOO_LARGE'
  | 'EMPTY_FILE'
  | 'EMAIL_NOT_FOUND'
  | 'EMAIL_PARSE_FAILED'
  | 'ANALYSIS_NOT_FOUND'
  | 'ANALYSIS_FAILED'
  | 'EVIDENCE_NOT_FOUND'
  | 'CASE_NOT_FOUND'
  | 'GRAPH_NOT_AVAILABLE'
  | 'TIMELINE_NOT_AVAILABLE'
  | 'REPORT_NOT_AVAILABLE'
  | 'INTERNAL_ERROR'
  | 'CONNECTION_REFUSED'
  | 'MALFORMED_RESPONSE'
  | string;

export interface ApiErrorBody {
  error: {
    code: ApiErrorCode;
    message: string;
  };
}

export interface UploadEmailResponse {
  case_id: string;
  email_id: string;
  status: UploadStatus | string;
}

export interface StartAnalysisResponse {
  analysis_id?: string;
  email_id: string;
  case_id: string;
  status: AnalysisStatus | string;
}

export interface MessageMetadata {
  message_id?: string;
  from?: string[];
  to?: string[];
  cc?: string[];
  reply_to?: string[];
  subject?: string;
  date?: string;
  return_path?: string;
}

export interface MimeInfo {
  content_type: string;
  has_plain_text: boolean;
  has_html: boolean;
  attachment_count: number;
}

export interface HeaderEntry {
  name: string;
  value: string;
  order: number;
}

export interface IndicatorSet {
  ips: string[];
  domains: string[];
  urls: string[];
}

export interface AttachmentMetadata {
  filename: string;
  mime_type: string;
  size_bytes: number;
  sha256: string;
}

export interface ParsedEmailResponse {
  email_id: string;
  case_id: string;
  status: EmailStatus | string;
  filename?: string;
  message?: MessageMetadata;
  mime?: MimeInfo;
  headers?: HeaderEntry[];
  indicators?: IndicatorSet;
  attachments?: AttachmentMetadata[];
}

export type AIAssessmentStatus =
  | 'not_available'
  | 'completed'
  | 'failed'
  | 'partial';

export interface AnalysisFailure {
  code: string;
  message: string;
}

export interface AIAssessment {
  status: AIAssessmentStatus;
  classification: string | null;
  confidence: number | null;
  supporting_signals: string[];
  evidence_references: string[];
  provider: string | null;
  model: string | null;
  failure: AnalysisFailure | null;
}

export interface RiskSignal {
  code: string;
  description: string;
  points: number;
  category: string;
  provenance: EpistemicClass | string;
  evidence_references?: Array<{
    header_order?: number;
    header_name?: string;
    source?: string;
    value?: string;
  }>;
  evidence_ids?: string[];
}

export interface RiskAssessment {
  score: number;
  level: 'low' | 'medium' | 'high' | 'critical' | string;
  verdict: 'benign' | 'suspicious' | 'phishing' | 'malware' | 'fraud' | 'unknown' | string;
  confidence: number | null;
  contributing_signals: RiskSignal[];
  evidence_references?: Array<{
    header_order?: number;
    header_name?: string;
    source?: string;
    value?: string;
  }>;
}

export interface AuthenticationCheck {
  status: 'pass' | 'fail' | 'neutral' | 'none' | 'unknown' | string;
  evidence_references?: Array<{
    header_order?: number;
    header_name?: string;
    source?: string;
    value?: string;
  }>;
  explanation: string;
}

export interface AuthenticationResults {
  spf: AuthenticationCheck;
  dkim: AuthenticationCheck;
  dmarc: AuthenticationCheck;
}

export interface EmailAnalysisResponse {
  analysis_id: string;
  email_id: string;
  case_id: string;
  status: AnalysisStatus;
  ai_assessment: AIAssessment;
  risk?: RiskAssessment | null;
  authentication?: AuthenticationResults | null;
  received_chain?: ReceivedRelay[];
  ip_enrichment?: IPEnrichment[];
  failure: AnalysisFailure | null;
}

export type EvidenceType =
  | 'authentication'
  | 'header'
  | 'indicator'
  | 'attachment'
  | 'ai_finding'
  | 'ip_enrichment'
  | 'relay'
  | 'body'
  | string;

export interface EvidenceItem {
  evidence_id: string;
  type: EvidenceType;
  source: string;
  value: string;
  snippet?: string | null;
  provenance: EpistemicClass;
  header_order?: number | null;
  related_signal_codes: string[];
  hash?: string | null;
}

export interface EmailEvidenceResponse {
  email_id: string;
  analysis_id?: string | null;
  evidence: EvidenceItem[];
}

export type EnrichmentStatus = 'enriched' | 'not_applicable' | 'failed' | 'not_configured' | string;

export interface IPEnrichment {
  ip_address: string;
  status: EnrichmentStatus;
  country: string | null;
  region: string | null;
  city: string | null;
  latitude: number | null;
  longitude: number | null;
  asn: string | null;
  organization: string | null;
  provider: string | null;
  confidence: number | null;
  retrieved_at: string | null;
  provenance: EpistemicClass | string;
  failure: AnalysisFailure | null;
}

export interface ReceivedRelay {
  sequence: number;
  hostname: string | null;
  ip_address: string | null;
  timestamp: string | null;
  source_header_order: number;
  confidence: string;
  provenance: EpistemicClass | string;
}

export interface GraphNode {
  id: string;
  type: string;
  label: string;
  value: string;
  provenance: EpistemicClass | string;
  evidence_ids: string[];
  metadata: Record<string, string>;
}

export interface GraphEdge {
  id: string;
  source_node_id: string;
  target_node_id: string;
  relationship: string;
  provenance: EpistemicClass | string;
  evidence_ids: string[];
  confidence: number | null;
}

export interface CaseGraphResponse {
  case_id: string;
  email_ids: string[];
  analysis_ids: string[];
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface TimelineEvent {
  id: string;
  type: string;
  sequence: number;
  timestamp: string | null;
  title: string;
  description: string;
  hostname: string | null;
  ip_address: string | null;
  source_header_order: number | null;
  provenance: EpistemicClass | string;
  confidence: string;
  evidence_ids: string[];
  related_node_id: string | null;
  metadata: Record<string, string>;
}

export interface CaseTimelineResponse {
  case_id: string;
  email_ids: string[];
  events: TimelineEvent[];
}

export interface ReportGraphSummary {
  node_count: number;
  edge_count: number;
  node_types: Record<string, number>;
  node_ids: string[];
  edge_ids: string[];
}

export interface ReportEmail {
  email_id: string;
  filename: string;
  created_at: string;
  message: MessageMetadata;
  mime: MimeInfo;
  indicators: IndicatorSet;
  attachments: AttachmentMetadata[];
}

export interface ReportAnalysis {
  analysis_id: string;
  email_id: string;
  case_id: string;
  status: AnalysisStatus;
  risk: RiskAssessment;
  authentication: AuthenticationResults;
  received_chain: ReceivedRelay[];
  ip_enrichment: IPEnrichment[];
  ai_assessment: AIAssessment;
}

export interface ForensicReport {
  report_id: string;
  schema_version: string;
  case: { id: string; status: string; created_at: string; updated_at: string };
  generated_at: string;
  status: 'completed' | 'partial' | string;
  emails: ReportEmail[];
  analyses: ReportAnalysis[];
  evidence: EvidenceItem[];
  graph: ReportGraphSummary;
  timeline: TimelineEvent[];
  limitations: string[];
}
