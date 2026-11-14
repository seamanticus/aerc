package ipc

type Handler interface {
	// Command runs an aerc command. It is only called from functions passed
	// to Schedule.
	Command(args []string) error
	// Schedule runs fn on the goroutine that owns the application state.
	// It may block until there is room in that goroutine's queue.
	Schedule(fn func())
	// ReportError shows, in the running instance, the error of a command
	// whose IPC client is no longer waiting for the result. It is only
	// called from functions passed to Schedule.
	ReportError(err error)
}
