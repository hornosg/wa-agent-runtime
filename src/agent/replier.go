package agent

import (
	"context"
	"fmt"
)

// GuadaReplier — respuestas de Guada por rama. FAQ usa RAG (P-05). Booking propone
// el próximo turno real desde scheduling (E06) con CTA al cierre (P-15). Respeta el
// modo del tenant (G-10).
type GuadaReplier struct {
	retriever FAQRetriever
	answerer  FAQAnswerer
	scheduler BookingScheduler // puede ser nil (sin agenda configurada)
}

func NewGuadaReplier(retriever FAQRetriever, answerer FAQAnswerer, scheduler BookingScheduler) *GuadaReplier {
	return &GuadaReplier{retriever: retriever, answerer: answerer, scheduler: scheduler}
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
		// Proponer el próximo turno real (disponibilidad desde Postgres) con CTA al cierre (P-15).
		if r.scheduler != nil {
			if slot, ok, err := r.scheduler.NextAvailable(ctx, m.TenantSlug); err == nil && ok {
				return Reply{Text: fmt.Sprintf(
					"Tengo un turno el %s a las %s. ¿Te lo reservo? 🙂",
					slot.Start.Format("02/01"), slot.Start.Format("15:04"),
				)}, nil
			}
		}
		return Reply{Text: "¡Dale! Te ayudo a agendar. ¿Qué día y horario te queda cómodo?"}, nil

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
