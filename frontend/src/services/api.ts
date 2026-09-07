import { API_BASE_URL } from '../config/env';

export interface HealthResponse {
  status: string;
}

export interface ApiErrorPayload {
  error?: {
    code: string;
    message: string;
  };
}

export class ApiError extends Error {
  readonly code?: string;
  readonly status?: number;

  constructor(message: string, code?: string, status?: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
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
      const body = (await response.json()) as ApiErrorPayload;
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

  const data = (await response.json()) as HealthResponse;
  return data;
}
