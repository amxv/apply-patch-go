package applypatch

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type snapshotEntry struct {
	IsDir   bool
	Content []byte
}

func TestUpstreamScenarios(t *testing.T) {
	scenariosDir := filepath.Join("testdata", "upstream-scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatalf("ReadDir scenarios: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			runScenario(t, filepath.Join(scenariosDir, name))
		})
	}
}

func runScenario(t *testing.T, dir string) {
	t.Helper()
	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs scenario dir: %v", err)
	}
	dir = absDir
	tmp := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	if info, err := os.Stat(inputDir); err == nil && info.IsDir() {
		if err := copyDirRecursive(inputDir, tmp); err != nil {
			t.Fatalf("copy input: %v", err)
		}
	}
	patchBytes, err := os.ReadFile(filepath.Join(dir, "patch.txt"))
	if err != nil {
		t.Fatalf("read patch: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	var stdout, stderr bytes.Buffer
	_ = ApplyPatch(string(patchBytes), &stdout, &stderr)

	expected, err := snapshotDir(filepath.Join(dir, "expected"))
	if err != nil {
		t.Fatalf("snapshot expected: %v", err)
	}
	actual, err := snapshotDir(tmp)
	if err != nil {
		t.Fatalf("snapshot actual: %v", err)
	}
	if diff := compareSnapshots(expected, actual); diff != "" {
		t.Fatalf("scenario mismatch:\n%s\nstdout=%q\nstderr=%q", diff, stdout.String(), stderr.String())
	}
}

func copyDirRecursive(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := os.Stat(srcPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyDirRecursive(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func snapshotDir(root string) (map[string]snapshotEntry, error) {
	entries := map[string]snapshotEntry{}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return entries, nil
	}
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			entries[rel] = snapshotEntry{IsDir: true}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries[rel] = snapshotEntry{Content: data}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func compareSnapshots(expected, actual map[string]snapshotEntry) string {
	keysMap := map[string]struct{}{}
	for k := range expected {
		keysMap[k] = struct{}{}
	}
	for k := range actual {
		keysMap[k] = struct{}{}
	}
	keys := make([]string, 0, len(keysMap))
	for k := range keysMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, k := range keys {
		e, eok := expected[k]
		a, aok := actual[k]
		switch {
		case !eok:
			buf.WriteString("unexpected entry: " + k + "\n")
		case !aok:
			buf.WriteString("missing entry: " + k + "\n")
		case e.IsDir != a.IsDir:
			buf.WriteString("entry type mismatch: " + k + "\n")
		case !e.IsDir && !bytes.Equal(e.Content, a.Content):
			buf.WriteString("file content mismatch: " + k + "\n")
		}
	}
	return buf.String()
}
