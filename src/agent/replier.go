package agent

import "context"

// GuadaReplier — respuestas de Guada para las ramas SIN estado (FAQ con RAG, cancel,
// handoff, y booking en tenant rag_chat → handoff). El flujo de booking con agenda
// (propone → confirma → reserva) lo maneja AgentRuntime (E07), no el replier.
type GuadaReplier struct {
	retriever FAQRetriever
	answerer  FAQAnswerer
}

func NewGuadaReplier(retriever FAQRetriever, answerer FAQAnswerer) *GuadaReplier {
	return &GuadaReplier{retriever: retriever, answerer: answerer}
}

func (r *GuadaReplier) Reply(ctx context.Context, m InboundMessage, intent Intent, tc TenantConfig) (Reply, error) {
	switch intent {
	case IntentFAQ:
		chunks, err := r.retriever.Retrieve(ctx, m.TenantSlug, m.Text)
		if err != nil {
			return Reply{}, err
		}
		if len(chunks) == 0 {
			// Sin evidencia en la KnowledgeBase → handoff, no inventar (P-05/G-09).
			return Reply{Text: "Mmm, eso no lo tengo a mano. Te paso con una persona del equipo para que te ayude. 🙌", Handoff: true}, nil
		}
		return r.answerer.Answer(ctx, m.Text, chunks)

	case IntentBooking, IntentReschedule:
		// Sólo llega acá en tenants sin agenda (rag_chat) → handoff. El flujo con
		// agenda lo intercepta AgentRuntime antes del replier (E07).
		return Reply{Text: "Por ahora no gestiono turnos por acá; te derivo con una persona del equipo.", Handoff: true}, nil

	case IntentCancel:
		if tc.Mode != ModeAgenda {
			return Reply{Text: "Te derivo con una persona del equipo para eso.", Handoff: true}, nil
		}
		return Reply{Text: "Listo, veamos tu turno para cancelarlo. ¿Me pasás tu nombre o el día del turno?"}, nil

	case IntentHandoff:
		return Reply{Text: "Te paso con una persona del equipo en un ratito. 🙌", Handoff: true}, nil

	default:
		return Reply{Text: "¡Hola! Soy Guada 🙂 Contame en qué te puedo ayudar (consultas o turnos)."}, nil
	}
}
