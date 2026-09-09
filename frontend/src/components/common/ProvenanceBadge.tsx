import React from 'react';
import { EpistemicClass } from '../../types/provenance';
import './ProvenanceBadge.css';

interface ProvenanceBadgeProps {
  classification?: EpistemicClass;
  className?: string;
  showIcon?: boolean;
}

const ICONS: Record<EpistemicClass, string> = {
  OBSERVED: '●',
  ENRICHED: '🌐',
  INFERRED: '⚡',
  'AI-ASSESSED': '✦',
};

export const ProvenanceBadge: React.FC<ProvenanceBadgeProps> = ({
  classification = 'OBSERVED',
  className = '',
  showIcon = true,
}) => {
  const icon = ICONS[classification] || '●';
  const classKey = classification.toLowerCase().replace(/[^a-z0-9]/g, '-');

  return (
    <span
      className={`provenance-badge provenance-${classKey} ${className}`}
      title={`Forensic Epistemic Class: ${classification}`}
      data-testid="provenance-badge"
    >
      {showIcon && <span className="provenance-icon" aria-hidden="true">{icon}</span>}
      <span className="provenance-label">{classification}</span>
    </span>
  );
};
