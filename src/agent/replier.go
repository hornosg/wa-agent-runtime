package agent

import "context"

// GuadaReplier — respuestas de Guada por rama. La rama FAQ usa RAG (retrieval +
// answer grounded; P-05: sin evidencia → handoff). El resto es hardcodeado por
// ahora (H1), respetando el modo del tenant (G-10). Booking real = E06.
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
		if tc.Mode != ModeAgenda {
			return Reply{Text: "Por ahora no gestiono turnos por acá; te derivo con una persona del equipo.", Handoff: true}, nil
		}
		return Reply{Text: "¡Genial! Te ayudo a agendar. ¿Qué día y horario te queda cómodo?"}, nil

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
