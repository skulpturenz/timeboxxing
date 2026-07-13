package sessionnew

type Reporter interface {
	Subscribe() <-chan ForegroundProcess
	Publish(incoming ForegroundProcess)
}
