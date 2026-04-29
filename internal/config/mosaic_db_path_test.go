package config_test

import (
	"path/filepath"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/config"
)

func TestResolveMosaicDBPath_dbFlag(t *testing.T) {
	t.Parallel()
	p, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{DBFlag: " /tmp/x.hexxla "}, true)
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Clean("/tmp/x.hexxla") {
		t.Fatalf("got %q", p)
	}
}

func TestResolveMosaicDBPath_nameAndDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{
		NameFlag:  "myproj",
		DBDirFlag: dir,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "myproj.hexxla")
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}

func TestResolveMosaicDBPath_nameDefaultDir(t *testing.T) {
	t.Setenv(config.EnvMosaicDBDir, "")
	p, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{NameFlag: "foo"}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(".tmp", "foo.hexxla")
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}

func TestResolveMosaicDBPath_dbAndNameConflict(t *testing.T) {
	t.Parallel()
	_, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{DBFlag: "/a", NameFlag: "b"}, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveMosaicDBPath_envFallback(t *testing.T) {
	t.Setenv(config.EnvDBPath, filepath.Join(t.TempDir(), "env.hexxla"))
	p, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "env.hexxla" {
		t.Fatalf("got %q", p)
	}
}

func TestResolveMosaicDBPath_defaultWhenUnset(t *testing.T) {
	t.Setenv(config.EnvDBPath, "")
	p, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if p != config.MosaicDefaultRelDBFile {
		t.Fatalf("got %q", p)
	}
}

func TestResolveMosaicDBPath_mcpRequiresPath(t *testing.T) {
	t.Setenv(config.EnvDBPath, "")
	_, err := config.ResolveMosaicDBPath(config.MosaicDBPathInput{}, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateMosaicDBBaseName(t *testing.T) {
	t.Parallel()
	if err := config.ValidateMosaicDBBaseName("../x"); err == nil {
		t.Fatal("expected error for path-ish name")
	}
	if err := config.ValidateMosaicDBBaseName("ok-name_1"); err != nil {
		t.Fatal(err)
	}
}
