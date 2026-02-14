package asset

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOBJ_Suzanne(t *testing.T) {
	m, err := LoadOBJ("../suzanne.obj")
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	if len(m.Vertices) != 511 {
		t.Errorf("vertices = %d, want 511", len(m.Vertices))
	}
	if len(m.Normals) != 507 {
		t.Errorf("normals = %d, want 507", len(m.Normals))
	}
	if len(m.UVs) != 590 {
		t.Errorf("UVs = %d, want 590", len(m.UVs))
	}
	if len(m.Faces) != 968 {
		t.Errorf("faces = %d, want 968", len(m.Faces))
	}

	// Spot-check first vertex.
	v0 := m.Vertices[0]
	if !floatEq(v0.X, 0.4375) || !floatEq(v0.Y, 0.164063) || !floatEq(v0.Z, 0.765625) {
		t.Errorf("vertex[0] = %v, want {0.4375 0.164063 0.765625}", v0)
	}

	// All indices in bounds.
	for i, f := range m.Faces {
		for j := range 3 {
			if f.V[j] < 0 || f.V[j] >= len(m.Vertices) {
				t.Errorf("face[%d].V[%d] = %d, out of range [0,%d)", i, j, f.V[j], len(m.Vertices))
			}
			if f.VN[j] != -1 && (f.VN[j] < 0 || f.VN[j] >= len(m.Normals)) {
				t.Errorf("face[%d].VN[%d] = %d, out of range [0,%d)", i, j, f.VN[j], len(m.Normals))
			}
			if f.VT[j] != -1 && (f.VT[j] < 0 || f.VT[j] >= len(m.UVs)) {
				t.Errorf("face[%d].VT[%d] = %d, out of range [0,%d)", i, j, f.VT[j], len(m.UVs))
			}
		}
	}
}

func TestLoadOBJ_VerticesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tri.obj")
	content := `# minimal
v 0 0 0
v 1 0 0
v 0 1 0
f 1 2 3
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadOBJ(path)
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	if len(m.Vertices) != 3 {
		t.Errorf("vertices = %d, want 3", len(m.Vertices))
	}
	if len(m.Faces) != 1 {
		t.Errorf("faces = %d, want 1", len(m.Faces))
	}

	f := m.Faces[0]
	if f.V != [3]int{0, 1, 2} {
		t.Errorf("face V = %v, want [0 1 2]", f.V)
	}
	if f.VN != [3]int{-1, -1, -1} {
		t.Errorf("face VN = %v, want [-1 -1 -1]", f.VN)
	}
	if f.VT != [3]int{-1, -1, -1} {
		t.Errorf("face VT = %v, want [-1 -1 -1]", f.VT)
	}
}

func TestLoadOBJ_QuadTriangulation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quad.obj")
	content := `v 0 0 0
v 1 0 0
v 1 1 0
v 0 1 0
f 1 2 3 4
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadOBJ(path)
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	if len(m.Faces) != 2 {
		t.Fatalf("quad should produce 2 triangles, got %d", len(m.Faces))
	}

	// Fan: (0,1,2), (0,2,3)
	if m.Faces[0].V != [3]int{0, 1, 2} {
		t.Errorf("face[0].V = %v, want [0 1 2]", m.Faces[0].V)
	}
	if m.Faces[1].V != [3]int{0, 2, 3} {
		t.Errorf("face[1].V = %v, want [0 2 3]", m.Faces[1].V)
	}
}

func TestLoadOBJ_CommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comments.obj")
	content := `# This is a comment
# Another comment

v 1 2 3

v 4 5 6
v 7 8 9
s off
o MyObject
g group1
usemtl material
mtllib file.mtl
f 1 2 3
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadOBJ(path)
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	if len(m.Vertices) != 3 {
		t.Errorf("vertices = %d, want 3", len(m.Vertices))
	}
	if len(m.Faces) != 1 {
		t.Errorf("faces = %d, want 1", len(m.Faces))
	}
}

func TestLoadOBJ_Nonexistent(t *testing.T) {
	_, err := LoadOBJ("/nonexistent/path.obj")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadOBJ_VVnFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vvn.obj")
	content := `v 0 0 0
v 1 0 0
v 0 1 0
vn 0 0 1
f 1//1 2//1 3//1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadOBJ(path)
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	f := m.Faces[0]
	if f.VN != [3]int{0, 0, 0} {
		t.Errorf("face VN = %v, want [0 0 0]", f.VN)
	}
	if f.VT != [3]int{-1, -1, -1} {
		t.Errorf("face VT = %v, want [-1 -1 -1]", f.VT)
	}
}

func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-4
}
