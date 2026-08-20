package services_aws

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// fileLock serializes writes to the shared AWS files. Account processing runs
// concurrently, and both config and credentials are read-modify-write.
var fileLock sync.Mutex

// iniSection is one [name] block, preserving the order and the keys ark does
// not manage so foreign settings survive a rewrite.
type iniSection struct {
	Name  string
	Keys  []string
	Value map[string]string
}

func (s *iniSection) set(key, value string) {
	if _, ok := s.Value[key]; !ok {
		s.Keys = append(s.Keys, key)
	}
	s.Value[key] = value
}

// iniFile is a minimal AWS shared-config parser. It keeps unknown keys and
// section order intact; comments attached to a section are dropped.
type iniFile struct {
	order    []string
	sections map[string]*iniSection
}

func newINIFile() *iniFile {
	return &iniFile{sections: map[string]*iniSection{}}
}

func parseINI(data []byte) *iniFile {
	f := newINIFile()
	var cur *iniSection

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			cur = f.section(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		if cur == nil {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		cur.set(strings.TrimSpace(key), strings.TrimSpace(value))
	}
	return f
}

// section returns the named section, creating it if absent.
func (f *iniFile) section(name string) *iniSection {
	if s, ok := f.sections[name]; ok {
		return s
	}
	s := &iniSection{Name: name, Value: map[string]string{}}
	f.sections[name] = s
	f.order = append(f.order, name)
	return s
}

func (f *iniFile) has(name string) bool {
	_, ok := f.sections[name]
	return ok
}

func (f *iniFile) render() []byte {
	var b strings.Builder
	for _, name := range f.order {
		s := f.sections[name]
		if len(s.Keys) == 0 {
			continue
		}
		fmt.Fprintf(&b, "[%s]\n", name)
		for _, k := range s.Keys {
			fmt.Fprintf(&b, "%s = %s\n", k, s.Value[k])
		}
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// sortSections orders sections by name, keeping "default" first.
func (f *iniFile) sortSections() {
	sort.SliceStable(f.order, func(i, j int) bool {
		if f.order[i] == "default" {
			return true
		}
		if f.order[j] == "default" {
			return false
		}
		return f.order[i] < f.order[j]
	})
}

// readINI loads path, returning an empty file when it does not exist.
func readINI(path string) (*iniFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return newINIFile(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return parseINI(data), nil
}

// writeFileAtomic writes data through a temp file in the same directory and
// renames it into place, so a crash mid-write cannot truncate the original.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", dir, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to replace %s: %w", path, err)
	}
	return nil
}
