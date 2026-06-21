package agent

import (
	"context"
	"strings"
	"testing"
	"time"

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

// fakeConvStore — store en memoria.
type fakeConvStore struct{ m map[string]Conversation }

func newFakeConvStore() *fakeConvStore { return &fakeConvStore{m: map[string]Conversation{}} }
func (s *fakeConvStore) Get(_ context.Context, tenant, contact string) (Conversation, error) {
	if c, ok := s.m[tenant+"|"+contact]; ok {
		return c, nil
	}
	return Conversation{TenantSlug: tenant, Contact: contact, State: StateIdle}, nil
}
func (s *fakeConvStore) Save(_ context.Context, c Conversation) error {
	s.m[c.TenantSlug+"|"+c.Contact] = c
	return nil
}

// fakeScheduler — disponibilidad fija; registra si se reservó.
type fakeScheduler struct {
	avail   bool
	booked  bool
	bookErr error
}

func (f *fakeScheduler) NextAvailable(context.Context, string) (SlotInfo, bool, error) {
	if !f.avail {
		return SlotInfo{}, false, nil
	}
	return SlotInfo{ResourceID: "res-1", ResourceName: "Silla 1", Start: time.Date(2026, 6, 22, 13, 0, 0, 0, time.UTC), Minutes: 30}, true, nil
}
func (f *fakeScheduler) Book(context.Context, string, string, string, time.Time, int) (string, error) {
	if f.bookErr != nil {
		return "", f.bookErr
	}
	f.booked = true
	return "bk-1", nil
}

func newRuntime(mode TenantMode, chunks []FAQChunk, sched *fakeScheduler) (*AgentRuntime, *captureOutbound) {
	out := &captureOutbound{}
	rt := New(fakeTenants{mode: mode}, newFakeConvStore(),
		NewStubClassifier(), NewGuadaReplier(fakeRetriever{chunks: chunks}, fakeAnswerer{}),
		sched, out, logging.New("test"))
	return rt, out
}

func send(t *testing.T, rt *AgentRuntime, text string) {
	t.Helper()
	if err := rt.Process(context.Background(), InboundMessage{TenantSlug: "demo", From: "549110", Text: text}); err != nil {
		t.Fatalf("Process error: %v", err)
	}
}

func TestBooking_ProposeThenConfirm(t *testing.T) {
	sched := &fakeScheduler{avail: true}
	rt, out := newRuntime(ModeAgenda, nil, sched)

	send(t, rt, "Hola, quiero un turno")
	if !strings.Contains(strings.ToLower(out.reply.Text), "reservo") {
		t.Fatalf("se esperaba propuesta de turno, got: %q", out.reply.Text)
	}
	send(t, rt, "sí, dale")
	if sched.booked != true {
		t.Fatalf("la confirmación debía reservar (Book), no se llamó")
	}
	if !strings.Contains(strings.ToLower(out.reply.Text), "reservé") {
		t.Fatalf("se esperaba confirmación de reserva, got: %q", out.reply.Text)
	}
}

func TestBooking_ProposeThenDecline(t *testing.T) {
	sched := &fakeScheduler{avail: true}
	rt, out := newRuntime(ModeAgenda, nil, sched)
	send(t, rt, "quiero un turno")
	send(t, rt, "no, mejor no")
	if sched.booked {
		t.Fatalf("no debía reservar tras negativa")
	}
	if !strings.Contains(strings.ToLower(out.reply.Text), "qué día") && !strings.Contains(strings.ToLower(out.reply.Text), "que día") {
		t.Fatalf("se esperaba re-pregunta de horario, got: %q", out.reply.Text)
	}
}

func TestBooking_RagChatHandsOff(t *testing.T) {
	rt, out := newRuntime(ModeRagChat, nil, &fakeScheduler{avail: true})
	send(t, rt, "Quiero un turno para mañana")
	if !out.reply.Handoff {
		t.Fatalf("rag_chat+booking debería derivar a handoff (P-13/G-10): %+v", out.reply)
	}
}

func TestFAQ_WithEvidence(t *testing.T) {
	chunks := []FAQChunk{{Content: "El corte cuesta $5000.", Distance: 0.2}}
	rt, out := newRuntime(ModeRagChat, chunks, &fakeScheduler{})
	send(t, rt, "Que precio tiene el corte?")
	if out.reply.Handoff || !strings.Contains(out.reply.Text, "5000") {
		t.Fatalf("se esperaba respuesta grounded, got: %+v", out.reply)
	}
}

func TestFAQ_NoEvidenceHandsOff(t *testing.T) {
	rt, out := newRuntime(ModeRagChat, nil, &fakeScheduler{})
	send(t, rt, "Tienen estacionamiento?")
	if !out.reply.Handoff {
		t.Fatalf("sin evidencia debería derivar a handoff (P-05): %+v", out.reply)
	}
}
