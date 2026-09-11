import React, { useState, useMemo } from 'react';
import { IndicatorSet } from '../../../types/api';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import './IndicatorTable.css';

interface IndicatorTableProps {
  indicators?: IndicatorSet;
  onSelectIndicator?: (value: string) => void;
}

type IndicatorTab = 'all' | 'ips' | 'domains' | 'urls';

export const IndicatorTable: React.FC<IndicatorTableProps> = ({ indicators, onSelectIndicator }) => {
  const [activeTab, setActiveTab] = useState<IndicatorTab>('all');
  const [searchTerm, setSearchTerm] = useState('');

  const ips = indicators?.ips || [];
  const domains = indicators?.domains || [];
  const urls = indicators?.urls || [];

  const totalCount = ips.length + domains.length + urls.length;

  const allIndicators = useMemo(() => {
    const list: Array<{ type: 'ip' | 'domain' | 'url'; value: string }> = [];
    ips.forEach((ip) => list.push({ type: 'ip', value: ip }));
    domains.forEach((d) => list.push({ type: 'domain', value: d }));
    urls.forEach((u) => list.push({ type: 'url', value: u }));
    return list;
  }, [ips, domains, urls]);

  const displayedList = useMemo(() => {
    let filtered = allIndicators;
    if (activeTab === 'ips') filtered = filtered.filter((i) => i.type === 'ip');
    if (activeTab === 'domains') filtered = filtered.filter((i) => i.type === 'domain');
    if (activeTab === 'urls') filtered = filtered.filter((i) => i.type === 'url');

    if (searchTerm.trim()) {
      const term = searchTerm.toLowerCase();
      filtered = filtered.filter((i) => i.value.toLowerCase().includes(term));
    }

    return filtered;
  }, [allIndicators, activeTab, searchTerm]);

  return (
    <div className="forensic-card" data-testid="indicator-table-container">
      <div className="card-header">
        <div className="title-group">
          <span className="section-eyebrow">Extracted IOCs</span>
          <div className="title-with-count">
            <h3 className="card-title">Extracted Indicators</h3>
            <span className="count-badge" data-testid="indicators-total-count">
              {totalCount} Total
            </span>
          </div>
        </div>
        <ProvenanceBadge classification="OBSERVED" />
      </div>

      <div className="indicator-nav-bar">
        <div className="indicator-tabs" role="tablist">
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'all'}
            className={`tab-btn ${activeTab === 'all' ? 'active' : ''}`}
            onClick={() => setActiveTab('all')}
            data-testid="tab-all-indicators"
          >
            All ({totalCount})
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'ips'}
            className={`tab-btn ${activeTab === 'ips' ? 'active' : ''}`}
            onClick={() => setActiveTab('ips')}
            data-testid="tab-ips"
          >
            IP Addresses ({ips.length})
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'domains'}
            className={`tab-btn ${activeTab === 'domains' ? 'active' : ''}`}
            onClick={() => setActiveTab('domains')}
            data-testid="tab-domains"
          >
            Domains ({domains.length})
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === 'urls'}
            className={`tab-btn ${activeTab === 'urls' ? 'active' : ''}`}
            onClick={() => setActiveTab('urls')}
            data-testid="tab-urls"
          >
            URLs ({urls.length})
          </button>
        </div>

        <div className="indicator-search-wrapper">
          <input
            type="text"
            className="indicator-search-input"
            placeholder="Search indicators..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            aria-label="Search indicators"
            data-testid="indicator-search-input"
          />
          {searchTerm && (
            <button
              type="button"
              className="clear-search-btn"
              onClick={() => setSearchTerm('')}
              aria-label="Clear indicator search"
            >
              ×
            </button>
          )}
        </div>
      </div>

      {totalCount === 0 ? (
        <p className="empty-indicators-text">No network indicators observed in message.</p>
      ) : displayedList.length === 0 ? (
        <p className="no-matches-text">No indicators match &quot;{searchTerm}&quot;.</p>
      ) : (
        <div className="indicator-list-viewport">
          <table className="forensic-table" data-testid="indicator-table">
            <thead>
              <tr>
                <th style={{ width: '90px' }}>Type</th>
                <th>Indicator Value</th>
                <th style={{ width: '130px' }}>Provenance</th>
                {onSelectIndicator && <th style={{ width: '100px' }}>Evidence</th>}
              </tr>
            </thead>
            <tbody>
              {displayedList.map((item, index) => (
                <tr key={`${item.type}-${item.value}-${index}`} className="indicator-row">
                  <td>
                    <span className={`ioc-type-chip type-${item.type}`}>
                      {item.type.toUpperCase()}
                    </span>
                  </td>
                  <td>
                    <MonoValue value={item.value} label={item.type.toUpperCase()} copyable={true} />
                  </td>
                  <td>
                    <ProvenanceBadge classification="OBSERVED" showIcon={false} />
                  </td>
                  {onSelectIndicator && (
                    <td>
                      <button
                        type="button"
                        className="btn-inspect-ioc"
                        onClick={() => onSelectIndicator(item.value)}
                        data-testid={`inspect-ioc-${item.value}`}
                        title={`Find evidence matching ${item.value}`}
                      >
                        Evidence ↗
                      </button>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
