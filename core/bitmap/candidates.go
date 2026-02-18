package bitmap

import "math/bits"

// candidate holds a Unicode character and its 54-bit template pattern.
// The pattern encodes a 6×9 sample grid (row-major, bit = row*6+col).
type candidate struct {
	r       rune
	pattern uint64
}

// candidates is the pre-computed table of character templates.
var candidates []candidate

func init() {
	candidates = buildCandidates()
}

// sampleCellThreshold extracts a 54-bit pattern from a bitmap at cell position (cx, cy).
// Each cell covers a 6×9 pixel region. A bit is set if the corresponding
// pixel has alpha strictly greater than thresh.
func sampleCellThreshold(bm *Bitmap, cx, cy int, thresh float64) uint64 {
	var pattern uint64
	for row := range 9 {
		for col := range 6 {
			px := cx*6 + col
			py := cy*9 + row
			if px < bm.Width && py < bm.Height && bm.Alpha[py*bm.Width+px] > thresh {
				pattern |= 1 << uint(row*6+col)
			}
		}
	}
	return pattern
}

// bestMatch finds the candidate with minimum hamming distance to the given pattern.
func bestMatch(pattern uint64) (rune, int) {
	bestRune := ' '
	bestDist := 55
	for _, c := range candidates {
		dist := bits.OnesCount64(pattern ^ c.pattern)
		if dist < bestDist {
			bestDist = dist
			bestRune = c.r
			if dist == 0 {
				break
			}
		}
	}
	return bestRune, bestDist
}

func buildCandidates() []candidate {
	var cs []candidate
	cs = append(cs, buildSextants()...)
	cs = append(cs, buildBlockElements()...)
	return cs
}

// --- Sextants ---
//
// Sextants divide the cell into a 2×3 grid of sub-cells:
//
//	1(TL) 2(TR)
//	3(ML) 4(MR)
//	5(BL) 6(BR)
//
// Each sub-cell maps to a 3×3 block of samples in the 6×9 grid.
// Unicode positions are numbered row-major (1=TL, 2=TR, 3=ML, 4=MR, 5=BL, 6=BR)
// and map to bits 0–5 respectively. Codepoints U+1FB00–U+1FB3B are assigned
// in order of increasing pattern number (1–62), skipping patterns 21 (left half,
// U+258C) and 42 (right half, U+2590) which have existing block element chars.

func sextantPattern(mask byte) uint64 {
	var pattern uint64
	for bit := 0; bit < 6; bit++ {
		if mask&(1<<uint(bit)) == 0 {
			continue
		}
		col0 := (bit % 2) * 3 // 0 or 3
		row0 := (bit / 2) * 3 // 0, 3, or 6
		for dr := range 3 {
			for dc := range 3 {
				pattern |= 1 << uint((row0+dr)*6+(col0+dc))
			}
		}
	}
	return pattern
}

func buildSextants() []candidate {
	var cs []candidate

	// Pattern 63 = full block.
	cs = append(cs, candidate{r: '\u2588', pattern: sextantPattern(0x3F)})

	sextantRune := rune(0x1FB00)
	for pat := byte(1); pat < 63; pat++ {
		switch pat {
		case 0x15: // left half (bits 0,2,4 = TL+ML+BL)
			cs = append(cs, candidate{r: '\u258C', pattern: sextantPattern(pat)})
		case 0x2A: // right half (bits 1,3,5 = TR+MR+BR)
			cs = append(cs, candidate{r: '\u2590', pattern: sextantPattern(pat)})
		default:
			cs = append(cs, candidate{r: sextantRune, pattern: sextantPattern(pat)})
			sextantRune++
		}
	}

	return cs
}

// --- Block elements ---

func buildBlockElements() []candidate {
	var cs []candidate

	// Upper half block U+2580
	cs = append(cs, candidate{r: '\u2580', pattern: rectPattern(0, 0, 6, 5)})
	// Lower half block U+2584
	cs = append(cs, candidate{r: '\u2584', pattern: rectPattern(0, 5, 6, 9)})

	// Quadrant characters U+2596–U+259F.
	// Split: cols 0-2 / 3-5, rows 0-4 / 5-8.
	type quadDef struct {
		r              rune
		tl, tr, bl, br bool
	}
	quads := []quadDef{
		{'\u2596', false, false, true, false},
		{'\u2597', false, false, false, true},
		{'\u2598', true, false, false, false},
		{'\u2599', true, false, true, true},
		{'\u259A', true, false, false, true},
		{'\u259B', true, true, true, false},
		{'\u259C', true, true, false, true},
		{'\u259D', false, true, false, false},
		{'\u259E', false, true, true, false},
		{'\u259F', false, true, true, true},
	}
	for _, q := range quads {
		var p uint64
		if q.tl {
			p |= rectPattern(0, 0, 3, 5)
		}
		if q.tr {
			p |= rectPattern(3, 0, 6, 5)
		}
		if q.bl {
			p |= rectPattern(0, 5, 3, 9)
		}
		if q.br {
			p |= rectPattern(3, 5, 6, 9)
		}
		cs = append(cs, candidate{r: q.r, pattern: p})
	}

	return cs
}

// rectPattern generates a 54-bit template for a filled rectangle
// within the 6×9 sample grid. Coordinates are in sample units.
func rectPattern(col0, row0, col1, row1 int) uint64 {
	var pattern uint64
	for row := row0; row < row1; row++ {
		for col := col0; col < col1; col++ {
			pattern |= 1 << uint(row*6+col)
		}
	}
	return pattern
}
