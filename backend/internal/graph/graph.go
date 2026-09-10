package graph

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"sih26106/backend/internal/domain"
)

type EmailAnalysis struct {
	Email  *domain.Email
	Parsed *domain.ParsedEmail
	Result *domain.AnalysisResult
}

const (
	Observed   = "OBSERVED"
	Enriched   = "ENRICHED"
	Inferred   = "INFERRED"
	AIAssessed = "AI-ASSESSED"
)

type builder struct {
	graph        domain.Graph
	nodes        map[string]int
	edges        map[string]int
	evidenceNode map[string][]string
	evidence     map[string]domain.Evidence
}

func Build(caseID string, inputs []EmailAnalysis) *domain.Graph {
	b := &builder{graph: domain.Graph{CaseID: caseID, EmailIDs: []string{}, AnalysisIDs: []string{}, Nodes: []domain.GraphNode{}, Edges: []domain.GraphEdge{}}, nodes: map[string]int{}, edges: map[string]int{}, evidenceNode: map[string][]string{}, evidence: map[string]domain.Evidence{}}
	for _, input := range inputs {
		if input.Email == nil || input.Parsed == nil || input.Result == nil {
			continue
		}
		b.addEmail(input)
	}
	sort.Slice(b.graph.Nodes, func(i, j int) bool { return b.graph.Nodes[i].ID < b.graph.Nodes[j].ID })
	sort.Slice(b.graph.Edges, func(i, j int) bool { return b.graph.Edges[i].ID < b.graph.Edges[j].ID })
	sort.Strings(b.graph.EmailIDs)
	sort.Strings(b.graph.AnalysisIDs)
	return &b.graph
}

func (b *builder) addEmail(input EmailAnalysis) {
	email, parsed, result := input.Email, input.Parsed, input.Result
	b.graph.EmailIDs = appendUnique(b.graph.EmailIDs, email.ID)
	b.graph.AnalysisIDs = appendUnique(b.graph.AnalysisIDs, result.AnalysisID)
	for _, item := range result.Evidence {
		b.evidence[item.EvidenceID] = item
	}
	emailID := "email:" + email.ID
	b.node(emailID, "email", email.Filename, email.ID, Observed, nil, nil)
	for _, sender := range parsed.Message.From {
		value := strings.TrimSpace(sender)
		id := "sender:" + value
		ids := b.evidenceFor("sender", value)
		b.node(id, "sender", value, value, Observed, ids, nil)
		b.edge(emailID, id, "from", Observed, ids, nil)
	}
	for _, values := range [][]string{parsed.Message.To, parsed.Message.CC, parsed.Message.ReplyTo} {
		for _, recipient := range values {
			value := strings.TrimSpace(recipient)
			id := "recipient:" + value
			ids := b.evidenceFor("recipient", value)
			b.node(id, "recipient", value, value, Observed, ids, nil)
			b.edge(emailID, id, "to", Observed, ids, nil)
		}
	}
	domainIDs := map[string]string{}
	for _, value := range parsed.Indicators.Domains {
		domainValue := strings.TrimSpace(value)
		id := "domain:" + domainValue
		domainIDs[strings.ToLower(domainValue)] = id
		ids := b.evidenceFor("domain", domainValue)
		b.node(id, "domain", domainValue, domainValue, Observed, ids, nil)
		b.edge(emailID, id, "contains", Observed, ids, nil)
	}
	for _, value := range parsed.Indicators.URLs {
		urlValue := graphURL(value)
		id := "url:" + urlValue
		ids := b.evidenceFor("url", urlValue)
		b.node(id, "url", urlValue, urlValue, Observed, ids, nil)
		b.edge(emailID, id, "contains", Observed, ids, nil)
		if parsedURL, err := url.Parse(value); err == nil {
			if domainID, ok := domainIDs[strings.ToLower(parsedURL.Hostname())]; ok {
				b.edge(id, domainID, "has_domain", Inferred, ids, nil)
			}
		}
	}
	for _, value := range parsed.Indicators.IPs {
		b.addIP(emailID, value, b.evidenceFor("ip", value), Observed)
	}
	for _, relay := range result.ReceivedChain {
		relayID := "relay:" + email.ID + ":" + strconv.Itoa(relay.SourceHeaderOrder)
		label := "relay"
		if relay.Hostname != nil {
			label = *relay.Hostname
		} else if relay.IPAddress != nil {
			label = *relay.IPAddress
		}
		ids := b.evidenceForHeader(relay.SourceHeaderOrder)
		metadata := map[string]string{"sequence": strconv.Itoa(relay.Sequence), "confidence": relay.Confidence}
		b.node(relayID, "relay", label, label, Inferred, ids, metadata)
		b.edge(emailID, relayID, "received_via", relay.Provenance, ids, confidencePointer(relay.Confidence))
		if relay.IPAddress != nil {
			b.addIP(emailID, *relay.IPAddress, appendUnique(ids, b.evidenceFor("ip", *relay.IPAddress)...), Observed)
			b.edge(relayID, "ip:"+*relay.IPAddress, "observed_ip", Observed, ids, nil)
		}
	}
	for index, attachment := range parsed.Attachments {
		id := "attachment:" + email.ID + ":" + strconv.Itoa(index+1)
		ids := b.evidenceFor("attachment", attachment.Filename)
		metadata := map[string]string{"mime_type": attachment.MIMEType, "size_bytes": strconv.FormatInt(attachment.SizeBytes, 10)}
		b.node(id, "attachment", attachment.Filename, attachment.Filename, Observed, ids, metadata)
		b.edge(emailID, id, "contains", Observed, ids, nil)
	}
	b.addEnrichment(emailID, result)
	b.addAI(emailID, result)
	b.addRiskSignals(emailID, result)
}

func (b *builder) addIP(emailID, value string, ids []string, provenance string) {
	id := "ip:" + value
	b.node(id, "ip", value, value, provenance, ids, nil)
	b.edge(emailID, id, "contains", provenance, ids, nil)
}

func (b *builder) addEnrichment(emailID string, result *domain.AnalysisResult) {
	for _, value := range result.IPEnrichment {
		ids := b.evidenceFor("ip_enrichment", value.IPAddress)
		if len(ids) == 0 {
			ids = b.evidenceFor("ip", value.IPAddress)
		}
		ipID := "ip:" + value.IPAddress
		if _, ok := b.nodes[ipID]; !ok {
			b.addIP(emailID, value.IPAddress, ids, Observed)
		}
		if value.Status != "enriched" {
			continue
		}
		if value.Organization != nil && *value.Organization != "" {
			orgID := "organization:" + *value.Organization
			b.node(orgID, "organization", *value.Organization, *value.Organization, Enriched, ids, nil)
			b.edge(ipID, orgID, "associated_with", Enriched, ids, value.Confidence)
		}
		if value.Country != nil || value.Region != nil || value.City != nil || value.Latitude != nil || value.Longitude != nil {
			geoID := "geolocation:" + value.IPAddress
			metadata := map[string]string{}
			if value.Country != nil {
				metadata["country"] = *value.Country
			}
			if value.Region != nil {
				metadata["region"] = *value.Region
			}
			if value.City != nil {
				metadata["city"] = *value.City
			}
			b.node(geoID, "geolocation", "Estimated location for "+value.IPAddress, value.IPAddress, Enriched, ids, metadata)
			b.edge(ipID, geoID, "located_in_estimate", Enriched, ids, value.Confidence)
		}
	}
}

func (b *builder) addAI(emailID string, result *domain.AnalysisResult) {
	if result.AIAssessment.Classification == nil || result.AIAssessment.Status == "failed" || result.AIAssessment.Status == "not_available" {
		return
	}
	ids := b.evidenceForType("ai_assessment")
	value := *result.AIAssessment.Classification
	id := "ai:" + strings.TrimPrefix(emailID, "email:") + ":" + value
	b.node(id, "ai_assessment", "AI: "+value, value, AIAssessed, ids, map[string]string{"status": result.AIAssessment.Status})
	b.edge("email:"+emailID, id, "assessed_by", AIAssessed, ids, result.AIAssessment.Confidence)
}

func (b *builder) addRiskSignals(emailID string, result *domain.AnalysisResult) {
	for _, signal := range result.Risk.ContributingSignals {
		if len(signal.EvidenceIDs) == 0 {
			continue
		}
		id := "risk:" + strings.TrimPrefix(emailID, "email:") + ":" + signal.Code
		b.node(id, "risk_signal", signal.Code, signal.Description, Inferred, signal.EvidenceIDs, map[string]string{"category": signal.Category, "points": strconv.Itoa(signal.Points)})
		for _, evidenceID := range signal.EvidenceIDs {
			for _, target := range b.evidenceNode[evidenceID] {
				b.edge(id, target, "supported_by", signal.Provenance, []string{evidenceID}, nil)
			}
		}
	}
}

func (b *builder) node(id, nodeType, label, value, provenance string, evidenceIDs []string, metadata map[string]string) {
	evidenceIDs = uniqueSorted(evidenceIDs)
	if index, ok := b.nodes[id]; ok {
		b.graph.Nodes[index].EvidenceIDs = uniqueSorted(append(b.graph.Nodes[index].EvidenceIDs, evidenceIDs...))
		for _, evidenceID := range evidenceIDs {
			b.evidenceNode[evidenceID] = appendUnique(b.evidenceNode[evidenceID], id)
		}
		return
	}
	if metadata == nil {
		metadata = map[string]string{}
	}
	b.nodes[id] = len(b.graph.Nodes)
	b.graph.Nodes = append(b.graph.Nodes, domain.GraphNode{ID: id, Type: nodeType, Label: label, Value: value, Provenance: provenance, EvidenceIDs: evidenceIDs, Metadata: metadata})
	for _, evidenceID := range evidenceIDs {
		b.evidenceNode[evidenceID] = appendUnique(b.evidenceNode[evidenceID], id)
	}
}

func (b *builder) edge(source, target, relationship, provenance string, evidenceIDs []string, confidence *float64) {
	if source == "" || target == "" {
		return
	}
	id := "edge:" + source + ":" + relationship + ":" + target
	evidenceIDs = uniqueSorted(evidenceIDs)
	if index, ok := b.edges[id]; ok {
		b.graph.Edges[index].EvidenceIDs = uniqueSorted(append(b.graph.Edges[index].EvidenceIDs, evidenceIDs...))
		return
	}
	b.edges[id] = len(b.graph.Edges)
	b.graph.Edges = append(b.graph.Edges, domain.GraphEdge{ID: id, SourceNodeID: source, TargetNodeID: target, Relationship: relationship, Provenance: provenance, EvidenceIDs: evidenceIDs, Confidence: confidence})
}

func (b *builder) evidenceFor(kind, value string) []string {
	ids := []string{}
	for _, item := range b.evidence {
		if item.Type == kind && (item.Value == value || (kind == "url" && graphURL(item.Value) == graphURL(value))) {
			ids = append(ids, item.EvidenceID)
		}
	}
	return uniqueSorted(ids)
}
func (b *builder) evidenceForHeader(order int) []string {
	ids := []string{}
	for _, item := range b.evidence {
		if item.HeaderOrder != nil && *item.HeaderOrder == order {
			ids = append(ids, item.EvidenceID)
		}
	}
	return uniqueSorted(ids)
}
func (b *builder) evidenceForType(kind string) []string {
	ids := []string{}
	for _, item := range b.evidence {
		if item.Type == kind {
			ids = append(ids, item.EvidenceID)
		}
	}
	return uniqueSorted(ids)
}
func graphURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	parsed.User = nil
	return parsed.String()
}
func confidencePointer(value string) *float64 {
	if value == "high" {
		v := .9
		return &v
	}
	if value == "medium" {
		v := .7
		return &v
	}
	if value == "low" {
		v := .3
		return &v
	}
	return nil
}
func appendUnique(values []string, additions ...string) []string {
	for _, value := range additions {
		found := false
		for _, existing := range values {
			if existing == value {
				found = true
				break
			}
		}
		if !found && value != "" {
			values = append(values, value)
		}
	}
	return values
}
func uniqueSorted(values []string) []string {
	sort.Strings(values)
	return appendUnique(nil, values...)
}
