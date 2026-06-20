package agent

import "context"

// HardcodedReplier — respuestas por rama (H1: "respuesta hardcodeada por rama").
// Respeta el modo del tenant (G-10): en rag_chat el booking deriva a handoff.
// El grounding real (RAG/FAQ E05, scheduling E06) reemplaza esto luego.
type HardcodedReplier struct{}

func NewHardcodedReplier() *HardcodedReplier { return &HardcodedReplier{} }

func (HardcodedReplier) Reply(_ context.Context, _ InboundMessage, intent Intent, tc TenantConfig) (Reply, error) {
	switch intent {
	case IntentFAQ:
		return Reply{Text: "¡Hola! Soy Guada. Dame un segundo que reviso eso y te confirmo. 🙂"}, nil

	case IntentBooking, IntentReschedule:
		if tc.Mode != ModeAgenda {
			// Tenant FAQ-only: no maneja turnos -> handoff.
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
