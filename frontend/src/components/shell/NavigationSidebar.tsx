import React from 'react';
import './NavigationSidebar.css';

interface NavigationSidebarProps {
  activeStage: 'ingest' | 'investigation';
  hasEmail: boolean;
}

const stages = [
  { number: '01', label: 'Case overview', key: 'investigation' as const, available: true },
  { number: '02', label: 'Email message', key: 'investigation' as const, available: true },
  { number: '03', label: 'Headers & auth', key: 'investigation' as const, available: true },
  { number: '04', label: 'Relay chain', key: 'investigation' as const, available: false },
  { number: '05', label: 'Indicators', key: 'investigation' as const, available: true },
  { number: '06', label: 'AI assessment', key: 'investigation' as const, available: true },
  { number: '07', label: 'Entity graph', key: 'investigation' as const, available: false },
  { number: '08', label: 'Infrastructure map', key: 'investigation' as const, available: false },
  { number: '09', label: 'Evidence vault', key: 'investigation' as const, available: false },
  { number: '10', label: 'Report', key: 'investigation' as const, available: false },
];

export const NavigationSidebar: React.FC<NavigationSidebarProps> = ({ activeStage, hasEmail }) => (
  <aside className="navigation-sidebar" aria-label="Investigation stages">
    <div className="sidebar-heading">
      <span className="sidebar-kicker">Investigation</span>
      <span className="sidebar-caption">{hasEmail ? 'Active case' : 'Awaiting artifact'}</span>
    </div>

    <nav className="stage-list">
      {stages.map((stage) => {
        const enabled = stage.available && hasEmail;
        return (
          <div
            key={stage.number}
            className={`stage-item ${activeStage === stage.key && enabled ? 'active' : ''} ${!enabled ? 'disabled' : ''}`}
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
