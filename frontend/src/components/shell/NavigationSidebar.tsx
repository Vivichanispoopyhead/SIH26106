import React from 'react';
import './NavigationSidebar.css';

interface NavigationSidebarProps {
  activeStage: 'ingest' | 'investigation';
  hasEmail: boolean;
  onNewAnalysis?: () => void;
}

const stages = [
  { number: '01', label: 'Case overview', key: 'investigation' as const, available: true },
  { number: '02', label: 'Email message', key: 'investigation' as const, available: true },
  { number: '03', label: 'Headers & auth', key: 'investigation' as const, available: true },
  { number: '04', label: 'Relay timeline', key: 'investigation' as const, available: true },
  { number: '05', label: 'Indicators', key: 'investigation' as const, available: true },
  { number: '06', label: 'AI assessment', key: 'investigation' as const, available: true },
  { number: '07', label: 'Entity graph', key: 'investigation' as const, available: true },
  { number: '08', label: 'Infrastructure', key: 'investigation' as const, available: true },
  { number: '09', label: 'Evidence vault', key: 'investigation' as const, available: true },
  { number: '10', label: 'Report', key: 'investigation' as const, available: true },
];

export const NavigationSidebar: React.FC<NavigationSidebarProps> = ({ activeStage, hasEmail, onNewAnalysis }) => (
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
      {stages.map((stage, index) => {
        const enabled = stage.available && hasEmail;
        return (
          <div
            key={stage.number}
            className={`stage-item ${activeStage === stage.key && enabled && index === 0 ? 'active' : ''} ${!enabled ? 'disabled' : ''}`}
            aria-disabled={!enabled}
            title={enabled ? stage.label : 'Available after the next analysis stage'}
          >
            <span className="stage-number">{stage.number}</span>
            <span className="stage-label">{stage.label}</span>
            {!stage.available && <span className="stage-state">soon</span>}
          </div>
        );
      })}
    </nav>

    <div className="sidebar-note">
      <span className="sidebar-note-mark">i</span>
      <p>Derived findings will always retain a link to their observed evidence.</p>
    </div>
  </aside>
);
