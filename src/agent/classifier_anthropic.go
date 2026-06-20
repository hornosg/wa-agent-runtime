package agent

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClassifier — L1 con Haiku 4.5 (ADR-0002, paso intent_classification).
// Salida acotada (max_tokens chico), sin thinking. Tiering: ver llm-routing.json.
type AnthropicClassifier struct {
	client anthropic.Client
	model  anthropic.Model
}

func NewAnthropicClassifier(apiKey string) *AnthropicClassifier {
	return &AnthropicClassifier{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  anthropic.ModelClaudeHaiku4_5_20251001,
	}
}

const classifyPrompt = `Clasificá el siguiente mensaje de WhatsApp en UNA sola palabra entre:
faq, booking, cancel, reschedule, handoff, other.
Respondé únicamente la palabra, sin explicación.

Mensaje: `

func (a *AnthropicClassifier) Classify(ctx context.Context, m InboundMessage, _ TenantConfig) (Intent, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: 16,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(classifyPrompt + m.Text)),
		},
	})
	if err != nil {
		return IntentOther, err
	}
	var out string
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			out += t.Text
		}
	}
	return normalizeIntent(out), nil
}

func normalizeIntent(s string) Intent {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "faq":
		return IntentFAQ
	case "booking":
		return IntentBooking
	case "cancel":
		return IntentCancel
	case "reschedule":
		return IntentReschedule
	case "handoff":
		return IntentHandoff
	default:
		return IntentOther
	}
}
