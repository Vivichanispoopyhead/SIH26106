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

export interface EmailAnalysisResponse {
  analysis_id: string;
  email_id: string;
  case_id: string;
  status: AnalysisStatus;
  ai_assessment: AIAssessment;
  failure: AnalysisFailure | null;
}

