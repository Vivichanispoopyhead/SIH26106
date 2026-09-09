import React from 'react';
import { AttachmentMetadata } from '../../../types/api';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import './AttachmentList.css';

interface AttachmentListProps {
  attachments?: AttachmentMetadata[];
}

export const AttachmentList: React.FC<AttachmentListProps> = ({ attachments = [] }) => {
  const formatSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  };

  return (
    <div className="forensic-card" data-testid="attachment-list-container">
      <div className="card-header">
        <div className="title-group">
          <span className="section-eyebrow">MIME Payloads</span>
          <div className="title-with-count">
            <h3 className="card-title">Detected Attachments</h3>
            <span className="count-badge" data-testid="attachment-count-badge">
              {attachments.length} {attachments.length === 1 ? 'file' : 'files'}
            </span>
          </div>
        </div>
        <ProvenanceBadge classification="OBSERVED" />
      </div>

      {attachments.length === 0 ? (
        <p className="no-attachments-text">No attachments detected in MIME structure.</p>
      ) : (
        <div className="table-viewport">
          <table className="forensic-table" data-testid="attachments-table">
            <thead>
              <tr>
                <th>Filename</th>
                <th>MIME Type</th>
                <th style={{ width: '100px' }}>Size</th>
                <th>SHA-256 Digest</th>
              </tr>
            </thead>
            <tbody>
              {attachments.map((att, idx) => (
                <tr key={`${att.filename}-${idx}`} className="attachment-row">
                  <td>
                    <div className="att-file-cell">
                      <span className="file-glyph" aria-hidden="true">📎</span>
                      <MonoValue value={att.filename} label="Filename" copyable={false} />
                    </div>
                  </td>
                  <td>
                    <MonoValue value={att.mime_type} label="MIME Type" copyable={false} />
                  </td>
                  <td className="mono">
                    <span className="size-label" title={`${att.size_bytes} bytes`}>
                      {formatSize(att.size_bytes)}
                    </span>
                  </td>
                  <td>
                    <MonoValue value={att.sha256} label="SHA-256" copyable={true} truncate={true} />
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
