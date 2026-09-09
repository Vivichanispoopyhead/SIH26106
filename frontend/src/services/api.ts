import { API_BASE_URL } from '../config/env';
import {
  UploadEmailResponse,
  StartAnalysisResponse,
  ParsedEmailResponse,
  ApiErrorBody,
  ApiErrorCode,
} from '../types/api';

export interface HealthResponse {
  status: string;
}

export class ApiError extends Error {
  readonly code: ApiErrorCode;
  readonly status?: number;

  constructor(message: string, code: ApiErrorCode = 'INTERNAL_ERROR', status?: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

/**
 * Maps documented error codes to human-readable forensic guidance.
 * Does not depend on the exact backend message string.
 */
export function getFriendlyErrorMessage(err: unknown): { title: string; message: string; code: string } {
  if (err instanceof ApiError) {
    const code = err.code;
    switch (code) {
      case 'UNSUPPORTED_FILE_TYPE':
        return {
          title: 'Unsupported File Format',
          message: 'The selected file is not an RFC 5322 .eml file. Only .eml files are supported for ingestion.',
          code,
        };
      case 'FILE_TOO_LARGE':
        return {
          title: 'File Exceeds Limit',
          message: 'The uploaded file exceeds the 50 MB maximum size threshold for forensic analysis.',
          code,
        };
      case 'EMPTY_FILE':
        return {
          title: 'Empty File',
          message: 'The uploaded email file contains 0 bytes and cannot be ingested.',
          code,
        };
      case 'MISSING_FILE':
        return {
          title: 'Missing File',
          message: 'No file was provided in the upload request payload.',
          code,
        };
      case 'EMAIL_PARSE_FAILED':
        return {
          title: 'Email Parse Failed',
          message: 'The forensic engine could not parse the RFC 5322 structure. The raw artifact remains preserved on the server.',
          code,
        };
      case 'ANALYSIS_FAILED':
        return {
          title: 'Analysis Execution Failed',
          message: 'The analysis pipeline failed to produce a structured inspection result.',
          code,
        };
      case 'EMAIL_NOT_FOUND':
        return {
          title: 'Artifact Not Found',
          message: 'The requested email identifier does not exist in the active case repository.',
          code,
        };
      case 'CONNECTION_REFUSED':
        return {
          title: 'Backend Unavailable',
          message: 'Could not establish connection to the forensic backend API. Ensure the backend server is running.',
          code,
        };
      case 'MALFORMED_RESPONSE':
        return {
          title: 'Malformed API Response',
          message: 'Received an invalid or unparseable response from the backend service.',
          code,
        };
      case 'INVALID_REQUEST':
        return {
          title: 'Invalid Request',
          message: err.message || 'The request payload failed backend validation.',
          code,
        };
      default:
        return {
          title: 'System Error',
          message: err.message || 'An unexpected error occurred during the forensic workflow.',
          code,
        };
    }
  }

  if (err instanceof Error) {
    return {
      title: 'Unexpected Failure',
      message: err.message,
      code: 'UNKNOWN_ERROR',
    };
  }

  return {
    title: 'Unknown Error',
    message: 'An unknown error occurred.',
    code: 'UNKNOWN_ERROR',
  };
}

/**
 * Upload .eml file to POST /api/emails
 */
export async function uploadEmail(
  file: File,
  baseUrl: string = API_BASE_URL
): Promise<UploadEmailResponse> {
  // Pre-flight client validations according to contract
  if (!file.name.toLowerCase().endsWith('.eml')) {
    throw new ApiError(
      'Only .eml files are accepted for ingestion.',
      'UNSUPPORTED_FILE_TYPE',
      400
    );
  }

  if (file.size === 0) {
    throw new ApiError('The provided file is empty (0 bytes).', 'EMPTY_FILE', 400);
  }

  const MAX_SIZE = 50 * 1024 * 1024; // 50 MB
  if (file.size > MAX_SIZE) {
    throw new ApiError(
      'File size exceeds the 50 MB maximum ingestion limit.',
      'FILE_TOO_LARGE',
      413
    );
  }

  const normalizedBase = baseUrl.replace(/\/+$/, '');
  const endpoint = `${normalizedBase}/api/emails`;

  const formData = new FormData();
  formData.append('file', file);

  let response: Response;
  try {
    response = await fetch(endpoint, {
      method: 'POST',
      body: formData,
      headers: {
        Accept: 'application/json',
      },
    });
  } catch (err) {
    const reason = err instanceof Error ? err.message : 'Connection failed';
    throw new ApiError(`Unable to connect to backend at ${endpoint}: ${reason}`, 'CONNECTION_REFUSED');
  }

  if (!response.ok) {
    let errorCode: string | undefined;
    let errorMessage: string | undefined;

    try {
      const body = (await response.json()) as ApiErrorBody;
      errorCode = body?.error?.code;
      errorMessage = body?.error?.message;
    } catch {
      // response is not JSON
    }

    throw new ApiError(
      errorMessage || `Upload failed with status ${response.status}`,
      errorCode || (response.status === 413 ? 'FILE_TOO_LARGE' : 'INVALID_REQUEST'),
      response.status
    );
  }

  try {
    return (await response.json()) as UploadEmailResponse;
  } catch {
    throw new ApiError('Backend returned malformed non-JSON data on upload.', 'MALFORMED_RESPONSE', response.status);
  }
}

/**
 * Trigger analysis: POST /api/emails/{email_id}/analysis
 */
export async function startAnalysis(
  emailId: string,
  baseUrl: string = API_BASE_URL
): Promise<StartAnalysisResponse> {
  const normalizedBase = baseUrl.replace(/\/+$/, '');
  const endpoint = `${normalizedBase}/api/emails/${encodeURIComponent(emailId)}/analysis`;

  let response: Response;
  try {
    response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
      },
    });
  } catch (err) {
    const reason = err instanceof Error ? err.message : 'Connection failed';
    throw new ApiError(`Unable to connect to backend at ${endpoint}: ${reason}`, 'CONNECTION_REFUSED');
  }

  if (!response.ok) {
    let errorCode: string | undefined;
    let errorMessage: string | undefined;

    try {
      const body = (await response.json()) as ApiErrorBody;
      errorCode = body?.error?.code;
      errorMessage = body?.error?.message;
    } catch {
      // response is not JSON
    }

    throw new ApiError(
      errorMessage || `Analysis start failed with status ${response.status}`,
      errorCode || 'ANALYSIS_FAILED',
      response.status
    );
  }

  try {
    return (await response.json()) as StartAnalysisResponse;
  } catch {
    throw new ApiError('Backend returned malformed non-JSON data on analysis trigger.', 'MALFORMED_RESPONSE', response.status);
  }
}

/**
 * Fetch parsed email state: GET /api/emails/{email_id}
 */
export async function getEmail(
  emailId: string,
  baseUrl: string = API_BASE_URL
): Promise<ParsedEmailResponse> {
  const normalizedBase = baseUrl.replace(/\/+$/, '');
  const endpoint = `${normalizedBase}/api/emails/${encodeURIComponent(emailId)}`;

  let response: Response;
  try {
    response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
    });
  } catch (err) {
    const reason = err instanceof Error ? err.message : 'Connection failed';
    throw new ApiError(`Unable to connect to backend at ${endpoint}: ${reason}`, 'CONNECTION_REFUSED');
  }

  if (!response.ok) {
    let errorCode: string | undefined;
    let errorMessage: string | undefined;

    try {
      const body = (await response.json()) as ApiErrorBody;
      errorCode = body?.error?.code;
      errorMessage = body?.error?.message;
    } catch {
      // response is not JSON
    }

    throw new ApiError(
      errorMessage || `Fetch email failed with status ${response.status}`,
      errorCode || (response.status === 422 ? 'EMAIL_PARSE_FAILED' : 'EMAIL_NOT_FOUND'),
      response.status
    );
  }

  try {
    return (await response.json()) as ParsedEmailResponse;
  } catch {
    throw new ApiError('Backend returned malformed non-JSON data when fetching parsed email.', 'MALFORMED_RESPONSE', response.status);
  }
}

/**
 * Fetch health status from backend endpoint GET /api/health.
 */
export async function fetchBackendHealth(baseUrl: string = API_BASE_URL): Promise<HealthResponse> {
  const normalizedBase = baseUrl.replace(/\/+$/, '');
  const endpoint = `${normalizedBase}/api/health`;

  let response: Response;
  try {
    response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
    });
  } catch (err) {
    const reason = err instanceof Error ? err.message : 'Connection failed';
    throw new ApiError(`Unable to connect to backend at ${endpoint}: ${reason}`, 'CONNECTION_REFUSED');
  }

  if (!response.ok) {
    let errorCode: string | undefined;
    let errorMessage: string | undefined;

    try {
      const body = (await response.json()) as ApiErrorBody;
      errorCode = body?.error?.code;
      errorMessage = body?.error?.message;
    } catch {
      // Body is not JSON
    }

    throw new ApiError(
      errorMessage || `Backend responded with HTTP status ${response.status}`,
      errorCode,
      response.status
    );
  }

  return (await response.json()) as HealthResponse;
}
