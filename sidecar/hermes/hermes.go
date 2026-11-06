package hermes

import (
	"context"
	"sync"

	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type Message = map[any]any

type MiddlewareHandlerFunc = func(sink string, domain string, msg Message, filtered bool)
type Middleware = func(next func()) MiddlewareHandlerFunc

type sink struct {
	key     string
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
	middleware  []Middleware
}

func (h *Hermes) Use(middleware Middleware) *Hermes {
	if h.initialized {
		panic("already initialized")
	}

	h.middleware = append(h.middleware, middleware)

	return h
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
	h.sources[domain] = srcs
}

func (h *Hermes) AddSink(domain string, key string, outChan chan<- Message, filters ...func(Message) bool) {
	if h.initialized {
		panic("already initialized")
	}

	sinks := []sink{}
	if h.sinks[domain] != nil {
		sinks = append(sinks, h.sinks[domain]...)
	}

	sinks = append(sinks, sink{key: key, outChan: outChan, filters: filters})
	h.sinks[domain] = sinks
}

func (h *Hermes) Plumb(ctx context.Context, domain string) {
	h.once.Do(func() {
		createMiddlewareStack := func() MiddlewareHandlerFunc {
			stack := func(sink string, domain string, msg Message, filtered bool) {
				noop := func(sink string, domain string, msg Message, filtered bool) {}

				if len(h.middleware) == 0 {
					return
				}

				combined := noop

				for _, m := range h.middleware {
					ptr := combined
					combined = m(func() { ptr(sink, domain, msg, filtered) })
				}

				combined(sink, domain, msg, filtered)
			}

			return stack
		}

		middleware := createMiddlewareStack()

		srcs := []source{}
		if h.sources[domain] != nil {
			srcs = append(srcs, h.sources[domain]...)
		}

		inChans := []<-chan Message{}
		for _, src := range srcs {
			inChans = append(inChans, src.sourceChan)
		}

		var outChan chan Message
		if h.domains[domain] != nil {
			outChan = h.domains[domain]
		} else {
			// size: sources are fanned in, so want to be able to accept at least 1 item from each source
			outChan = make(chan Message, len(inChans))
		}
		h.domains[domain] = outChan

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
						middleware(sink.key, domain, item, filtered)
						if filtered {
							continue
						}

						sink.outChan <- item
					}
				}
			}()

			outChans = append(outChans, inChan)
		}

		utils.FanInChan(ctx, outChan, inChans...)
		utils.FanOutChanWithBackpressure(ctx, outChan, outChans...)

		h.initialized = true
	})
}
