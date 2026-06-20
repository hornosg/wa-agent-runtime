package knowledge

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/hornosg/wa-agent-runtime/src/agent"
)

// StubAnswerer — responde con el chunk más relevante (sin LLM). Para dev/CI.
type StubAnswerer struct{}

func NewStubAnswerer() *StubAnswerer { return &StubAnswerer{} }

func (StubAnswerer) Answer(_ context.Context, _ string, chunks []agent.FAQChunk) (agent.Reply, error) {
	if len(chunks) == 0 {
		return agent.Reply{Handoff: true}, nil
	}
	return agent.Reply{Text: chunks[0].Content}, nil
}

// AnthropicAnswerer — L2 FAQ con Sonnet 4.6 (ADR-0002, paso faq_answer), grounded
// SOLO en los chunks recuperados (P-05: si no alcanza, deriva).
type AnthropicAnswerer struct {
	client anthropic.Client
	model  anthropic.Model
}

func NewAnthropicAnswerer(apiKey string) *AnthropicAnswerer {
	return &AnthropicAnswerer{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  anthropic.Model("claude-sonnet-4-6"), // ADR-0002 paso faq_answer
	}
}

const answerSystem = `Sos Guada, la asistente de WhatsApp de un negocio. Respondé la consulta del cliente
USANDO ÚNICAMENTE la información de CONTEXTO. Si el contexto no alcanza para responder, respondé
exactamente "HANDOFF". Tono cálido, rioplatense, breve. No inventes datos.`

func (a *AnthropicAnswerer) Answer(ctx context.Context, query string, chunks []agent.FAQChunk) (agent.Reply, error) {
	var ctxText strings.Builder
	for _, c := range chunks {
		ctxText.WriteString("- ")
		ctxText.WriteString(c.Content)
		ctxText.WriteByte('\n')
	}
	prompt := "CONTEXTO:\n" + ctxText.String() + "\nConsulta del cliente: " + query

	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: 500,
		System:    []anthropic.TextBlockParam{{Text: answerSystem}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return agent.Reply{}, err
	}
	var out string
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			out += t.Text
		}
	}
	out = strings.TrimSpace(out)
	if out == "" || strings.EqualFold(out, "HANDOFF") {
		return agent.Reply{Text: "Mejor te paso con una persona del equipo para eso. 🙌", Handoff: true}, nil
	}
	return agent.Reply{Text: out}, nil
}
