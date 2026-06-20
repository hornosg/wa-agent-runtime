package agent

import (
	"context"

	"github.com/hornosg/wa-agent-runtime/src/contract"
	"github.com/riverqueue/river"
)

// InboundWorker — adaptador de entrada: River entrega el job y lo pasa al runtime.
type InboundWorker struct {
	river.WorkerDefaults[contract.InboundMessageArgs]
	rt *AgentRuntime
}

func NewInboundWorker(rt *AgentRuntime) *InboundWorker { return &InboundWorker{rt: rt} }

func (w *InboundWorker) Work(ctx context.Context, job *river.Job[contract.InboundMessageArgs]) error {
	a := job.Args
	return w.rt.Process(ctx, InboundMessage{
		TenantSlug:        a.TenantSlug,
		ProviderMessageID: a.ProviderMessageID,
		From:              a.From,
		To:                a.To,
		Text:              a.Text,
		ReceivedAt:        a.ReceivedAt,
	})
}
