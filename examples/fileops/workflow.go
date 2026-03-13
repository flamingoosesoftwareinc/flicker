package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
)

// CycleRequest is the input for each file-wrangling cycle.
type CycleRequest struct {
	WorkDir    string `json:"work_dir"`
	ArchiveDir string `json:"archive_dir"`
	CycleNum   int    `json:"cycle_num"`
}

// CycleReport is the result of a completed cycle.
type CycleReport struct {
	CycleID        string            `json:"cycle_id"`
	CycleNum       int               `json:"cycle_num"`
	StartedAt      time.Time         `json:"started_at"`
	FilesProcessed int               `json:"files_processed"`
	FilesArchived  int               `json:"files_archived"`
	Checksums      map[string]string `json:"checksums"`
	Types          map[string]string `json:"types"`
	Sizes          map[string]int64  `json:"sizes"`
	ManifestPath   string            `json:"manifest_path"`
	CleanedUp      bool              `json:"cleaned_up"`
}

// FileEntry describes a file found during scanning.
type FileEntry struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// CleanupSignal is the event payload received from the named pipe.
type CleanupSignal struct {
	Reason string `json:"reason"`
}

type fileWrangler struct {
	wc       *flicker.WorkflowContext
	archName *flicker.Provider[string]
}

// FileWranglerDef is the workflow definition — registered on the engine at startup.
var FileWranglerDef = flicker.Define[CycleRequest, CycleReport](
	"file-wrangler", "v1",
	func(wc *flicker.WorkflowContext) flicker.Workflow[CycleRequest, CycleReport] {
		return &fileWrangler{
			wc: wc,
			archName: flicker.NewProvider[string](wc, "archive-name", func() (string, error) {
				return fmt.Sprintf("batch-%d", time.Now().UnixNano()), nil
			}),
		}
	},
)

func (w *fileWrangler) Execute(ctx context.Context, req CycleRequest) (CycleReport, error) {
	var zero CycleReport

	// --- wc.Time.Now: record cycle start ---
	cycleStart, err := w.wc.Time.Now(ctx)
	if err != nil {
		return zero, err
	}

	// --- wc.ID.New: generate cycle ID ---
	cycleID, err := w.wc.ID.New(ctx)
	if err != nil {
		return zero, err
	}

	w.wc.Log("cycle starting", "cycle_id", cycleID, "cycle_num", req.CycleNum)

	// --- Run: seed test files using dd (os/exec) ---
	_, err = flicker.Run[int](ctx, w.wc, "seed-files", func(ctx context.Context) (*int, error) {
		count := 0
		for i := range 5 {
			name := fmt.Sprintf("data-%s-%03d.bin", cycleID[:8], i)
			path := filepath.Join(req.WorkDir, name)
			cmd := exec.CommandContext(ctx, "dd",
				"if=/dev/urandom",
				fmt.Sprintf("of=%s", path),
				"bs=1024",
				fmt.Sprintf("count=%d", (i+1)*10),
			)
			cmd.Stderr = io.Discard
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("seed file %s: %w", name, err)
			}
			count++
		}
		return &count, nil
	})
	if err != nil {
		return zero, err
	}

	// --- Run: scan directory ---
	files, err := flicker.Run[[]FileEntry](
		ctx,
		w.wc,
		"scan-directory",
		func(_ context.Context) (*[]FileEntry, error) {
			entries, err := os.ReadDir(req.WorkDir)
			if err != nil {
				return nil, err
			}
			var result []FileEntry
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				result = append(result, FileEntry{
					Name: e.Name(),
					Size: info.Size(),
					Path: filepath.Join(req.WorkDir, e.Name()),
				})
			}
			return &result, nil
		},
	)
	if err != nil {
		return zero, err
	}

	if len(*files) == 0 {
		return zero, flicker.Permanent(fmt.Errorf("no files found in %s", req.WorkDir))
	}

	// --- Parallel: classify + checksum + measure ---
	var checksums *map[string]string
	var types *map[string]string
	var sizes *map[string]int64

	err = flicker.Parallel(ctx, w.wc,
		// Branch 1: detect file types using `file` command (os/exec)
		flicker.NewBranch("classify", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			result, err := flicker.Run[map[string]string](
				ctx,
				wc,
				"detect-types",
				func(ctx context.Context) (*map[string]string, error) {
					m := make(map[string]string)
					for _, f := range *files {
						out, err := exec.CommandContext(ctx, "file", "-b", f.Path).Output()
						if err != nil {
							m[f.Name] = "unknown"
						} else {
							m[f.Name] = strings.TrimSpace(string(out))
						}
					}
					return &m, nil
				},
			)
			types = result
			return err
		}),

		// Branch 2: compute SHA256 checksums (Go crypto)
		flicker.NewBranch("checksum", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			result, err := flicker.Run[map[string]string](
				ctx,
				wc,
				"compute-checksums",
				func(_ context.Context) (*map[string]string, error) {
					m := make(map[string]string)
					for _, f := range *files {
						h, err := hashFile(f.Path)
						if err != nil {
							m[f.Name] = "error"
						} else {
							m[f.Name] = h
						}
					}
					return &m, nil
				},
			)
			checksums = result
			return err
		}),

		// Branch 3: measure sizes using `du` (os/exec)
		flicker.NewBranch("measure", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			result, err := flicker.Run[map[string]int64](
				ctx,
				wc,
				"measure-sizes",
				func(ctx context.Context) (*map[string]int64, error) {
					m := make(map[string]int64)
					for _, f := range *files {
						out, err := exec.CommandContext(ctx, "du", "-b", f.Path).Output()
						if err == nil {
							parts := strings.Fields(string(out))
							if len(parts) > 0 {
								var sz int64
								fmt.Sscanf(parts[0], "%d", &sz)
								m[f.Name] = sz
							}
						}
					}
					return &m, nil
				},
			)
			sizes = result
			return err
		}),
	)
	if err != nil {
		return zero, err
	}

	// --- Provider.Get: generate archive name ---
	archiveName, err := w.archName.Get(ctx)
	if err != nil {
		return zero, err
	}

	// --- Run: write JSON manifest to disk ---
	manifestPath, err := flicker.Run[string](
		ctx,
		w.wc,
		"write-manifest",
		func(_ context.Context) (*string, error) {
			manifest := map[string]any{
				"cycle_id":  cycleID,
				"timestamp": cycleStart,
				"files":     files,
				"checksums": checksums,
				"types":     types,
				"sizes":     sizes,
			}
			data, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				return nil, err
			}
			p := filepath.Join(req.WorkDir, "manifest.json")
			if err := os.WriteFile(p, data, 0o644); err != nil {
				return nil, err
			}
			return &p, nil
		},
	)
	if err != nil {
		return zero, err
	}

	// --- SleepUntil: pause 5s before archiving (demonstrates suspend/resume) ---
	if err := w.wc.SleepUntil(ctx, cycleStart.Add(5*time.Second)); err != nil {
		return zero, err
	}

	// --- Run: archive files using mv (os/exec) ---
	archived, err := flicker.Run[int](
		ctx,
		w.wc,
		"archive-files",
		func(ctx context.Context) (*int, error) {
			archDir := filepath.Join(req.ArchiveDir, archiveName)
			if err := os.MkdirAll(archDir, 0o755); err != nil {
				return nil, err
			}
			count := 0
			for _, f := range *files {
				dest := filepath.Join(archDir, f.Name)
				if err := exec.CommandContext(ctx, "mv", f.Path, dest).Run(); err == nil {
					count++
				}
			}
			// Move manifest too.
			_ = exec.CommandContext(ctx, "mv",
				filepath.Join(req.WorkDir, "manifest.json"),
				filepath.Join(archDir, "manifest.json"),
			).Run()
			return &count, nil
		},
	)
	if err != nil {
		return zero, err
	}

	// --- wc.Time.Now: record post-archive time ---
	_, err = w.wc.Time.Now(ctx)
	if err != nil {
		return zero, err
	}

	// --- WaitForEvent: wait for cleanup signal from named pipe ---
	cleanedUp := false
	signal, err := flicker.WaitForEvent[CleanupSignal](
		ctx, w.wc, "await-cleanup",
		fmt.Sprintf("cleanup:%d", req.CycleNum),
		15*time.Second,
	)
	if err != nil {
		if !errors.Is(err, flicker.ErrEventTimeout) {
			return zero, err
		}
		// Timeout is fine — skip cleanup.
		w.wc.Log("cleanup timeout, skipping", "cycle_num", req.CycleNum)
	}

	if signal != nil {
		// Got cleanup signal — prune old archives.
		result, err := flicker.Run[bool](
			ctx,
			w.wc,
			"cleanup-archives",
			func(_ context.Context) (*bool, error) {
				entries, err := os.ReadDir(req.ArchiveDir)
				if err != nil {
					return flicker.Val(false), nil
				}
				// Keep only the 3 most recent archives.
				if len(entries) > 3 {
					for _, e := range entries[:len(entries)-3] {
						os.RemoveAll(filepath.Join(req.ArchiveDir, e.Name()))
					}
				}
				return flicker.Val(true), nil
			},
		)
		if err == nil && result != nil {
			cleanedUp = *result
		}
	}

	return CycleReport{
		CycleID:        cycleID,
		CycleNum:       req.CycleNum,
		StartedAt:      cycleStart,
		FilesProcessed: len(*files),
		FilesArchived:  *archived,
		Checksums:      *checksums,
		Types:          *types,
		Sizes:          *sizes,
		ManifestPath:   *manifestPath,
		CleanedUp:      cleanedUp,
	}, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func seedFiles(dir string, count int) (int, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	return count, nil
}
