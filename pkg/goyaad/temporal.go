package goyaad

// TemporalState is a classification of time for an object
type TemporalState int

const (
	// Past in time
	Past TemporalState = iota
	// Current in time
	Current
	// Future in time
	Future
)
