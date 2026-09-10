package forensics

import (
	"testing"

	"sih26106/backend/internal/domain"
)

func TestAnalyzeHeadersAuthenticationAndReceivedChain(t *testing.T) {
	auth, relays := AnalyzeHeaders([]domain.Header{
		{Name: "Authentication-Results", Value: "mx; spf=pass smtp.mailfrom=example.org; dkim=fail header.d=example.org; dmarc=pass", Order: 3},
		{Name: "Received", Value: "from edge.example.org (203.0.113.7) by mx.example.org; Tue, 09 Sep 2026 10:00:00 +0000", Order: 4},
		{Name: "Received", Value: "malformed relay data", Order: 5},
	})
	if auth.SPF.Status != "pass" || auth.DKIM.Status != "fail" || auth.DMARC.Status != "pass" {
		t.Fatalf("authentication = %#v", auth)
	}
	if len(auth.SPF.EvidenceReferences) != 1 || auth.SPF.EvidenceReferences[0].HeaderOrder != 3 {
		t.Fatalf("SPF evidence = %#v", auth.SPF.EvidenceReferences)
	}
	if len(relays) != 2 || relays[0].Sequence != 2 || relays[0].SourceHeaderOrder != 4 || relays[1].Sequence != 1 {
		t.Fatalf("received chain = %#v", relays)
	}
	if relays[0].Hostname == nil || *relays[0].Hostname != "edge.example.org" || relays[0].IPAddress == nil || *relays[0].IPAddress != "203.0.113.7" || relays[0].Timestamp == nil {
		t.Fatalf("parsed relay = %#v", relays[0])
	}
	if relays[1].Hostname != nil || relays[1].IPAddress != nil || relays[1].Timestamp != nil || relays[1].Confidence != "low" {
		t.Fatalf("malformed relay = %#v", relays[1])
	}
}

func TestAnalyzeHeadersMissingAuthentication(t *testing.T) {
	auth, relays := AnalyzeHeaders([]domain.Header{{Name: "Subject", Value: "notice", Order: 1}})
	if auth.SPF.Status != "none" || auth.DKIM.Status != "none" || auth.DMARC.Status != "none" {
		t.Fatalf("missing authentication = %#v", auth)
	}
	if len(auth.SPF.EvidenceReferences) != 0 || len(relays) != 0 {
		t.Fatalf("unexpected evidence or relays: %#v %#v", auth, relays)
	}
}
