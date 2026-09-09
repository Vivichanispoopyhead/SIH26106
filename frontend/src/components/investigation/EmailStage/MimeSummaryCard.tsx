import React from 'react';
import { MimeInfo } from '../../../types/api';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { Badge } from '../../common/Badge';
import { MonoValue } from '../../common/MonoValue';
import './MimeSummaryCard.css';

interface MimeSummaryCardProps {
  mime?: MimeInfo;
}

export const MimeSummaryCard: React.FC<MimeSummaryCardProps> = ({ mime }) => {
  if (!mime) {
    return (
      <div className="forensic-card" data-testid="mime-summary-card">
        <div className="card-header">
          <h3 className="card-title">MIME Structure & Body Attributes</h3>
          <ProvenanceBadge classification="OBSERVED" />
        </div>
        <p className="text-muted">No MIME information reported by backend parser.</p>
      </div>
    );
  }

  return (
    <div className="forensic-card" data-testid="mime-summary-card">
      <div className="card-header">
        <div className="title-group">
          <span className="section-eyebrow">Structural Attributes</span>
          <h3 className="card-title">MIME Structure</h3>
        </div>
        <ProvenanceBadge classification="OBSERVED" />
      </div>

      <div className="mime-grid">
        <div className="mime-item">
          <span className="mime-label">Root Content-Type</span>
          <div className="mime-value" data-testid="mime-content-type">
            <MonoValue value={mime.content_type || 'unspecified'} label="Content-Type" copyable={false} />
          </div>
        </div>

        <div className="mime-item">
          <span className="mime-label">Plain Text Part</span>
          <div className="mime-value" data-testid="mime-has-plain-text">
            {mime.has_plain_text ? (
              <Badge variant="clean">Present</Badge>
            ) : (
              <Badge variant="muted">Absent</Badge>
            )}
          </div>
        </div>

        <div className="mime-item">
          <span className="mime-label">HTML Part</span>
          <div className="mime-value" data-testid="mime-has-html">
            {mime.has_html ? (
              <Badge variant="cyan">Present</Badge>
            ) : (
              <Badge variant="muted">Absent</Badge>
            )}
          </div>
        </div>

        <div className="mime-item">
          <span className="mime-label">Attachment Count</span>
          <div className="mime-value" data-testid="mime-attachment-count">
            <span className="count-pill">
              {mime.attachment_count} {mime.attachment_count === 1 ? 'part' : 'parts'}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
