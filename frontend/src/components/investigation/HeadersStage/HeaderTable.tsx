import React, { useState, useMemo } from 'react';
import { HeaderEntry } from '../../../types/api';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import './HeaderTable.css';

interface HeaderTableProps {
  headers?: HeaderEntry[];
}

export const HeaderTable: React.FC<HeaderTableProps> = ({ headers = [] }) => {
  const [searchTerm, setSearchTerm] = useState('');

  // Headers must preserve their source email order
  const sortedHeaders = useMemo(() => {
    return [...headers].sort((a, b) => a.order - b.order);
  }, [headers]);

  const filteredHeaders = useMemo(() => {
    if (!searchTerm.trim()) return sortedHeaders;
    const term = searchTerm.toLowerCase();
    return sortedHeaders.filter(
      (h) =>
        h.name.toLowerCase().includes(term) ||
        h.value.toLowerCase().includes(term)
    );
  }, [sortedHeaders, searchTerm]);

  return (
    <div className="forensic-card" data-testid="header-table-container">
      <div className="card-header">
        <div className="title-group">
          <span className="section-eyebrow">Raw RFC 5322 Metadata</span>
          <div className="title-with-count">
            <h3 className="card-title">Parsed Email Headers</h3>
            <span className="count-badge" data-testid="header-count-badge">
              {headers.length} {headers.length === 1 ? 'header' : 'headers'}
            </span>
          </div>
        </div>
        <ProvenanceBadge classification="OBSERVED" />
      </div>

      <div className="header-controls">
        <div className="search-input-wrapper">
          <span className="search-icon" aria-hidden="true">🔍</span>
          <input
            type="text"
            className="header-search-input"
            placeholder="Filter headers by key or value..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            aria-label="Filter headers"
            data-testid="header-search-input"
          />
          {searchTerm && (
            <button
              type="button"
              className="clear-search-btn"
              onClick={() => setSearchTerm('')}
              aria-label="Clear filter"
            >
              ×
            </button>
          )}
        </div>
        {searchTerm && (
          <span className="filter-summary">
            Showing {filteredHeaders.length} of {headers.length}
          </span>
        )}
      </div>

      {headers.length === 0 ? (
        <p className="empty-headers-text">No headers parsed.</p>
      ) : filteredHeaders.length === 0 ? (
        <p className="no-matches-text">No headers match &quot;{searchTerm}&quot;.</p>
      ) : (
        <div className="table-viewport">
          <table className="forensic-table" data-testid="header-table">
            <thead>
              <tr>
                <th className="col-order">#</th>
                <th className="col-name">Header Field</th>
                <th className="col-value">Value</th>
              </tr>
            </thead>
            <tbody>
              {filteredHeaders.map((header, idx) => (
                <tr key={`${header.order}-${header.name}-${idx}`} className="header-row">
                  <td className="col-order mono">
                    <span className="order-number">{header.order}</span>
                  </td>
                  <td className="col-name mono">
                    <span className="header-name-text" title={header.name}>
                      {header.name}
                    </span>
                  </td>
                  <td className="col-value">
                    <MonoValue value={header.value} label={header.name} copyable={true} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
