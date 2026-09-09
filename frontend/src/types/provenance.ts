/**
 * Epistemic Provenance Classes as defined in docs/ui-ux-design-system.md
 * and docs/api-contract.md.
 */
export type EpistemicClass = 'OBSERVED' | 'ENRICHED' | 'INFERRED' | 'AI-ASSESSED';

export interface EpistemicMeta {
  class: EpistemicClass;
  source?: string;
  confidence?: number;
}
