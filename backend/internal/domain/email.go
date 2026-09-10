package domain

import "time"

type Case struct {
	ID        string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Email struct {
	ID           string
	CaseID       string
	Filename     string
	RawContent   []byte
	Status       string
	ParseFailure string
	Parsed       *ParsedEmail
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Analysis struct {
	ID        string
	EmailID   string
	CaseID    string
	Status    string
	Failure   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AnalysisResult is the stable API representation of a completed or partial
// analysis. It separates the pipeline status from any AI assessment.
type AnalysisResult struct {
	AnalysisID     string                `json:"analysis_id"`
	EmailID        string                `json:"email_id"`
	CaseID         string                `json:"case_id"`
	Status         string                `json:"status"`
	AIAssessment   AIAssessment          `json:"ai_assessment"`
	Authentication AuthenticationResults `json:"authentication"`
	ReceivedChain  []ReceivedRelay       `json:"received_chain"`
	IPEnrichment   []IPEnrichment        `json:"ip_enrichment"`
	Evidence       []Evidence            `json:"evidence"`
	Risk           RiskAssessment        `json:"risk"`
	Failure        *Failure              `json:"failure"`
}

type AIAssessment struct {
	Status             string   `json:"status"`
	Classification     *string  `json:"classification"`
	Confidence         *float64 `json:"confidence"`
	SupportingSignals  []string `json:"supporting_signals"`
	EvidenceReferences []string `json:"evidence_references"`
	Provider           *string  `json:"provider"`
	Model              *string  `json:"model"`
	Failure            *Failure `json:"failure"`
}

type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AuthenticationResults struct {
	SPF   AuthenticationCheck `json:"spf"`
	DKIM  AuthenticationCheck `json:"dkim"`
	DMARC AuthenticationCheck `json:"dmarc"`
}

type AuthenticationCheck struct {
	Status             string              `json:"status"`
	EvidenceReferences []EvidenceReference `json:"evidence_references"`
	Explanation        string              `json:"explanation"`
}

type EvidenceReference struct {
	HeaderOrder int    `json:"header_order,omitempty"`
	HeaderName  string `json:"header_name,omitempty"`
	Source      string `json:"source,omitempty"`
	Value       string `json:"value,omitempty"`
}

type ReceivedRelay struct {
	Sequence          int     `json:"sequence"`
	Hostname          *string `json:"hostname"`
	IPAddress         *string `json:"ip_address"`
	Timestamp         *string `json:"timestamp"`
	SourceHeaderOrder int     `json:"source_header_order"`
	Confidence        string  `json:"confidence"`
	Provenance        string  `json:"provenance"`
}

type RiskAssessment struct {
	Score               int                 `json:"score"`
	Level               string              `json:"level"`
	Verdict             string              `json:"verdict"`
	Confidence          *float64            `json:"confidence"`
	ContributingSignals []RiskSignal        `json:"contributing_signals"`
	EvidenceReferences  []EvidenceReference `json:"evidence_references"`
}

type RiskSignal struct {
	Code               string              `json:"code"`
	Description        string              `json:"description"`
	Points             int                 `json:"points"`
	Category           string              `json:"category"`
	Provenance         string              `json:"provenance"`
	EvidenceReferences []EvidenceReference `json:"evidence_references"`
	EvidenceIDs        []string            `json:"evidence_ids,omitempty"`
}

type Evidence struct {
	EvidenceID                  string          `json:"evidence_id"`
	EmailID                     string          `json:"email_id"`
	AnalysisID                  string          `json:"analysis_id"`
	Source                      string          `json:"source"`
	Type                        string          `json:"type"`
	Value                       string          `json:"value,omitempty"`
	Snippet                     string          `json:"snippet,omitempty"`
	Provenance                  string          `json:"provenance"`
	ObservedAt                  *time.Time      `json:"observed_at,omitempty"`
	SourceLocation              string          `json:"source_location,omitempty"`
	HeaderOrder                 *int            `json:"header_order,omitempty"`
	RelatedSignalCodes          []string        `json:"related_signal_codes"`
	RelatedAIEvidenceReferences []string        `json:"related_ai_evidence_references"`
	Hash                        *string         `json:"hash"`
	SafeDisplay                 EvidenceDisplay `json:"safe_display"`
}

type EvidenceDisplay struct {
	Label    string `json:"label"`
	Redacted bool   `json:"redacted"`
}

type ParsedEmail struct {
	Message       MessageMetadata `json:"message"`
	MIME          MIMEMetadata    `json:"mime"`
	Headers       []Header        `json:"headers"`
	Indicators    Indicators      `json:"indicators"`
	Attachments   []Attachment    `json:"attachments"`
	PlainTextBody string          `json:"plain_text_body"`
}

type MessageMetadata struct {
	MessageID  *string  `json:"message_id,omitempty"`
	From       []string `json:"from"`
	To         []string `json:"to"`
	CC         []string `json:"cc"`
	ReplyTo    []string `json:"reply_to"`
	Subject    *string  `json:"subject,omitempty"`
	Date       *string  `json:"date,omitempty"`
	ReturnPath *string  `json:"return_path,omitempty"`
}

type MIMEMetadata struct {
	ContentType     string `json:"content_type"`
	HasPlainText    bool   `json:"has_plain_text"`
	HasHTML         bool   `json:"has_html"`
	AttachmentCount int    `json:"attachment_count"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Order int    `json:"order"`
}

type Indicators struct {
	IPs     []string `json:"ips"`
	Domains []string `json:"domains"`
	URLs    []string `json:"urls"`
}

// IPEnrichment is optional provider metadata about an observed IP. It never
// represents identity or maliciousness by itself.
type IPEnrichment struct {
	IPAddress    string     `json:"ip_address"`
	Status       string     `json:"status"`
	Country      *string    `json:"country"`
	Region       *string    `json:"region"`
	City         *string    `json:"city"`
	Latitude     *float64   `json:"latitude"`
	Longitude    *float64   `json:"longitude"`
	ASN          *string    `json:"asn"`
	Organization *string    `json:"organization"`
	Provider     *string    `json:"provider"`
	Confidence   *float64   `json:"confidence"`
	RetrievedAt  *time.Time `json:"retrieved_at"`
	Provenance   string     `json:"provenance"`
	Failure      *Failure   `json:"failure"`
}

type Attachment struct {
	Filename  string `json:"filename"`
	MIMEType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}
