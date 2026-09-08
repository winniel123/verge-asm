package queue

import (
	"time"

	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

func withdrawalMessages(observedAt time.Time, departures []departure) []*message.Message {
	var msgs []*message.Message
	for _, d := range departures {
		// A cascade closure is silent and a descoping fires at the scope, so only this ground writes (ADR-0087).
		if d.Reason != string(drift.ReasonMeasuredAbsent) {
			continue
		}
		if m := message.Withdrawal(d.SubjectKind, d.SubjectKey, d.Timelines, observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}
