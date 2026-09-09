import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { EmlUploadZone } from '../components/ingestion/EmlUploadZone';

describe('EmlUploadZone Component', () => {
  it('A. renders dropzone UI with accepted format, max size, and file picker fallback', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    expect(screen.getByTestId('eml-dropzone')).toBeInTheDocument();
    expect(screen.getByText(/Ingest RFC 5322 \.EML Sample/i)).toBeInTheDocument();
    expect(screen.getByText(/Format: \.eml only/i)).toBeInTheDocument();
    expect(screen.getByText(/Max Size: 50 MB/i)).toBeInTheDocument();
    expect(screen.getByTestId('file-picker-input')).toBeInTheDocument();
  });

  it('B. handles dragover and dragleave interactions', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const dropzone = screen.getByTestId('eml-dropzone');

    fireEvent.dragOver(dropzone);
    expect(dropzone).toHaveClass('dragover');

    fireEvent.dragLeave(dropzone);
    expect(dropzone).not.toHaveClass('dragover');
  });

  it('B. handles drop interaction with valid file', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const dropzone = screen.getByTestId('eml-dropzone');
    const validFile = new File(['From: test@example.com\nSubject: Test\n\nBody'], 'sample.eml', {
      type: 'message/rfc822',
    });

    fireEvent.drop(dropzone, {
      dataTransfer: {
        files: [validFile],
      },
    });

    expect(handleFile).toHaveBeenCalledTimes(1);
    expect(handleFile).toHaveBeenCalledWith(validFile);
  });

  it('C. handles valid file selection via file picker input', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const input = screen.getByTestId('file-picker-input');
    const validFile = new File(['From: sender@example.com'], 'investigation.eml', {
      type: 'message/rfc822',
    });

    fireEvent.change(input, {
      target: { files: [validFile] },
    });

    expect(handleFile).toHaveBeenCalledTimes(1);
    expect(handleFile).toHaveBeenCalledWith(validFile);
  });

  it('D. rejects unsupported file extensions (.msg, .pdf, .txt)', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const input = screen.getByTestId('file-picker-input');
    const invalidFile = new File(['content'], 'malicious.msg', {
      type: 'application/vnd.ms-outlook',
    });

    fireEvent.change(input, {
      target: { files: [invalidFile] },
    });

    expect(handleFile).not.toHaveBeenCalled();
    const errorAlert = screen.getByTestId('upload-validation-error');
    expect(errorAlert).toBeInTheDocument();
    expect(errorAlert).toHaveTextContent(/Unsupported file type.*malicious\.msg/i);
  });

  it('D. rejects empty files (0 bytes)', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const input = screen.getByTestId('file-picker-input');
    const emptyFile = new File([], 'empty.eml', {
      type: 'message/rfc822',
    });

    fireEvent.change(input, {
      target: { files: [emptyFile] },
    });

    expect(handleFile).not.toHaveBeenCalled();
    const errorAlert = screen.getByTestId('upload-validation-error');
    expect(errorAlert).toHaveTextContent(/empty file \(0 bytes\)/i);
  });

  it('D. rejects files exceeding 50 MB limit', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const input = screen.getByTestId('file-picker-input');
    // Mock a file larger than 50 MB
    const hugeFile = new File(['x'], 'huge.eml', { type: 'message/rfc822' });
    Object.defineProperty(hugeFile, 'size', { value: 52 * 1024 * 1024 });

    fireEvent.change(input, {
      target: { files: [hugeFile] },
    });

    expect(handleFile).not.toHaveBeenCalled();
    const errorAlert = screen.getByTestId('upload-validation-error');
    expect(errorAlert).toHaveTextContent(/exceeds the 50 MB forensic ingestion limit/i);
  });

  it('supports keyboard accessibility (Enter to open file picker)', () => {
    const handleFile = vi.fn();
    render(<EmlUploadZone onFileSelected={handleFile} />);

    const dropzone = screen.getByTestId('eml-dropzone');
    const input = screen.getByTestId('file-picker-input');
    const clickSpy = vi.spyOn(input, 'click');

    fireEvent.keyDown(dropzone, { key: 'Enter' });
    expect(clickSpy).toHaveBeenCalled();
  });
});
