package configfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validJSON(content []byte) ([]string, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(content)), "{") {
		return nil, errors.New("not an object")
	}
	return []string{"a warning"}, nil
}

func newTestStore(t *testing.T) (store *Store, path string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "config.json")
	store = New(path, validJSON)
	store.fromEnvironment = func() bool { return false }
	return store, path
}

func Test_Store_Save_newFile(t *testing.T) {
	t.Parallel()
	store, path := newTestStore(t)

	warnings, err := store.Save([]byte(`{"settings":[]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 1 || warnings[0] != "a warning" {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	b, _ := os.ReadFile(path)
	if string(b) != `{"settings":[]}` {
		t.Errorf("unexpected content: %q", b)
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no backup expected for a new file, got err %v", err)
	}
}

func Test_Store_Save_backupAndMode(t *testing.T) {
	t.Parallel()
	store, path := newTestStore(t)
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0o640); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Save([]byte(`{"new":true}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, _ := os.ReadFile(path)
	if string(b) != `{"new":true}` {
		t.Errorf("unexpected content: %q", b)
	}
	bak, _ := os.ReadFile(path + ".bak")
	if string(bak) != `{"old":true}` {
		t.Errorf("unexpected backup: %q", bak)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o640 {
		t.Errorf("file mode changed to %v", info.Mode().Perm())
	}
	bakInfo, _ := os.Stat(path + ".bak")
	if bakInfo.Mode().Perm() != 0o600 {
		t.Errorf("backup mode is %v instead of 0600", bakInfo.Mode().Perm())
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".config-*.tmp"))
	if len(leftovers) != 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
}

func Test_Store_Save_invalidKeepsFile(t *testing.T) {
	t.Parallel()
	store, path := newTestStore(t)
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := store.Save([]byte(`nope`))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != `{"old":true}` {
		t.Errorf("file was modified: %q", b)
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no backup expected when validation fails, got err %v", err)
	}
}

func Test_Store_Save_fromEnvironment(t *testing.T) {
	t.Parallel()
	store, path := newTestStore(t)
	store.fromEnvironment = func() bool { return true }

	_, err := store.Save([]byte(`{}`))
	if !errors.Is(err, ErrFromEnvironment) {
		t.Fatalf("expected ErrFromEnvironment, got %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file must not be created, got err %v", err)
	}
}

func Test_Store_Save_tooLarge(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t)

	big := []byte("{" + strings.Repeat(" ", MaxSize) + "}")
	_, err := store.Save(big)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func Test_Store_Save_readOnlyDirFallsBackToInPlace(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("read only directories do not restrict root")
	}
	store, path := newTestStore(t)
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(path)
	// The backup needs the directory, so only test the writer directly.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if err := writeFile(path, []byte(`{"new":true}`), 0o600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != `{"new":true}` {
		t.Errorf("unexpected content: %q", b)
	}
	_ = store
}

func Test_Store_Read(t *testing.T) {
	t.Parallel()
	store, path := newTestStore(t)
	if _, err := store.Read(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := store.Read()
	if err != nil || string(b) != `{"a":1}` {
		t.Fatalf("unexpected result %q, %v", b, err)
	}
}
