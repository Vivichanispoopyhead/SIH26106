import React from 'react';
import './NavigationSidebar.css';

interface NavigationSidebarProps {
  activeStage: 'ingest' | 'investigation';
  hasEmail: boolean;
  activeView?: 'all' | 'metadata' | 'headers' | 'timeline' | 'indicators' | 'assessment' | 'graph' | 'enrichment' | 'evidence' | 'attachments' | 'report';
  onNewAnalysis?: () => void;
  onSelectView?: (view: 'all' | 'metadata' | 'headers' | 'timeline' | 'indicators' | 'assessment' | 'graph' | 'enrichment' | 'evidence' | 'attachments' | 'report') => void;
}

const stages = [
  { number: '01', label: 'Case overview', view: 'all' as const },
  { number: '02', label: 'Email message', view: 'metadata' as const },
  { number: '03', label: 'Headers & auth', view: 'headers' as const },
  { number: '04', label: 'Relay timeline', view: 'timeline' as const },
  { number: '05', label: 'Indicators', view: 'indicators' as const },
  { number: '06', label: 'AI assessment', view: 'assessment' as const },
  { number: '07', label: 'Entity graph', view: 'graph' as const },
  { number: '08', label: 'Infrastructure', view: 'enrichment' as const },
  { number: '09', label: 'Evidence vault', view: 'evidence' as const },
  { number: '10', label: 'Report', view: 'report' as const },
];

export const NavigationSidebar: React.FC<NavigationSidebarProps> = ({ activeStage, hasEmail, activeView = 'all', onNewAnalysis, onSelectView }) => (
  <aside className="navigation-sidebar" aria-label="Investigation stages">
    <button type="button" className="sidebar-new-analysis" onClick={onNewAnalysis}>
      <span className="new-analysis-plus">+</span>
      <span>New analysis</span>
    </button>
    <div className="sidebar-heading">
      <span className="sidebar-kicker">Investigation</span>
      <span className="sidebar-caption">{hasEmail ? 'Active case' : 'Awaiting artifact'}</span>
    </div>

    <nav className="stage-list">
      {stages.map((stage) => {
        const enabled = hasEmail;
        return (
          <button
            key={stage.number}
            type="button"
            className={`stage-item ${activeStage === 'investigation' && enabled && activeView === stage.view ? 'active' : ''} ${!enabled ? 'disabled' : ''}`}
            aria-disabled={!enabled}
            title={enabled ? stage.label : 'Available after an email is analyzed'}
            disabled={!enabled}
            onClick={() => onSelectView?.(stage.view)}
          >
            <span className="stage-number">{stage.number}</span>
            <span className="stage-label">{stage.label}</span>
          </button>
        );
      })}
    </nav>

    <div className="sidebar-note">
      <span className="sidebar-note-mark">i</span>
      <p>Derived findings will always retain a link to their observed evidence.</p>
    </div>
  </aside>
);
