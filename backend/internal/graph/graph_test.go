package graph

import (
	"testing"

	"sih26106/backend/internal/domain"
)

func TestBuildEvidenceBackedGraphIsDeterministic(t *testing.T) {
	email := &domain.Email{ID: "email-1", Filename: "dangerous.eml"}
	parsed := &domain.ParsedEmail{Message: domain.MessageMetadata{From: []string{"sender@example.test"}, To: []string{"recipient@example.test"}}, Indicators: domain.Indicators{Domains: []string{"example.test"}, URLs: []string{"https://example.test/login", "https://example.test/login"}, IPs: []string{"8.8.8.8"}}, Attachments: []domain.Attachment{{Filename: "invoice.exe", MIMEType: "application/octet-stream"}}}
	result := &domain.AnalysisResult{AnalysisID: "analysis-1", Status: "partial", ReceivedChain: []domain.ReceivedRelay{{SourceHeaderOrder: 3, Hostname: stringPtr("mx.example.test"), IPAddress: stringPtr("8.8.8.8"), Confidence: "medium", Provenance: Inferred}}, IPEnrichment: []domain.IPEnrichment{{IPAddress: "8.8.8.8", Status: "enriched", Organization: stringPtr("Example ISP"), Provenance: Enriched}}, Evidence: []domain.Evidence{
		{EvidenceID: "ev-email-ip", Type: "ip", Value: "8.8.8.8"}, {EvidenceID: "ev-url", Type: "url", Value: "https://example.test/login"}, {EvidenceID: "ev-domain", Type: "domain", Value: "example.test"}, {EvidenceID: "ev-sender", Type: "sender", Value: "sender@example.test"}, {EvidenceID: "ev-recipient", Type: "recipient", Value: "recipient@example.test"}, {EvidenceID: "ev-attachment", Type: "attachment", Value: "invoice.exe"}, {EvidenceID: "ev-relay", Type: "received", HeaderOrder: intPtr(3)}, {EvidenceID: "ev-enrich", Type: "ip_enrichment", Value: "8.8.8.8"},
	}, Risk: domain.RiskAssessment{ContributingSignals: []domain.RiskSignal{{Code: "URL_PRESENT", Description: "URL", Provenance: Observed, EvidenceIDs: []string{"ev-url"}}}}}
	first := Build("case-1", []EmailAnalysis{{Email: email, Parsed: parsed, Result: result}})
	second := Build("case-1", []EmailAnalysis{{Email: email, Parsed: parsed, Result: result}})
	if len(first.Nodes) == 0 || len(first.Edges) == 0 || len(first.Nodes) != len(second.Nodes) || len(first.Edges) != len(second.Edges) {
		t.Fatalf("graphs = %#v / %#v", first, second)
	}
	for index := range first.Nodes {
		if first.Nodes[index].ID != second.Nodes[index].ID {
			t.Fatalf("node IDs are not stable: %#v / %#v", first.Nodes, second.Nodes)
		}
	}
	if findNode(first, "sender:sender@example.test") == nil || findNode(first, "recipient:recipient@example.test") == nil || findNode(first, "url:https://example.test/login") == nil || findNode(first, "ip:8.8.8.8") == nil || findNode(first, "attachment:email-1:1") == nil {
		t.Fatalf("missing expected nodes: %#v", first.Nodes)
	}
	if findNode(first, "organization:Example ISP") == nil {
		t.Fatalf("missing enrichment node: %#v", first.Nodes)
	}
	if findNode(first, "risk:email-1:URL_PRESENT") == nil {
		t.Fatalf("missing risk node: %#v", first.Nodes)
	}
}

func TestBuildDoesNotInventAIEntitiesWithoutClassification(t *testing.T) {
	graph := Build("case-2", []EmailAnalysis{{Email: &domain.Email{ID: "email-2", Filename: "safe.eml"}, Parsed: &domain.ParsedEmail{}, Result: &domain.AnalysisResult{AnalysisID: "analysis-2", Status: "completed", AIAssessment: domain.AIAssessment{Status: "not_available"}}}})
	for _, node := range graph.Nodes {
		if node.Type == "ai_assessment" || node.Type == "domain" || node.Type == "url" || node.Type == "ip" {
			t.Fatalf("invented node: %#v", node)
		}
	}
}

func findNode(graph *domain.Graph, id string) *domain.GraphNode {
	for index := range graph.Nodes {
		if graph.Nodes[index].ID == id {
			return &graph.Nodes[index]
		}
	}
	return nil
}
func stringPtr(value string) *string { return &value }
func intPtr(value int) *int          { return &value }
