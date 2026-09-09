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
	AnalysisID   string       `json:"analysis_id"`
	EmailID      string       `json:"email_id"`
	CaseID       string       `json:"case_id"`
	Status       string       `json:"status"`
	AIAssessment AIAssessment `json:"ai_assessment"`
	Failure      *Failure     `json:"failure"`
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

type Attachment struct {
	Filename  string `json:"filename"`
	MIMEType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}
