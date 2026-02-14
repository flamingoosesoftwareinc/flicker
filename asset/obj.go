package asset

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"flicker/fmath"
)

type Mesh struct {
	Vertices []fmath.Vec3
	Normals  []fmath.Vec3
	UVs      []fmath.Vec2
	Faces    []Face
}

type Face struct {
	V  [3]int // vertex indices (0-based)
	VN [3]int // normal indices (-1 if absent)
	VT [3]int // UV indices (-1 if absent)
}

func LoadOBJ(path string) (*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	m := &Mesh{}
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "v":
			v, err := parseVec3(fields[1:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			m.Vertices = append(m.Vertices, v)

		case "vn":
			vn, err := parseVec3(fields[1:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			m.Normals = append(m.Normals, vn)

		case "vt":
			vt, err := parseVec2(fields[1:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			m.UVs = append(m.UVs, vt)

		case "f":
			faces, err := parseFace(fields[1:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			m.Faces = append(m.Faces, faces...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Cache) GetOrLoadOBJ(path string) (*Mesh, error) {
	if v, ok := c.Get(path); ok {
		return v.(*Mesh), nil
	}
	m, err := LoadOBJ(path)
	if err != nil {
		return nil, err
	}
	c.Put(path, m)
	return m, nil
}

func parseVec3(fields []string) (fmath.Vec3, error) {
	if len(fields) < 3 {
		return fmath.Vec3{}, fmt.Errorf("expected 3 floats, got %d", len(fields))
	}
	x, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return fmath.Vec3{}, err
	}
	y, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return fmath.Vec3{}, err
	}
	z, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return fmath.Vec3{}, err
	}
	return fmath.Vec3{X: x, Y: y, Z: z}, nil
}

func parseVec2(fields []string) (fmath.Vec2, error) {
	if len(fields) < 2 {
		return fmath.Vec2{}, fmt.Errorf("expected 2 floats, got %d", len(fields))
	}
	x, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return fmath.Vec2{}, err
	}
	y, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return fmath.Vec2{}, err
	}
	return fmath.Vec2{X: x, Y: y}, nil
}

// parseFace parses face vertex references and fan-triangulates polygons with >3 vertices.
// Handles formats: v, v/vt, v/vt/vn, v//vn
func parseFace(fields []string) ([]Face, error) {
	if len(fields) < 3 {
		return nil, fmt.Errorf("face needs at least 3 vertices, got %d", len(fields))
	}

	type faceVert struct {
		v, vt, vn int
	}
	verts := make([]faceVert, len(fields))

	for i, field := range fields {
		parts := strings.Split(field, "/")
		vi, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("bad vertex index %q: %w", parts[0], err)
		}
		fv := faceVert{v: vi - 1, vt: -1, vn: -1}

		if len(parts) >= 2 && parts[1] != "" {
			vti, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("bad UV index %q: %w", parts[1], err)
			}
			fv.vt = vti - 1
		}
		if len(parts) >= 3 && parts[2] != "" {
			vni, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, fmt.Errorf("bad normal index %q: %w", parts[2], err)
			}
			fv.vn = vni - 1
		}
		verts[i] = fv
	}

	// Fan-triangulate: (0,1,2), (0,2,3), (0,3,4), ...
	faces := make([]Face, 0, len(verts)-2)
	for i := 2; i < len(verts); i++ {
		faces = append(faces, Face{
			V:  [3]int{verts[0].v, verts[i-1].v, verts[i].v},
			VN: [3]int{verts[0].vn, verts[i-1].vn, verts[i].vn},
			VT: [3]int{verts[0].vt, verts[i-1].vt, verts[i].vt},
		})
	}
	return faces, nil
}
