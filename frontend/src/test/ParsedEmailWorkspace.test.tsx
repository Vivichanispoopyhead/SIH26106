import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { ParsedEmailWorkspace } from '../components/investigation/ParsedEmailWorkspace';
import { ParsedEmailResponse } from '../types/api';

const mockParsedEmail: ParsedEmailResponse = {
  email_id: 'email_01J8TEST999',
  case_id: 'case_01J8TEST111',
  status: 'parsed',
  filename: 'suspicious_invoice.eml',
  message: {
    message_id: '<msg-12345@adversary.org>',
    from: ['attacker@adversary.org'],
    to: ['victim@enterprise.com'],
    cc: ['colleague@enterprise.com'],
    reply_to: ['drop-box@relay-proxy.net'],
    subject: 'Action Required: Urgent Wire Transfer',
    date: '2026-09-08T10:20:30Z',
    return_path: 'bounce@adversary.org',
  },
  mime: {
    content_type: 'multipart/mixed',
    has_plain_text: true,
    has_html: true,
    attachment_count: 1,
  },
  headers: [
    { name: 'From', value: 'attacker@adversary.org', order: 1 },
    { name: 'To', value: 'victim@enterprise.com', order: 2 },
    { name: 'Subject', value: 'Action Required: Urgent Wire Transfer', order: 3 },
    { name: 'Received', value: 'from mail.adversary.org (198.51.100.42) by mx.enterprise.com', order: 4 },
  ],
  indicators: {
    ips: ['198.51.100.42', '203.0.113.5'],
    domains: ['adversary.org', 'relay-proxy.net'],
    urls: ['https://adversary.org/auth/login', 'https://cdn.adversary.org/payload.exe'],
  },
  attachments: [
    {
      filename: 'invoice.pdf.exe',
      mime_type: 'application/octet-stream',
      size_bytes: 145120,
      sha256: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    },
  ],
};

describe('ParsedEmailWorkspace Component', () => {
  it('H. renders triage ribbon with identifiers and observed status', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    expect(screen.getByTestId('parsed-email-workspace')).toBeInTheDocument();
    expect(screen.getByTestId('triage-subject')).toHaveTextContent('Action Required: Urgent Wire Transfer');
    expect(screen.getByText('email_01J8TEST999')).toBeInTheDocument();
    expect(screen.getByText('case_01J8TEST111')).toBeInTheDocument();
    expect(screen.getAllByText('suspicious_invoice.eml').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByTestId('parsed-status-badge')).toHaveTextContent('PARSED');

    // All epistemic badges should report OBSERVED
    const badges = screen.getAllByTestId('provenance-badge');
    expect(badges.length).toBeGreaterThan(0);
    badges.forEach((b) => {
      expect(b).toHaveTextContent('OBSERVED');
    });
  });

  it('H. renders Message Envelope metadata correctly', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    const headerCard = screen.getByTestId('email-header-card');
    expect(within(headerCard).getByText('attacker@adversary.org')).toBeInTheDocument();
    expect(within(headerCard).getByText('victim@enterprise.com')).toBeInTheDocument();
    expect(within(headerCard).getByText('colleague@enterprise.com')).toBeInTheDocument();
    expect(within(headerCard).getByText('drop-box@relay-proxy.net')).toBeInTheDocument();
    expect(within(headerCard).getByText('<msg-12345@adversary.org>')).toBeInTheDocument();
    expect(within(headerCard).getByText('bounce@adversary.org')).toBeInTheDocument();
    expect(within(headerCard).getByText('2026-09-08T10:20:30Z')).toBeInTheDocument();
  });

  it('H. renders MIME structure attributes correctly', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    expect(screen.getByTestId('mime-content-type')).toHaveTextContent('multipart/mixed');
    expect(screen.getByTestId('mime-has-plain-text')).toHaveTextContent('Present');
    expect(screen.getByTestId('mime-has-html')).toHaveTextContent('Present');
    expect(screen.getByTestId('mime-attachment-count')).toHaveTextContent('1 part');
  });

  it('H. renders complete headers and supports search filtering', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    expect(screen.getByTestId('header-count-badge')).toHaveTextContent('4 headers');

    // Filter headers
    const searchInput = screen.getByTestId('header-search-input');
    fireEvent.change(searchInput, { target: { value: 'Received' } });

    const headerTable = screen.getByTestId('header-table');
    expect(within(headerTable).getByText('from mail.adversary.org (198.51.100.42) by mx.enterprise.com')).toBeInTheDocument();
    expect(within(headerTable).queryByText('Action Required: Urgent Wire Transfer')).not.toBeInTheDocument();
  });

  it('H. renders network indicators with tabs and filtering', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    expect(screen.getByTestId('indicators-total-count')).toHaveTextContent('6 Total');

    // Check indicator items are rendered
    expect(screen.getByText('198.51.100.42')).toBeInTheDocument();
    expect(screen.getByText('adversary.org')).toBeInTheDocument();
    expect(screen.getByText('https://adversary.org/auth/login')).toBeInTheDocument();

    // Switch to IP tab
    fireEvent.click(screen.getByTestId('tab-ips'));
    expect(screen.getByText('198.51.100.42')).toBeInTheDocument();
    expect(screen.queryByText('https://adversary.org/auth/login')).not.toBeInTheDocument();
  });

  it('H. renders detected attachments with hashes and sizes', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    expect(screen.getByTestId('attachment-count-badge')).toHaveTextContent('1 file');
    expect(screen.getByText('invoice.pdf.exe')).toBeInTheDocument();
    expect(screen.getByText('application/octet-stream')).toBeInTheDocument();
    expect(screen.getByText('141.7 KB')).toBeInTheDocument();
    expect(screen.getByText('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855')).toBeInTheDocument();
  });

  it('H. displays pipeline notice explaining subsequent stages', () => {
    render(<ParsedEmailWorkspace emailData={mockParsedEmail} />);

    const notice = screen.getByTestId('pipeline-notice');
    expect(notice).toHaveTextContent(/Stage 01: Ingestion & Parsing Complete/i);
    expect(notice).toHaveTextContent(/OBSERVED facts/i);
  });
});
