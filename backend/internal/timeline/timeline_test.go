package timeline

import (
	"testing"
	"time"

	"sih26106/backend/internal/domain"
)

func TestBuildOrdersRelaysAndLinksEvidence(t *testing.T) {
	uploaded := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	first := "2026-09-10T10:00:00Z"
	second := "2026-09-10T11:00:00Z"
	firstOrder, secondOrder := 4, 5
	result := &domain.AnalysisResult{
		AnalysisID: "analysis-1", EmailID: "email-1", CaseID: "case-1", Status: "completed",
		ReceivedChain: []domain.ReceivedRelay{
			{Sequence: 2, Hostname: stringPointer("edge.example"), IPAddress: stringPointer("203.0.113.2"), Timestamp: &second, SourceHeaderOrder: secondOrder, Confidence: "medium", Provenance: "inferred"},
			{Sequence: 1, Hostname: stringPointer("origin.example"), IPAddress: stringPointer("203.0.113.1"), Timestamp: &first, SourceHeaderOrder: firstOrder, Confidence: "medium", Provenance: "inferred"},
		},
		Evidence: []domain.Evidence{
			{EvidenceID: "evidence-received-4", Type: "received", HeaderOrder: &firstOrder},
			{EvidenceID: "evidence-received-5", Type: "received", HeaderOrder: &secondOrder},
		},
	}
	analysisAt := uploaded.Add(time.Hour)
	graph := Build("case-1", []EmailAnalysis{{
		Email:    &domain.Email{ID: "email-1", CaseID: "case-1", Filename: "message.eml", CreatedAt: uploaded},
		Parsed:   &domain.ParsedEmail{Headers: []domain.Header{{Name: "Received", Order: firstOrder}, {Name: "Received", Order: secondOrder}}},
		Result:   result,
		Analysis: &domain.Analysis{ID: "analysis-1", EmailID: "email-1", CaseID: "case-1", Status: "completed", UpdatedAt: analysisAt},
	}})
	if len(graph.Events) != 4 {
		t.Fatalf("events = %#v", graph.Events)
	}
	if graph.Events[0].ID != "timeline:email-1:relay:1" || graph.Events[1].ID != "timeline:email-1:relay:2" {
		t.Fatalf("relay ordering = %#v", graph.Events)
	}
	if graph.Events[0].Provenance != Inferred || graph.Events[0].RelatedNodeID == nil || *graph.Events[0].RelatedNodeID != "relay:email-1:4" {
		t.Fatalf("relay provenance/node = %#v", graph.Events[0])
	}
	if len(graph.Events[0].EvidenceIDs) != 1 || graph.Events[0].EvidenceIDs[0] != "evidence-received-4" {
		t.Fatalf("relay evidence = %#v", graph.Events[0].EvidenceIDs)
	}
	if graph.Events[2].Type != "email_received" || graph.Events[2].Metadata["timestamp_source"] != "upload" {
		t.Fatalf("upload event = %#v", graph.Events[2])
	}
	if graph.Events[3].Type != "analysis_completed" || graph.Events[3].Timestamp == nil || !graph.Events[3].Timestamp.Equal(analysisAt) {
		t.Fatalf("analysis event = %#v", graph.Events[3])
	}
}

func TestBuildDoesNotInventMissingTimestampsOrIndicators(t *testing.T) {
	graph := Build("case-1", []EmailAnalysis{{
		Email:  &domain.Email{ID: "email-1", CaseID: "case-1"},
		Parsed: &domain.ParsedEmail{Indicators: domain.Indicators{IPs: []string{"192.0.2.1", "192.0.2.1"}}},
		Result: &domain.AnalysisResult{AnalysisID: "analysis-1", EmailID: "email-1", CaseID: "case-1", Status: "completed", ReceivedChain: []domain.ReceivedRelay{{Sequence: 1, SourceHeaderOrder: 2, Provenance: "inferred"}}},
	}})
	if len(graph.Events) != 3 {
		t.Fatalf("events = %#v", graph.Events)
	}
	for _, event := range graph.Events {
		if event.Timestamp != nil {
			t.Fatalf("invented timestamp in %#v", event)
		}
	}
	if graph.Events[1].Type != "indicator_observed" || graph.Events[1].ID != "timeline:email-1:indicator:ip:192.0.2.1" {
		t.Fatalf("indicator event = %#v", graph.Events[1])
	}
}
