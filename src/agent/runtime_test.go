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

func (c *captureOutbound) Send(_ context.Context, to string, r Reply) error {
	c.to, c.reply = to, r
	return nil
}

func run(t *testing.T, mode TenantMode, text string) (Reply, string) {
	t.Helper()
	out := &captureOutbound{}
	rt := New(fakeTenants{mode: mode}, NewStubClassifier(), NewHardcodedReplier(), out, logging.New("test"))
	if err := rt.Process(context.Background(), InboundMessage{TenantSlug: "demo", From: "549110", Text: text}); err != nil {
		t.Fatalf("Process error: %v", err)
	}
	return out.reply, out.to
}

func TestProcess_BookingAgenda(t *testing.T) {
	r, to := run(t, ModeAgenda, "Hola! Quiero sacar un turno para mañana")
	if r.Handoff {
		t.Fatalf("agenda+booking no debería derivar a handoff: %+v", r)
	}
	if !strings.Contains(strings.ToLower(r.Text), "agend") {
		t.Fatalf("se esperaba respuesta de agenda, got: %q", r.Text)
	}
	if to != "549110" {
		t.Fatalf("outbound a destinatario incorrecto: %q", to)
	}
}

func TestProcess_BookingRagChatHandsOff(t *testing.T) {
	r, _ := run(t, ModeRagChat, "Quiero un turno para mañana")
	if !r.Handoff {
		t.Fatalf("rag_chat+booking debería derivar a handoff (P-13/G-10): %+v", r)
	}
}

func TestProcess_FAQ(t *testing.T) {
	r, _ := run(t, ModeRagChat, "Que precio tiene el corte?")
	if r.Handoff {
		t.Fatalf("una FAQ no debería derivar a handoff: %+v", r)
	}
}
