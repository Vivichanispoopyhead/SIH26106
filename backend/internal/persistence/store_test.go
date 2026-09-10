package persistence

import (
	"context"
	"testing"
	"time"

	"sih26106/backend/internal/domain"
)

func TestMemoryStoreAnalysisResultPreservesIPEnrichment(t *testing.T) {
	store := NewMemoryStore()
	email, err := store.CreateUpload(context.Background(), "message.eml", []byte("From: a@example.test\r\n\r\nbody"))
	if err != nil {
		t.Fatal(err)
	}
	analysis, err := store.CreateAnalysis(context.Background(), email.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	result := &domain.AnalysisResult{
		AnalysisID: analysis.ID, EmailID: email.ID, CaseID: email.CaseID, Status: "partial",
		IPEnrichment: []domain.IPEnrichment{{IPAddress: "8.8.8.8", Status: "enriched", Provenance: "ENRICHED", RetrievedAt: &now}},
		Evidence:     []domain.Evidence{{EvidenceID: "evidence-1", EmailID: email.ID, AnalysisID: analysis.ID, Type: "ip", Source: "indicator", Value: "8.8.8.8", Provenance: "OBSERVED", RelatedSignalCodes: []string{"IP_PRESENT"}}},
	}
	if err := store.SaveAnalysisResult(context.Background(), analysis.ID, result); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetAnalysisResult(context.Background(), email.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.IPEnrichment) != 1 || loaded.IPEnrichment[0].IPAddress != "8.8.8.8" || loaded.IPEnrichment[0].RetrievedAt == nil || !loaded.IPEnrichment[0].RetrievedAt.Equal(now) {
		t.Fatalf("loaded = %#v", loaded.IPEnrichment)
	}
	if len(loaded.Evidence) != 1 || loaded.Evidence[0].EvidenceID != "evidence-1" || loaded.Evidence[0].RelatedSignalCodes[0] != "IP_PRESENT" {
		t.Fatalf("loaded evidence = %#v", loaded.Evidence)
	}
}
