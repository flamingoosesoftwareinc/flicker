package particle

import (
	"math"
	"testing"

	"flicker/core"
	"flicker/fmath"
)

func TestVelocityColor(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()

	gradient := ColorGradient{
		MinSpeed: 0.0,
		MaxSpeed: 10.0,
		MinColor: core.Color{R: 0, G: 0, B: 255}, // blue
		MaxColor: core.Color{R: 255, G: 0, B: 0}, // red
	}

	mat := VelocityColor(gradient)

	tests := []struct {
		name     string
		velocity fmath.Vec2
		wantR    uint8
		wantG    uint8
		wantB    uint8
	}{
		{
			name:     "min speed",
			velocity: fmath.Vec2{X: 0, Y: 0},
			wantR:    0,
			wantG:    0,
			wantB:    255,
		},
		{
			name:     "max speed",
			velocity: fmath.Vec2{X: 10, Y: 0},
			wantR:    255,
			wantG:    0,
			wantB:    0,
		},
		{
			name:     "mid speed",
			velocity: fmath.Vec2{X: 5, Y: 0},
			wantR:    127,
			wantG:    0,
			wantB:    127,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world.AddBody(entity, &core.Body{Velocity: tt.velocity})

			f := core.Fragment{
				Cell:   core.Cell{Rune: '·'},
				World:  world,
				Entity: entity,
			}

			got := mat(f)
			if got.FG.R != tt.wantR || got.FG.G != tt.wantG || got.FG.B != tt.wantB {
				t.Errorf("VelocityColor() = RGB(%d, %d, %d), want RGB(%d, %d, %d)",
					got.FG.R, got.FG.G, got.FG.B, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

func TestVelocityColorNoBody(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()
	// No Body component added

	gradient := ColorGradient{
		MinSpeed: 0.0,
		MaxSpeed: 10.0,
		MinColor: core.Color{R: 0, G: 0, B: 255},
		MaxColor: core.Color{R: 255, G: 0, B: 0},
	}

	mat := VelocityColor(gradient)

	originalCell := core.Cell{Rune: '·', FG: core.Color{R: 100, G: 100, B: 100}}
	f := core.Fragment{
		Cell:   originalCell,
		World:  world,
		Entity: entity,
	}

	got := mat(f)
	if got != originalCell {
		t.Errorf(
			"VelocityColor() with no Body should return original cell, got %+v, want %+v",
			got,
			originalCell,
		)
	}
}

func TestIdleAndMotion(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()

	idleRunes := []rune{'·', '•', '○', '●'}
	mat := IdleAndMotion(idleRunes, 1.0)

	// Test idle state
	world.AddBody(entity, &core.Body{Velocity: fmath.Vec2{X: 0, Y: 0}})
	f := core.Fragment{
		Cell:   core.Cell{Rune: ' '},
		World:  world,
		Entity: entity,
		Time:   core.Time{Total: 0.0},
	}

	got := mat(f)
	if got.Rune != idleRunes[0] {
		t.Errorf("IdleAndMotion() idle state = %c, want %c", got.Rune, idleRunes[0])
	}

	// Test motion state (above threshold)
	world.AddBody(entity, &core.Body{Velocity: fmath.Vec2{X: 2, Y: 0}})
	got = mat(f)
	// Should be a Braille character, not an idle rune
	isIdle := false
	for _, r := range idleRunes {
		if got.Rune == r {
			isIdle = true
			break
		}
	}
	if isIdle {
		t.Errorf("IdleAndMotion() motion state should not be an idle rune, got %c", got.Rune)
	}
}

func TestIdleAndMotionNoBody(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()
	// No Body component

	idleRunes := []rune{'·', '•', '○', '●'}
	mat := IdleAndMotion(idleRunes, 1.0)

	originalCell := core.Cell{Rune: 'X'}
	f := core.Fragment{
		Cell:   originalCell,
		World:  world,
		Entity: entity,
		Time:   core.Time{Total: 0.0},
	}

	got := mat(f)
	if got != originalCell {
		t.Errorf("IdleAndMotion() with no Body should return original cell")
	}
}

func TestBrailleDirectional(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()

	mat := BrailleDirectional()

	tests := []struct {
		name     string
		velocity fmath.Vec2
		wantRune rune
	}{
		{
			name:     "east",
			velocity: fmath.Vec2{X: 1, Y: 0},
			wantRune: '⠤',
		},
		{
			name:     "south",
			velocity: fmath.Vec2{X: 0, Y: 1},
			wantRune: '⡇',
		},
		{
			name:     "west",
			velocity: fmath.Vec2{X: -1, Y: 0},
			wantRune: '⠒',
		},
		{
			name:     "north",
			velocity: fmath.Vec2{X: 0, Y: -1},
			wantRune: '⡀',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world.AddBody(entity, &core.Body{Velocity: tt.velocity})

			f := core.Fragment{
				Cell:   core.Cell{Rune: ' '},
				World:  world,
				Entity: entity,
			}

			got := mat(f)
			if got.Rune != tt.wantRune {
				t.Errorf("BrailleDirectional() = %c, want %c", got.Rune, tt.wantRune)
			}
		})
	}
}

func TestBrailleDirectionalNoBody(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()
	// No Body component

	mat := BrailleDirectional()

	f := core.Fragment{
		Cell:   core.Cell{Rune: ' '},
		World:  world,
		Entity: entity,
	}

	got := mat(f)
	if got.Rune != '·' {
		t.Errorf("BrailleDirectional() with no Body should return '·', got %c", got.Rune)
	}
}

func TestSpeedStates(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()

	thresholds := []float64{1.0, 5.0, 10.0}
	runes := []rune{'·', '○', '◎', '●'}

	mat := SpeedStates(thresholds, runes)

	tests := []struct {
		name     string
		velocity fmath.Vec2
		wantRune rune
	}{
		{
			name:     "very slow",
			velocity: fmath.Vec2{X: 0.5, Y: 0},
			wantRune: '·',
		},
		{
			name:     "slow",
			velocity: fmath.Vec2{X: 3, Y: 0},
			wantRune: '○',
		},
		{
			name:     "medium",
			velocity: fmath.Vec2{X: 7, Y: 0},
			wantRune: '◎',
		},
		{
			name:     "fast",
			velocity: fmath.Vec2{X: 15, Y: 0},
			wantRune: '●',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world.AddBody(entity, &core.Body{Velocity: tt.velocity})

			f := core.Fragment{
				Cell:   core.Cell{Rune: ' '},
				World:  world,
				Entity: entity,
			}

			got := mat(f)
			if got.Rune != tt.wantRune {
				t.Errorf("SpeedStates() = %c, want %c", got.Rune, tt.wantRune)
			}
		})
	}
}

func TestSpeedStatesNoBody(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()
	// No Body component

	thresholds := []float64{1.0, 5.0}
	runes := []rune{'·', '○', '●'}

	mat := SpeedStates(thresholds, runes)

	f := core.Fragment{
		Cell:   core.Cell{Rune: 'X'},
		World:  world,
		Entity: entity,
	}

	got := mat(f)
	if got.Rune != ' ' {
		t.Errorf("SpeedStates() with no Body should return ' ', got %c", got.Rune)
	}
}

func TestAgeBasedSize(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()

	ageThresholds := []float64{0.5, 1.0, 2.0}
	runes := []rune{'·', '○', '◎', '●'}

	mat := AgeBasedSize(ageThresholds, runes)

	tests := []struct {
		name     string
		age      float64
		wantRune rune
	}{
		{
			name:     "newborn",
			age:      0.2,
			wantRune: '·',
		},
		{
			name:     "young",
			age:      0.7,
			wantRune: '○',
		},
		{
			name:     "mature",
			age:      1.5,
			wantRune: '◎',
		},
		{
			name:     "old",
			age:      3.0,
			wantRune: '●',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world.AddAge(entity, &core.Age{Age: tt.age})

			f := core.Fragment{
				Cell:   core.Cell{Rune: ' '},
				World:  world,
				Entity: entity,
			}

			got := mat(f)
			if got.Rune != tt.wantRune {
				t.Errorf("AgeBasedSize() = %c, want %c", got.Rune, tt.wantRune)
			}
		})
	}
}

func TestAgeBasedSizeNoAge(t *testing.T) {
	world := core.NewWorld()
	entity := world.Spawn()
	// No Age component

	ageThresholds := []float64{0.5, 1.0}
	runes := []rune{'·', '○', '●'}

	mat := AgeBasedSize(ageThresholds, runes)

	f := core.Fragment{
		Cell:   core.Cell{Rune: 'X'},
		World:  world,
		Entity: entity,
	}

	got := mat(f)
	if got.Rune != '·' {
		t.Errorf("AgeBasedSize() with no Age should return first rune '·', got %c", got.Rune)
	}
}

func TestBrailleForAngle(t *testing.T) {
	tests := []struct {
		name     string
		angle    float64
		wantRune rune
	}{
		{
			name:     "east (0 rad)",
			angle:    0,
			wantRune: '⠤',
		},
		{
			name:     "southeast (π/4 rad)",
			angle:    math.Pi / 4,
			wantRune: '⠡',
		},
		{
			name:     "south (π/2 rad)",
			angle:    math.Pi / 2,
			wantRune: '⡇',
		},
		{
			name:     "southwest (3π/4 rad)",
			angle:    3 * math.Pi / 4,
			wantRune: '⢇',
		},
		{
			name:     "west (π rad)",
			angle:    math.Pi,
			wantRune: '⠒',
		},
		{
			name:     "northwest (-3π/4 rad)",
			angle:    -3 * math.Pi / 4,
			wantRune: '⠊',
		},
		{
			name:     "north (-π/2 rad)",
			angle:    -math.Pi / 2,
			wantRune: '⡀',
		},
		{
			name:     "northeast (-π/4 rad)",
			angle:    -math.Pi / 4,
			wantRune: '⠈',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := brailleForAngle(tt.angle)
			if got != tt.wantRune {
				t.Errorf("brailleForAngle(%f) = %c, want %c", tt.angle, got, tt.wantRune)
			}
		})
	}
}

func TestLerpColor(t *testing.T) {
	a := core.Color{R: 0, G: 0, B: 255}
	b := core.Color{R: 255, G: 0, B: 0}

	tests := []struct {
		name  string
		t     float64
		wantR uint8
		wantG uint8
		wantB uint8
	}{
		{
			name:  "start",
			t:     0.0,
			wantR: 0,
			wantG: 0,
			wantB: 255,
		},
		{
			name:  "mid",
			t:     0.5,
			wantR: 127,
			wantG: 0,
			wantB: 127,
		},
		{
			name:  "end",
			t:     1.0,
			wantR: 255,
			wantG: 0,
			wantB: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lerpColor(a, b, tt.t)
			if got.R != tt.wantR || got.G != tt.wantG || got.B != tt.wantB {
				t.Errorf("lerpColor() = RGB(%d, %d, %d), want RGB(%d, %d, %d)",
					got.R, got.G, got.B, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}
