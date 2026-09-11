import React, { useEffect } from 'react';
import { EvidenceItem } from '../../../types/api';
import { EpistemicClass } from '../../../types/provenance';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import './EvidenceDrawer.css';
import { normalizeEvidence } from '../../../services/normalize';

export interface EvidenceDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  evidenceItem?: EvidenceItem | null;
  onSelectSignalCode?: (code: string) => void;
}

export const EvidenceDrawer: React.FC<EvidenceDrawerProps> = ({
  isOpen,
  onClose,
  evidenceItem: rawEvidenceItem,
  onSelectSignalCode,
}) => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen || !rawEvidenceItem) {
    return null;
  }
  const evidenceItem = normalizeEvidence({ evidence: [rawEvidenceItem] }).evidence[0];
  if (!evidenceItem) return null;

  return (
    <aside
      className="evidence-drawer-container"
      data-testid="evidence-drawer"
      aria-label="Forensic Evidence Inspector"
      role="complementary"
    >
      {/* Drawer Sticky Header */}
      <div className="drawer-header">
        <div className="drawer-title-group">
          <span className="drawer-eyebrow">FORENSIC EVIDENCE INSPECTOR</span>
          <h3 className="drawer-title" data-testid="drawer-evidence-id">
            <MonoValue value={evidenceItem.evidence_id} label="Evidence ID" copyable={true} />
          </h3>
        </div>
        <div className="drawer-header-actions">
          <ProvenanceBadge classification={evidenceItem.provenance as EpistemicClass} />
          <button
            type="button"
            className="drawer-close-btn"
            onClick={onClose}
            aria-label="Close evidence inspector drawer"
            data-testid="close-evidence-drawer-btn"
          >
            ×
          </button>
        </div>
      </div>

      {/* Drawer Body */}
      <div className="drawer-body">
        {/* Source metadata block */}
        <section className="drawer-section">
          <h4 className="section-label">OBSERVED PROVENANCE</h4>
          <div className="provenance-detail-grid">
            <div className="detail-row">
              <span className="detail-name">TYPE:</span>
              <span className="type-badge">{evidenceItem.type}</span>
            </div>
            <div className="detail-row">
              <span className="detail-name">SOURCE:</span>
              <code className="source-mono">{evidenceItem.source}</code>
            </div>
            {evidenceItem.header_order != null && (
              <div className="detail-row">
                <span className="detail-name">HEADER ORDER:</span>
                <span className="order-mono">#{evidenceItem.header_order}</span>
              </div>
            )}
            {evidenceItem.hash && (
              <div className="detail-row">
                <span className="detail-name">SHA-256 HASH:</span>
                <MonoValue value={evidenceItem.hash} label="SHA-256" copyable={true} truncate={true} />
              </div>
            )}
          </div>
        </section>

        {/* Snippet or Value block */}
        <section className="drawer-section">
          <h4 className="section-label">
            {evidenceItem.snippet ? 'RAW ARTIFACT SNIPPET' : 'OBSERVED ARTIFACT VALUE'}
          </h4>
          {evidenceItem.snippet ? (
            <div className="drawer-snippet-box" data-testid="drawer-snippet">
              <pre className="drawer-snippet-pre font-mono">
                <code>{evidenceItem.snippet}</code>
              </pre>
            </div>
          ) : (
            <div className="drawer-value-box font-mono" data-testid="drawer-value">
              <MonoValue value={evidenceItem.value} label={evidenceItem.source} copyable={true} />
            </div>
          )}
        </section>

        {/* Related Signal Codes */}
        {evidenceItem.related_signal_codes && evidenceItem.related_signal_codes.length > 0 && (
          <section className="drawer-section">
            <h4 className="section-label">CORROBORATED RISK SIGNALS</h4>
            <div className="signals-chips-grid">
              {evidenceItem.related_signal_codes.map((code) => (
                <button
                  key={code}
                  type="button"
                  className="drawer-signal-btn"
                  onClick={() => onSelectSignalCode?.(code)}
                  title={`Filter evidence list by ${code}`}
                  data-testid={`drawer-signal-${code}`}
                >
                  ⚡ {code}
                </button>
              ))}
            </div>
          </section>
        )}

        {/* Forensic Disclaimers */}
        <section className="drawer-disclaimer-box">
          <p className="disclaimer-note">
            <strong>Chain of Custody:</strong> This evidence record represents an unmodifiable observed or derived forensic artifact tied to the case ledger.
          </p>
        </section>
      </div>
    </aside>
  );
};
