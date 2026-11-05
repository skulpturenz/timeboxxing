package hermes

import (
	"context"
	"sync"

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
	sources     map[string][]source
	sinks       map[string][]sink
	domains     map[string]chan Message
	once        sync.Once
	initialized bool
}

func (h *Hermes) AddSource(domain string, src <-chan Message) {
	if h.initialized {
		panic("already initialized")
	}

	srcs := []source{}
	if h.sources[domain] != nil {
		srcs = append(srcs, h.sources[domain]...)
	}

	srcs = append(srcs, source{sourceChan: src})
}

func (h *Hermes) AddSink(domain string, sink chan<- Message, filters ...func(Message) bool) {}

func (h *Hermes) Plumb(ctx context.Context, domain string) {
	h.once.Do(func() {
		var outChan chan Message
		if h.domains[domain] != nil {
			outChan = h.domains[domain]
		} else {
			outChan = make(chan Message)
		}
		h.domains[domain] = outChan

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

						filtered := false
						for _, filter := range sink.filters {
							if !filter(item) {
								filtered = true
								break
							}
						}
						if filtered {
							continue
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

		h.initialized = true
	})
}
