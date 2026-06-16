package workers

import "fmt"

type QueueName string

const (
	TransitionEventQueueName         QueueName = "queue_transition_events"
	TransitionEventReportedQueueName QueueName = "queue_transition_event_reported"
	TransitionEventIndexedQueueName  QueueName = "queue_transition_event_indexed"
)

func ParseQueueName(value string) (QueueName, error) {
	switch name := QueueName(value); name {
	case TransitionEventQueueName, TransitionEventReportedQueueName, TransitionEventIndexedQueueName:
		return name, nil
	default:
		return "", fmt.Errorf("unknown queue name %q", value)
	}
}

func (q QueueName) String() string {
	return string(q)
}
