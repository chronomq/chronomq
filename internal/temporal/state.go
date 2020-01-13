package temporal

// State is a classification of time for an object
type State int

const (
	// Past in time
	Past State = iota
	// Current in time
	Current
	// Future in time
	Future
)
