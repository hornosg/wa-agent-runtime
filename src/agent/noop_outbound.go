package agent

import (
	"context"

	"github.com/hornosg/wa-agent-runtime/src/contract"
	"github.com/riverqueue/river"
)

// NoopOutboundWorker registra el kind "outbound_message" en el cliente insert-only
// para que River permita Insert. NUNCA corre (ese cliente no trabaja colas).
type NoopOutboundWorker struct {
	river.WorkerDefaults[contract.OutboundMessageArgs]
}

func NewNoopOutboundWorker() *NoopOutboundWorker { return &NoopOutboundWorker{} }

func (*NoopOutboundWorker) Work(context.Context, *river.Job[contract.OutboundMessageArgs]) error {
	return nil
}
