/**
 * Central frontend environment configuration.
 * Single source of truth for runtime API location.
 */
export const API_BASE_URL: string =
  import.meta.env.VITE_API_BASE_URL !== undefined && import.meta.env.VITE_API_BASE_URL !== ''
    ? import.meta.env.VITE_API_BASE_URL
    : 'http://localhost:8080';
