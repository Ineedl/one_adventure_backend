package registration

type State string

const (
	StateStarting     State = "starting"
	StateRegistering  State = "registering"
	StateActive       State = "active"
	StateRetryWaiting State = "retry_waiting"
	StateStopped      State = "stopped"
)
