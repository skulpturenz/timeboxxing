package monitor

type Reporter interface {
	Subscribe(id string) <-chan ForegroundProcess
}
