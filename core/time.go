package core

// Time holds engine timing information passed to all systems each frame.
type Time struct {
	Total float64 // seconds since start
	Delta float64 // seconds since last frame
}
