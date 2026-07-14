package sessionnew

type Reporter interface {
	Subscribe(id string) <-chan ForegroundProcess
}
