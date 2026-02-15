package core

import (
	"fmt"
	"strings"

	"flicker/fmath"
)

// ============================================================================
// Core Timeline Types
// ============================================================================

// Timeline orchestrates timed events and animations within a scene.
// It follows the behavior pattern - wraps as a behavior attached to a container entity.
type Timeline struct {
	world     *World
	entity    Entity // Container entity for timeline behavior
	tracks    []*Track
	startTime float64 // Absolute time when timeline started (t.Total)
	isRunning bool
	isPaused  bool
	loop      bool
}

// Track represents one channel of clips that play sequentially.
// Multiple tracks run in parallel.
type Track struct {
	timeline     *Timeline
	clips        []Clip
	currentIndex int
	clipStart    float64 // Time when current clip started (relative to timeline start)
}

// Clip is the interface for any timed action.
// Follows the Phase/Controller pattern from particle system.
type Clip interface {
	Duration() float64           // Duration in seconds (0 = instant)
	Start(ctx ClipContext)       // Initialize
	Update(elapsed float64) bool // Update, returns true when complete
	End()                        // Cleanup
}

// ClipContext provides resources to clips during initialization.
type ClipContext struct {
	Timeline  *Timeline
	World     *World
	StartTime float64 // Absolute time (t.Total) when clip starts
}

// ============================================================================
// Timeline API
// ============================================================================

// NewTimeline creates a new timeline attached to the given world.
// The timeline is implemented as a behavior on a container entity.
func NewTimeline(world *World) *Timeline {
	tl := &Timeline{
		world:     world,
		tracks:    make([]*Track, 0),
		startTime: 0,
		isRunning: false,
		isPaused:  false,
		loop:      false,
	}

	// Create container entity for timeline update behavior
	tl.entity = world.Spawn()
	world.AddBehavior(tl.entity, NewBehavior(tl.update))

	return tl
}

// Start begins timeline playback from the given time.
// Uses absolute time pattern (startTime = t.Total).
func (tl *Timeline) Start(t Time) {
	tl.startTime = t.Total
	tl.isRunning = true
	tl.isPaused = false

	// Initialize all tracks
	for _, track := range tl.tracks {
		track.currentIndex = 0
		track.clipStart = 0

		// Start first clip if available
		if len(track.clips) > 0 {
			ctx := ClipContext{
				Timeline:  tl,
				World:     tl.world,
				StartTime: t.Total,
			}
			track.clips[0].Start(ctx)
		}
	}
}

// Pause pauses timeline playback.
func (tl *Timeline) Pause() {
	tl.isPaused = true
}

// Resume resumes timeline playback.
func (tl *Timeline) Resume() {
	tl.isPaused = false
}

// Stop stops timeline playback without completing current clips.
// Clips remain at their current state and do not call End().
func (tl *Timeline) Stop() {
	tl.isRunning = false
	// Don't call End() on clips - just stop updating them
}

// SetLoop controls whether the timeline loops when all tracks complete.
func (tl *Timeline) SetLoop(loop bool) {
	tl.loop = loop
}

// AddTrack adds a new track to the timeline and returns it for chaining.
func (tl *Timeline) AddTrack() *Track {
	track := &Track{
		timeline:     tl,
		clips:        make([]Clip, 0),
		currentIndex: 0,
		clipStart:    0,
	}
	tl.tracks = append(tl.tracks, track)
	return track
}

// Cleanup despawns the container entity, removing the timeline behavior.
// Should be called in scene OnExit.
func (tl *Timeline) Cleanup() {
	if tl.entity != 0 {
		tl.world.Despawn(tl.entity)
		tl.entity = 0
	}
}

// update is the behavior function called each frame.
// Implements the timeline update loop with parallel track execution.
func (tl *Timeline) update(t Time, e Entity, w *World) {
	if !tl.isRunning || tl.isPaused {
		return
	}

	elapsed := t.Total - tl.startTime // Absolute time math

	// Update all tracks in parallel
	for _, track := range tl.tracks {
		// Keep updating clips until we hit one that's not complete
		// This handles instant clips (duration = 0) that complete immediately
		for track.currentIndex < len(track.clips) {
			clip := track.clips[track.currentIndex]
			clipElapsed := elapsed - track.clipStart // Relative to clip start

			if clip.Update(clipElapsed) {
				clip.End()
				track.currentIndex++
				track.clipStart = elapsed // Next clip starts now

				// Start next clip if available
				if track.currentIndex < len(track.clips) {
					ctx := ClipContext{
						Timeline:  tl,
						World:     w,
						StartTime: t.Total,
					}
					track.clips[track.currentIndex].Start(ctx)
					// Continue loop to update the new clip in the same frame
				} else {
					// This track is done
					break
				}
			} else {
				// Clip not complete yet, stop updating this track for this frame
				break
			}
		}
	}

	// Check if all tracks are complete
	allComplete := true
	for _, track := range tl.tracks {
		if track.currentIndex < len(track.clips) {
			allComplete = false
			break
		}
	}

	// Handle loop/completion
	if allComplete {
		if tl.loop {
			// Restart timeline
			tl.Start(t)
		} else {
			tl.isRunning = false
		}
	}
}

// ============================================================================
// Track API (Fluent Builder)
// ============================================================================

// Add adds a clip to the track and returns the track for chaining.
func (tr *Track) Add(clip Clip) *Track {
	tr.clips = append(tr.clips, clip)
	return tr
}

// At adds a clip at a specific time by inserting a delay if needed.
// Time is relative to the track's current position.
func (tr *Track) At(time float64, clip Clip) *Track {
	// Calculate current track duration
	currentDuration := 0.0
	for _, c := range tr.clips {
		currentDuration += c.Duration()
	}

	// Add delay if needed
	if time > currentDuration {
		delay := time - currentDuration
		tr.clips = append(tr.clips, NewDelayClip(delay))
	}

	// Add the clip
	tr.clips = append(tr.clips, clip)
	return tr
}

// Sequence adds multiple clips in sequence and returns the track for chaining.
func (tr *Track) Sequence(clips ...Clip) *Track {
	tr.clips = append(tr.clips, clips...)
	return tr
}

// ============================================================================
// CallbackClip - Execute a function at a specific time
// ============================================================================

// CallbackClip executes a callback function instantly.
type CallbackClip struct {
	callback  func(*World, Time)
	executed  bool
	world     *World
	startTime float64
}

// NewCallbackClip creates a clip that executes a callback instantly.
func NewCallbackClip(callback func(*World, Time)) *CallbackClip {
	return &CallbackClip{
		callback: callback,
		executed: false,
	}
}

func (c *CallbackClip) Duration() float64 {
	return 0 // Instant
}

func (c *CallbackClip) Start(ctx ClipContext) {
	c.world = ctx.World
	c.startTime = ctx.StartTime
	c.executed = false
}

func (c *CallbackClip) Update(elapsed float64) bool {
	if !c.executed && c.callback != nil {
		// Execute callback with time from context
		c.callback(c.world, Time{Total: c.startTime + elapsed, Delta: 0})
		c.executed = true
	}
	return true // Always complete immediately
}

func (c *CallbackClip) End() {
	// No cleanup needed
}

// ============================================================================
// DelayClip - Wait for a duration
// ============================================================================

// DelayClip waits for a specified duration without doing anything.
type DelayClip struct {
	duration float64
}

// NewDelayClip creates a clip that waits for the specified duration.
func NewDelayClip(duration float64) *DelayClip {
	return &DelayClip{duration: duration}
}

func (c *DelayClip) Duration() float64 {
	return c.duration
}

func (c *DelayClip) Start(ctx ClipContext) {
	// No initialization needed
}

func (c *DelayClip) Update(elapsed float64) bool {
	return elapsed >= c.duration
}

func (c *DelayClip) End() {
	// No cleanup needed
}

// ============================================================================
// TweenClip - Animate a value over time
// ============================================================================

// TweenClip animates a value from start to end over a duration.
type TweenClip struct {
	from     float64
	to       float64
	duration float64
	setter   func(float64)
	easing   func(float64) float64
}

// NewTweenClip creates a clip that tweens a value over time.
// The setter function is called each frame with the interpolated value.
func NewTweenClip(from, to, duration float64, setter func(float64)) *TweenClip {
	return &TweenClip{
		from:     from,
		to:       to,
		duration: duration,
		setter:   setter,
		easing:   fmath.EaseLinear, // Default to linear
	}
}

// WithEasing sets the easing function for the tween.
func (c *TweenClip) WithEasing(easing func(float64) float64) *TweenClip {
	c.easing = easing
	return c
}

func (c *TweenClip) Duration() float64 {
	return c.duration
}

func (c *TweenClip) Start(ctx ClipContext) {
	// No initialization needed
}

func (c *TweenClip) Update(elapsed float64) bool {
	t := elapsed / c.duration
	if t > 1.0 {
		t = 1.0
	}

	// Apply easing
	easedT := c.easing(t)

	// Interpolate and set value
	value := fmath.Lerp(c.from, c.to, easedT)
	if c.setter != nil {
		c.setter(value)
	}

	return elapsed >= c.duration
}

func (c *TweenClip) End() {
	// Ensure final value is set
	if c.setter != nil {
		c.setter(c.to)
	}
}

// ============================================================================
// PropertyTweenClip - Animate entity properties
// ============================================================================

// PropertyTweenClip animates specific properties of an entity's transform.
// Supports: "position.x", "position.y", "position.z", "rotation", "scale.x", "scale.y", "scale.z"
type PropertyTweenClip struct {
	entity   Entity
	property string
	from     float64
	to       float64
	duration float64
	easing   func(float64) float64
	world    *World
}

// NewPropertyTweenClip creates a clip that animates an entity property.
func NewPropertyTweenClip(
	entity Entity,
	property string,
	from, to, duration float64,
) *PropertyTweenClip {
	return &PropertyTweenClip{
		entity:   entity,
		property: property,
		from:     from,
		to:       to,
		duration: duration,
		easing:   fmath.EaseLinear, // Default to linear
	}
}

// WithEasing sets the easing function for the tween.
func (c *PropertyTweenClip) WithEasing(easing func(float64) float64) *PropertyTweenClip {
	c.easing = easing
	return c
}

func (c *PropertyTweenClip) Duration() float64 {
	return c.duration
}

func (c *PropertyTweenClip) Start(ctx ClipContext) {
	c.world = ctx.World
}

func (c *PropertyTweenClip) Update(elapsed float64) bool {
	t := elapsed / c.duration
	if t > 1.0 {
		t = 1.0
	}

	// Apply easing
	easedT := c.easing(t)

	// Interpolate value
	value := fmath.Lerp(c.from, c.to, easedT)

	// Set the property
	c.setProperty(value)

	return elapsed >= c.duration
}

func (c *PropertyTweenClip) End() {
	// Ensure final value is set
	c.setProperty(c.to)
}

// setProperty sets the property value on the entity's transform.
func (c *PropertyTweenClip) setProperty(value float64) {
	transform := c.world.Transform(c.entity)
	if transform == nil {
		return
	}

	switch c.property {
	case "position.x":
		transform.Position.X = value
	case "position.y":
		transform.Position.Y = value
	case "position.z":
		transform.Position.Z = value
	case "rotation":
		transform.Rotation = value
	case "scale.x":
		transform.Scale.X = value
	case "scale.y":
		transform.Scale.Y = value
	case "scale.z":
		transform.Scale.Z = value
	default:
		// Invalid property - ignore
	}
}

// ============================================================================
// ParallelClip - Run multiple clips simultaneously
// ============================================================================

// ParallelClip runs multiple clips at the same time.
// Completes when all child clips have completed.
type ParallelClip struct {
	clips    []Clip
	duration float64
}

// NewParallelClip creates a clip that runs multiple clips in parallel.
func NewParallelClip(clips ...Clip) *ParallelClip {
	// Calculate duration (longest clip)
	maxDuration := 0.0
	for _, clip := range clips {
		if clip.Duration() > maxDuration {
			maxDuration = clip.Duration()
		}
	}

	return &ParallelClip{
		clips:    clips,
		duration: maxDuration,
	}
}

func (c *ParallelClip) Duration() float64 {
	return c.duration
}

func (c *ParallelClip) Start(ctx ClipContext) {
	// Start all child clips
	for _, clip := range c.clips {
		clip.Start(ctx)
	}
}

func (c *ParallelClip) Update(elapsed float64) bool {
	allComplete := true

	// Update all clips
	for _, clip := range c.clips {
		if !clip.Update(elapsed) {
			allComplete = false
		}
	}

	// Also check if we've exceeded the max duration
	return allComplete || elapsed >= c.duration
}

func (c *ParallelClip) End() {
	// End all child clips
	for _, clip := range c.clips {
		clip.End()
	}
}

// ============================================================================
// SequenceClip - Run clips one after another
// ============================================================================

// SequenceClip runs clips sequentially.
// Stores context during Start() to properly initialize sub-clips.
type SequenceClip struct {
	clips        []Clip
	duration     float64
	currentIndex int
	clipStart    float64
	ctx          ClipContext // Store context from Start()
}

// NewSequenceClip creates a clip that runs clips in sequence.
func NewSequenceClip(clips ...Clip) *SequenceClip {
	// Calculate total duration
	totalDuration := 0.0
	for _, clip := range clips {
		totalDuration += clip.Duration()
	}

	return &SequenceClip{
		clips:        clips,
		duration:     totalDuration,
		currentIndex: 0,
		clipStart:    0,
	}
}

func (c *SequenceClip) Duration() float64 {
	return c.duration
}

func (c *SequenceClip) Start(ctx ClipContext) {
	c.ctx = ctx // Store context for later use
	c.currentIndex = 0
	c.clipStart = 0

	// Start first clip if available
	if len(c.clips) > 0 {
		c.clips[0].Start(ctx)
	}
}

func (c *SequenceClip) Update(elapsed float64) bool {
	// Keep updating clips until we hit one that's not complete
	// This handles instant clips (duration = 0) that complete immediately
	for c.currentIndex < len(c.clips) {
		clip := c.clips[c.currentIndex]
		clipElapsed := elapsed - c.clipStart

		if clip.Update(clipElapsed) {
			clip.End()
			c.currentIndex++
			c.clipStart = elapsed

			// Start next clip if available
			if c.currentIndex < len(c.clips) {
				// Use stored context (StartTime is from parent, which is fine)
				c.clips[c.currentIndex].Start(c.ctx)
				// Continue loop to update the new clip in the same frame
			} else {
				// Sequence complete
				break
			}
		} else {
			// Clip not complete yet, return false
			return false
		}
	}

	return c.currentIndex >= len(c.clips)
}

func (c *SequenceClip) End() {
	// End current clip if any
	if c.currentIndex < len(c.clips) {
		c.clips[c.currentIndex].End()
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// PropertyPath is a helper to parse property paths for debugging.
// Not used in the implementation but useful for error messages.
func PropertyPath(property string) (component string, field string, ok bool) {
	parts := strings.Split(property, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ValidateProperty checks if a property string is valid.
func ValidateProperty(property string) error {
	validProperties := []string{
		"position.x", "position.y", "position.z",
		"rotation",
		"scale.x", "scale.y", "scale.z",
	}

	for _, valid := range validProperties {
		if property == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid property: %s (valid: %v)", property, validProperties)
}
