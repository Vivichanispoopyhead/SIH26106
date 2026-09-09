import React, { useState, useRef, DragEvent, ChangeEvent, KeyboardEvent } from 'react';
import './EmlUploadZone.css';

interface EmlUploadZoneProps {
  onFileSelected: (file: File) => void;
  disabled?: boolean;
  isUploading?: boolean;
}

export const EmlUploadZone: React.FC<EmlUploadZoneProps> = ({
  onFileSelected,
  disabled = false,
  isUploading = false,
}) => {
  const [isDragOver, setIsDragOver] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validateAndHandleFile = (file: File) => {
    setValidationError(null);

    // Validation 1: Extension check (.eml)
    if (!file.name.toLowerCase().endsWith('.eml')) {
      setValidationError(
        `Unsupported file type: "${file.name}". Only RFC 5322 .eml files are accepted.`
      );
      return;
    }

    // Validation 2: Empty file
    if (file.size === 0) {
      setValidationError(
        `Invalid sample: "${file.name}" is an empty file (0 bytes).`
      );
      return;
    }

    // Validation 3: Maximum 50 MB
    const MAX_SIZE = 50 * 1024 * 1024;
    if (file.size > MAX_SIZE) {
      const sizeMB = (file.size / (1024 * 1024)).toFixed(2);
      setValidationError(
        `File too large: ${sizeMB} MB exceeds the 50 MB forensic ingestion limit.`
      );
      return;
    }

    onFileSelected(file);
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    if (!disabled && !isUploading) {
      setIsDragOver(true);
    }
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);

    if (disabled || isUploading) return;

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const file = e.dataTransfer.files[0];
      validateAndHandleFile(file);
    }
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const file = e.target.files[0];
      validateAndHandleFile(file);
    }
    // Reset file input value so the same file can be re-selected if desired
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleZoneClick = () => {
    if (!disabled && !isUploading && fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if ((e.key === 'Enter' || e.key === ' ') && !disabled && !isUploading) {
      e.preventDefault();
      fileInputRef.current?.click();
    }
  };

  return (
    <div className="upload-zone-wrapper">
      <div
        className={`upload-dropzone ${isDragOver ? 'dragover' : ''} ${
          disabled || isUploading ? 'disabled' : ''
        }`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={handleZoneClick}
        onKeyDown={handleKeyDown}
        role="button"
        tabIndex={disabled || isUploading ? -1 : 0}
        aria-label="Upload RFC 5322 .EML sample for forensic ingestion"
        data-testid="eml-dropzone"
      >
        <input
          ref={fileInputRef}
          type="file"
          accept=".eml"
          onChange={handleFileChange}
          style={{ display: 'none' }}
          data-testid="file-picker-input"
          disabled={disabled || isUploading}
        />

        <div className="dropzone-icon" aria-hidden="true">
          <svg
            width="36"
            height="36"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
            <polyline points="7 9 12 4 17 9" />
            <line x1="12" y1="4" x2="12" y2="16" />
          </svg>
        </div>

        <div className="dropzone-text">
          <h3 className="dropzone-title">
            {isUploading ? 'Ingesting Forensic Sample...' : 'Ingest RFC 5322 .EML Sample'}
          </h3>
          <p className="dropzone-sub">
            Drag and drop an .eml sample here, or{' '}
            <span className="file-picker-link">browse filesystem</span>
          </p>
        </div>

        <div className="dropzone-specs" data-testid="upload-spec-details">
          <span className="spec-pill">Format: .eml only</span>
          <span className="spec-pill">Max Size: 50 MB</span>
          <span className="spec-pill">Raw Artifact Preservation</span>
        </div>
      </div>

      {validationError && (
        <div
          className="upload-validation-error"
          role="alert"
          data-testid="upload-validation-error"
        >
          <span className="error-icon" aria-hidden="true">⚠</span>
          <span className="error-text">{validationError}</span>
          <button
            type="button"
            className="dismiss-error-btn"
            onClick={(e) => {
              e.stopPropagation();
              setValidationError(null);
            }}
            aria-label="Dismiss validation error"
          >
            ×
          </button>
        </div>
      )}
    </div>
  );
};
