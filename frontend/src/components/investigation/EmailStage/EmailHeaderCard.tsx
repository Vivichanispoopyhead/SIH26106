import React from 'react';
import { MessageMetadata } from '../../../types/api';
import { ProvenanceBadge } from '../../common/ProvenanceBadge';
import { MonoValue } from '../../common/MonoValue';
import './EmailHeaderCard.css';

interface EmailHeaderCardProps {
  message?: MessageMetadata;
  filename?: string;
}

export const EmailHeaderCard: React.FC<EmailHeaderCardProps> = ({
  message,
  filename,
}) => {
  if (!message) {
    return (
      <div className="forensic-card" data-testid="email-header-card">
        <div className="card-header">
          <h3 className="card-title">Message Envelope & Metadata</h3>
          <ProvenanceBadge classification="OBSERVED" />
        </div>
        <p className="empty-message-note">No message metadata available.</p>
      </div>
    );
  }

  const renderAddressList = (addresses?: string[], label?: string) => {
    if (!addresses || addresses.length === 0) {
      return <span className="text-muted italic">None</span>;
    }

    return (
      <div className="address-list">
        {addresses.map((addr, idx) => (
          <MonoValue key={idx} value={addr} label={label} />
        ))}
      </div>
    );
  };

  return (
    <div className="forensic-card" data-testid="email-header-card">
      <div className="card-header">
        <div className="title-group">
          <span className="section-eyebrow">RFC 5322 Envelope</span>
          <h3 className="card-title">
            {message.subject ? message.subject : <span className="text-muted">(No Subject)</span>}
          </h3>
        </div>
        <ProvenanceBadge classification="OBSERVED" />
      </div>

      <div className="envelope-grid">
        <div className="envelope-field">
          <span className="field-label">From</span>
          <div className="field-value">
            {renderAddressList(message.from, 'From')}
          </div>
        </div>

        <div className="envelope-field">
          <span className="field-label">To</span>
          <div className="field-value">
            {renderAddressList(message.to, 'To')}
          </div>
        </div>

        <div className="envelope-field">
          <span className="field-label">Cc</span>
          <div className="field-value">
            {renderAddressList(message.cc, 'Cc')}
          </div>
        </div>

        <div className="envelope-field">
          <span className="field-label">Reply-To</span>
          <div className="field-value">
            {renderAddressList(message.reply_to, 'Reply-To')}
          </div>
        </div>

        <div className="envelope-field">
          <span className="field-label">Date</span>
          <div className="field-value">
            {message.date ? (
              <MonoValue value={message.date} label="Date" copyable={false} />
            ) : (
              <span className="text-muted">Not specified</span>
            )}
          </div>
        </div>

        <div className="envelope-field">
          <span className="field-label">Return-Path</span>
          <div className="field-value">
            {message.return_path ? (
              <MonoValue value={message.return_path} label="Return-Path" />
            ) : (
              <span className="text-muted">Not specified</span>
            )}
          </div>
        </div>

        <div className="envelope-field full-width">
          <span className="field-label">Message-ID</span>
          <div className="field-value">
            {message.message_id ? (
              <MonoValue value={message.message_id} label="Message-ID" />
            ) : (
              <span className="text-muted">Not present in headers</span>
            )}
          </div>
        </div>

        {filename && (
          <div className="envelope-field full-width">
            <span className="field-label">Source Sample</span>
            <div className="field-value">
              <MonoValue value={filename} label="Sample file" copyable={false} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
