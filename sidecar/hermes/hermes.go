package hermes

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type Message = map[any]any

type sink struct {
	outChan chan<- Message
	filters []func(Message) bool
}

type source struct {
	sourceChan <-chan Message
}

type Hermes struct {
	sources    map[string][]source
	sinks      map[string][]sink
	domains    map[string]chan Message
	cancelFunc context.CancelFunc
}

func (h *Hermes) AddSource(domain string, src <-chan Message) {
	srcs := []source{}
	if h.sources[domain] != nil {
		srcs = append(srcs, h.sources[domain]...)
	}

	srcs = append(srcs, source{sourceChan: src})

	if h.cancelFunc != nil {
		h.cancelFunc()
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	h.plumb(ctx, domain)

	h.cancelFunc = cancelFunc
}

func (h *Hermes) AddSink(domain string, sink chan<- Message, filters ...func(Message) bool) {}

func (h *Hermes) plumb(ctx context.Context, domain string) {
	var outChan chan Message
	if h.domains[domain] != nil {
		outChan = h.domains[domain]
	} else {
		outChan = make(chan Message)
	}

	srcs := []source{}
	if h.sources[domain] != nil {
		srcs = append(srcs, h.sources[domain]...)
	}

	inChans := []<-chan Message{}
	for _, src := range srcs {
		inChans = append(inChans, src.sourceChan)
	}

	sinks := []sink{}
	if h.sinks[domain] != nil {
		sinks = append(sinks, h.sinks[domain]...)
	}

	outChans := []chan<- Message{}
	for _, sink := range sinks {
		inChan := make(chan Message)

		go func() {
			defer close(inChan)

			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inChan:
					if !ok {
						return
					}

					for _, filter := range sink.filters {
						if !filter(item) {
							continue
						}
					}

					select {
					case sink.outChan <- item:
					default:
					}
				}
			}
		}()

		outChans = append(outChans, inChan)
	}

	utils.FanInChan(ctx, outChan, inChans...)
	utils.FanOutChan(ctx, outChan, outChans...)
}
