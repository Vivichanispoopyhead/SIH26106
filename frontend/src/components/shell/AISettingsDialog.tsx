import React, { useEffect, useState } from 'react';
import { configureAIProvider, getAIProviderStatus, AIProviderStatus } from '../../services/api';
import './AISettingsDialog.css';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const AISettingsDialog: React.FC<Props> = ({ isOpen, onClose }) => {
  const [status, setStatus] = useState<AIProviderStatus | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('gemini-2.5-flash');
  const [message, setMessage] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setMessage('');
    setApiKey('');
    getAIProviderStatus().then((value) => {
      setStatus(value);
      if (value.model) setModel(value.model);
    }).catch(() => setMessage('Could not load AI provider status.'));
  }, [isOpen]);

  if (!isOpen) return null;

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    setMessage('');
    setSaving(true);
    try {
      const value = await configureAIProvider(apiKey, model);
      setStatus(value);
      setApiKey('');
      setMessage('AI provider configured for this backend session.');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Configuration failed.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="ai-settings-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="ai-settings-dialog" role="dialog" aria-modal="true" aria-labelledby="ai-settings-title">
        <div className="ai-settings-header">
          <div><span className="section-eyebrow">Provider configuration</span><h2 id="ai-settings-title">AI assessment settings</h2></div>
          <button type="button" className="ai-settings-close" onClick={onClose} aria-label="Close AI settings">×</button>
        </div>
        <p className="ai-settings-copy">Your key is sent directly to the backend over this connection, kept in memory, and never displayed or returned by the API.</p>
        <form onSubmit={save}>
          <label>Provider<select value="google" disabled><option value="google">Google Gemini</option></select></label>
          <label>Model<input value={model} onChange={(event) => setModel(event.target.value)} placeholder="gemini-2.5-flash" /></label>
          <label>API key<input type="password" value={apiKey} onChange={(event) => setApiKey(event.target.value)} placeholder={status?.configured ? 'Configured — enter a new key to replace it' : 'Paste your Gemini API key'} autoComplete="off" required /></label>
          {message && <p className="ai-settings-message" role="status">{message}</p>}
          <div className="ai-settings-actions"><button type="button" className="ai-settings-secondary" onClick={onClose}>Cancel</button><button type="submit" className="ai-settings-primary" disabled={saving || !apiKey.trim()}>{saving ? 'Saving…' : 'Save provider key'}</button></div>
        </form>
      </section>
    </div>
  );
};
