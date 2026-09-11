import React, { useState, useMemo } from 'react';
import { EmailEvidenceResponse, EvidenceItem, EvidenceType } from '../../../types/api';
import { EpistemicClass } from '../../../types/provenance';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import { ErrorBanner } from '../../states/ErrorBanner';
import './EvidencePanel.css';
import { normalizeEvidence } from '../../../services/normalize';

export interface EvidencePanelProps {
  evidenceData?: EmailEvidenceResponse | null;
  isLoading?: boolean;
  error?: unknown | null;
  selectedSignalCode?: string | null;
  selectedQuery?: string | null;
  onClearFilter?: () => void;
  onSelectSignalCode?: (code: string) => void;
  onSelectEvidenceItem?: (item: EvidenceItem) => void;
  onRetry?: () => void;
}

export const EvidencePanel: React.FC<EvidencePanelProps> = ({
  evidenceData,
  isLoading = false,
  error = null,
  selectedSignalCode = null,
  selectedQuery = null,
  onClearFilter,
  onSelectSignalCode,
  onSelectEvidenceItem,
  onRetry,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState<EvidenceType | 'all'>('all');
  const [provenanceFilter, setProvenanceFilter] = useState<string>('all');

  const evidenceList: EvidenceItem[] = useMemo(() => {
    return normalizeEvidence(evidenceData).evidence;
  }, [evidenceData]);

  // Filter evidence based on props (selectedSignalCode, selectedQuery) and local filters
  const filteredEvidence = useMemo(() => {
    return evidenceList.filter((item) => {
      // 1. Filter by selectedSignalCode from props
      if (selectedSignalCode) {
        const matchesSignal = item.related_signal_codes?.some(
          (code) => code.toLowerCase() === selectedSignalCode.toLowerCase()
        );
        if (!matchesSignal) return false;
      }

      // 2. Filter by selectedQuery from props (e.g. indicator value or AI reference)
      if (selectedQuery) {
        const queryLower = selectedQuery.toLowerCase();
        const matchesQuery =
          item.source.toLowerCase().includes(queryLower) ||
          item.value.toLowerCase().includes(queryLower) ||
          (item.snippet && item.snippet.toLowerCase().includes(queryLower)) ||
          item.evidence_id.toLowerCase().includes(queryLower);
        if (!matchesQuery) return false;
      }

      // 3. Filter by type
      if (typeFilter !== 'all' && item.type !== typeFilter) {
        return false;
      }

      // 4. Filter by provenance
      if (provenanceFilter !== 'all' && item.provenance !== provenanceFilter) {
        return false;
      }

      // 5. Filter by local search term
      if (searchTerm.trim()) {
        const term = searchTerm.toLowerCase();
        const matchesSearch =
          item.evidence_id.toLowerCase().includes(term) ||
          item.source.toLowerCase().includes(term) ||
          item.value.toLowerCase().includes(term) ||
          (item.snippet && item.snippet.toLowerCase().includes(term)) ||
          item.related_signal_codes?.some((c) => c.toLowerCase().includes(term));
        if (!matchesSearch) return false;
      }

      return true;
    });
  }, [evidenceList, selectedSignalCode, selectedQuery, typeFilter, provenanceFilter, searchTerm]);

  // Unique evidence types available in this dataset
  const availableTypes = useMemo(() => {
    const types = new Set<string>();
    evidenceList.forEach((e) => {
      if (e.type) types.add(e.type);
    });
    return Array.from(types).sort();
  }, [evidenceList]);

  const hasActiveFilter = Boolean(
    selectedSignalCode ||
    selectedQuery ||
    typeFilter !== 'all' ||
    provenanceFilter !== 'all' ||
    searchTerm.trim()
  );

  const handleClearAll = () => {
    setSearchTerm('');
    setTypeFilter('all');
    setProvenanceFilter('all');
    onClearFilter?.();
  };

  // 1. Loading State
  if (isLoading) {
    return (
      <div className="evidence-panel-container" data-testid="evidence-panel">
        <div className="evidence-panel-header">
          <div className="title-group">
            <span className="section-eyebrow">Forensic Chain of Custody</span>
            <h3 className="panel-title">Evidence & Provenance Vault</h3>
          </div>
          <ProvenanceBadge classification="OBSERVED" />
        </div>

        <div className="evidence-loading-state" data-testid="evidence-loading">
          <span className="status-dot pulsing" aria-hidden="true" />
          <span className="loading-text">
            Retrieving cryptographically verifiable evidence and provenance records...
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="evidence-panel-container" data-testid="evidence-panel">
      {/* Panel Header */}
      <div className="evidence-panel-header">
        <div className="title-group">
          <span className="section-eyebrow">Forensic Chain of Custody</span>
          <div className="title-with-count">
            <h3 className="panel-title">Evidence & Provenance Vault</h3>
            <span className="count-badge" data-testid="evidence-count-badge">
              {evidenceList.length} {evidenceList.length === 1 ? 'record' : 'records'}
            </span>
          </div>
        </div>
        <div className="header-badges">
          <ProvenanceBadge classification="OBSERVED" />
        </div>
      </div>

      {/* Forensic Epistemic Integrity Disclaimers */}
      <div className="evidence-disclaimers-card" data-testid="evidence-disclaimers">
        <div className="disclaimer-row">
          <span className="disclaimer-bullet" aria-hidden="true">✦</span>
          <span className="disclaimer-text">
            <strong>AI Assessment:</strong> AI findings are evaluated assessments, not ground truth.
          </span>
        </div>
        <div className="disclaimer-row">
          <span className="disclaimer-bullet" aria-hidden="true">🌐</span>
          <span className="disclaimer-text">
            <strong>Geolocation:</strong> Estimated IP location; does not prove physical actor presence.
          </span>
        </div>
        <div className="disclaimer-row">
          <span className="disclaimer-bullet" aria-hidden="true">⚖</span>
          <span className="disclaimer-text">
            <strong>Enrichment:</strong> Enriched data is not proof of identity or maliciousness.
          </span>
        </div>
      </div>

      {/* Non-fatal Error Banner if evidence failed to load */}
      {error !== null && error !== undefined && (
        <div className="evidence-error-container" data-testid="evidence-error">
          <ErrorBanner
            error={error}
            onRetry={onRetry}
          />
        </div>
      )}

      {/* Filter and Search Bar */}
      <div className="evidence-controls-bar">
        <div className="search-input-wrapper">
          <span className="search-icon" aria-hidden="true">🔍</span>
          <input
            type="text"
            className="evidence-search-input"
            placeholder="Search evidence by ID, source, value, snippet, or signal..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            aria-label="Search evidence records"
            data-testid="evidence-search-input"
          />
          {searchTerm && (
            <button
              type="button"
              className="clear-search-btn"
              onClick={() => setSearchTerm('')}
              aria-label="Clear search"
            >
              ×
            </button>
          )}
        </div>

        <div className="filter-dropdowns">
          <select
            className="filter-select"
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            aria-label="Filter by evidence type"
            data-testid="evidence-type-filter"
          >
            <option value="all">All Types ({evidenceList.length})</option>
            {availableTypes.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>

          <select
            className="filter-select"
            value={provenanceFilter}
            onChange={(e) => setProvenanceFilter(e.target.value)}
            aria-label="Filter by provenance class"
            data-testid="evidence-provenance-filter"
          >
            <option value="all">All Provenance</option>
            <option value="OBSERVED">OBSERVED</option>
            <option value="ENRICHED">ENRICHED</option>
            <option value="INFERRED">INFERRED</option>
            <option value="AI-ASSESSED">AI-ASSESSED</option>
          </select>
        </div>
      </div>

      {/* Active Filter Notification Ribbon */}
      {hasActiveFilter && (
        <div className="active-filter-ribbon" data-testid="active-filter-ribbon">
          <span className="filter-tag-label">Active Filter:</span>
          {selectedSignalCode && (
            <span className="filter-pill" data-testid="active-signal-pill">
              Signal: <code>{selectedSignalCode}</code>
            </span>
          )}
          {selectedQuery && (
            <span className="filter-pill" data-testid="active-query-pill">
              Query: <code>{selectedQuery}</code>
            </span>
          )}
          {typeFilter !== 'all' && (
            <span className="filter-pill">Type: {typeFilter}</span>
          )}
          {provenanceFilter !== 'all' && (
            <span className="filter-pill">Provenance: {provenanceFilter}</span>
          )}
          {searchTerm.trim() && (
            <span className="filter-pill">Text: &quot;{searchTerm}&quot;</span>
          )}
          <span className="filter-matches-text">
            Showing {filteredEvidence.length} of {evidenceList.length} records
          </span>
          <button
            type="button"
            className="btn-clear-filter"
            onClick={handleClearAll}
            data-testid="clear-evidence-filter-btn"
          >
            Clear Filter ×
          </button>
        </div>
      )}

      {/* Evidence Items List */}
      <div className="evidence-list-viewport">
        {evidenceList.length === 0 ? (
          <div className="evidence-empty-card" data-testid="evidence-empty">
            <div className="empty-icon" aria-hidden="true">📋</div>
            <h4 className="empty-title">No Evidence Available</h4>
            <p className="empty-text">
              No verified evidence records are available for this analysis.
            </p>
          </div>
        ) : filteredEvidence.length === 0 ? (
          <div className="evidence-empty-card" data-testid="evidence-no-matches">
            <h4 className="empty-title">No Evidence Matches Filter</h4>
            <p className="empty-text">
              No evidence records matched your current query or signal filter criteria.
            </p>
            <button
              type="button"
              className="btn-clear-filter inline-btn"
              onClick={handleClearAll}
              data-testid="reset-filter-btn"
            >
              Reset Filters
            </button>
          </div>
        ) : (
          <div className="evidence-cards-grid" data-testid="evidence-cards-grid">
            {filteredEvidence.map((item) => (
              <article
                key={item.evidence_id}
                className="evidence-item-card"
                data-testid="evidence-item"
                data-evidence-id={item.evidence_id}
              >
                {/* Item Top Bar */}
                <div className="item-top-bar">
                  <div className="item-identifiers">
                    <span className="evidence-id-label">ID:</span>
                    <MonoValue value={item.evidence_id} label="Evidence ID" copyable={true} />
                    <span className={`evidence-type-chip type-${item.type.toLowerCase().replace(/[^a-z0-9]/g, '-')}`}>
                      {item.type}
                    </span>
                    {item.header_order != null && (
                      <span className="header-order-pill" data-testid="header-order-pill">
                        Order #{item.header_order}
                      </span>
                    )}
                  </div>
                  <div className="item-top-actions">
                    <ProvenanceBadge classification={item.provenance as EpistemicClass} />
                    {onSelectEvidenceItem && (
                      <button
                        type="button"
                        className="inspect-evidence-btn"
                        onClick={() => onSelectEvidenceItem(item)}
                        data-testid={`inspect-evidence-${item.evidence_id}`}
                        title="Open evidence inspector drawer"
                      >
                        Inspect ↗
                      </button>
                    )}
                  </div>
                </div>

                {/* Source Line */}
                <div className="item-source-row">
                  <span className="source-label">Source:</span>
                  <code className="source-value">{item.source}</code>
                </div>

                {/* Snippet / Value Display */}
                {item.snippet ? (
                  <div className="item-snippet-wrapper" data-testid="evidence-snippet">
                    <div className="snippet-header">
                      <span className="snippet-caption">VERIFIABLE ARTIFACT SNIPPET</span>
                    </div>
                    <pre className="snippet-block font-mono">
                      <code>{item.snippet}</code>
                    </pre>
                  </div>
                ) : item.value ? (
                  <div className="item-value-wrapper" data-testid="evidence-value">
                    <div className="snippet-header">
                      <span className="snippet-caption">EXTRACTED VALUE</span>
                    </div>
                    <div className="value-box font-mono">
                      <MonoValue value={item.value} label={item.source} copyable={true} />
                    </div>
                  </div>
                ) : null}

                {/* Hash Display when present */}
                {item.hash && (
                  <div className="item-hash-row" data-testid="evidence-hash">
                    <span className="hash-label">ARTIFACT SHA-256:</span>
                    <MonoValue value={item.hash} label="SHA-256" copyable={true} truncate={true} />
                  </div>
                )}

                {/* Related Signal Codes */}
                {item.related_signal_codes && item.related_signal_codes.length > 0 && (
                  <div className="item-signals-row" data-testid="related-signals-container">
                    <span className="signals-label">RELATED SIGNALS:</span>
                    <div className="signal-codes-list">
                      {item.related_signal_codes.map((code) => (
                        <button
                          key={code}
                          type="button"
                          className={`signal-code-btn ${selectedSignalCode === code ? 'active' : ''}`}
                          onClick={() => onSelectSignalCode?.(code)}
                          title={`Filter evidence by signal code ${code}`}
                          data-testid={`signal-code-${code}`}
                        >
                          ⚡ {code}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
