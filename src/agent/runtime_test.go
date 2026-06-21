package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/hornosg/wa-agent-runtime/src/logging"
)

type fakeTenants struct{ mode TenantMode }

func (f fakeTenants) Get(_ context.Context, slug string) (TenantConfig, error) {
	return TenantConfig{Slug: slug, Mode: f.mode, Status: "active"}, nil
}

type captureOutbound struct {
	to    string
	reply Reply
}

func (c *captureOutbound) Send(_ context.Context, _, to string, r Reply) error {
	c.to, c.reply = to, r
	return nil
}

// fakeRetriever / fakeAnswerer — dobles inline (no se importa knowledge: evita ciclo).
type fakeRetriever struct{ chunks []FAQChunk }

func (f fakeRetriever) Retrieve(_ context.Context, _, _ string) ([]FAQChunk, error) {
	return f.chunks, nil
}

type fakeAnswerer struct{}

func (fakeAnswerer) Answer(_ context.Context, _ string, chunks []FAQChunk) (Reply, error) {
	if len(chunks) == 0 {
		return Reply{Handoff: true}, nil
	}
	return Reply{Text: chunks[0].Content}, nil
}

func run(t *testing.T, mode TenantMode, text string, chunks []FAQChunk) (Reply, string) {
	t.Helper()
	out := &captureOutbound{}
	replier := NewGuadaReplier(fakeRetriever{chunks: chunks}, fakeAnswerer{})
	rt := New(fakeTenants{mode: mode}, NewStubClassifier(), replier, out, logging.New("test"))
	if err := rt.Process(context.Background(), InboundMessage{TenantSlug: "demo", From: "549110", Text: text}); err != nil {
		t.Fatalf("Process error: %v", err)
	}
	return out.reply, out.to
}

func TestProcess_BookingAgenda(t *testing.T) {
	r, to := run(t, ModeAgenda, "Hola! Quiero sacar un turno para mañana", nil)
	if r.Handoff || !strings.Contains(strings.ToLower(r.Text), "agend") {
		t.Fatalf("se esperaba respuesta de agenda sin handoff, got: %+v", r)
	}
	if to != "549110" {
		t.Fatalf("destinatario incorrecto: %q", to)
	}
}

func TestProcess_BookingRagChatHandsOff(t *testing.T) {
	r, _ := run(t, ModeRagChat, "Quiero un turno para mañana", nil)
	if !r.Handoff {
		t.Fatalf("rag_chat+booking debería derivar a handoff (P-13/G-10): %+v", r)
	}
}

func TestProcess_FAQ_WithEvidence(t *testing.T) {
	chunks := []FAQChunk{{Content: "El corte de pelo cuesta $5000.", Distance: 0.2}}
	r, _ := run(t, ModeRagChat, "Que precio tiene el corte?", chunks)
	if r.Handoff {
		t.Fatalf("con evidencia no debería derivar: %+v", r)
	}
	if !strings.Contains(r.Text, "5000") {
		t.Fatalf("se esperaba respuesta grounded en el chunk, got: %q", r.Text)
	}
}

func TestProcess_FAQ_NoEvidenceHandsOff(t *testing.T) {
	r, _ := run(t, ModeRagChat, "Tienen estacionamiento?", nil) // retriever sin chunks
	if !r.Handoff {
		t.Fatalf("sin evidencia en la KB debería derivar a handoff (P-05): %+v", r)
	}
}
