package config_test

import (
	"testing"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/config"
)

func TestBuildHexxlaOpenOptions_empty(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "")
	t.Setenv(config.EnvDBEncryptionKeyHex, "")
	o, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if o != nil {
		t.Fatalf("expected nil, got %#v", o)
	}
}

func TestBuildHexxlaOpenOptions_flagWinsOverEnv(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "from-env")
	o, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{FlagPassphrase: "from-flag"})
	if err != nil {
		t.Fatal(err)
	}
	if o == nil || o.Passphrase != "from-flag" {
		t.Fatalf("got %#v", o)
	}
}

func TestBuildHexxlaOpenOptions_envWhenNoFlag(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "secret-env")
	t.Setenv(config.EnvDBEncryptionKeyHex, "")
	o, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if o == nil || o.Passphrase != "secret-env" {
		t.Fatalf("got %#v", o)
	}
}

func TestBuildHexxlaOpenOptions_yamlWhenNoFlagOrEnv(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "")
	o, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{YAMLPassphrase: "yaml-pass"})
	if err != nil {
		t.Fatal(err)
	}
	if o == nil || o.Passphrase != "yaml-pass" {
		t.Fatalf("got %#v", o)
	}
}

func TestBuildHexxlaOpenOptions_keyHex(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "")
	t.Setenv(config.EnvDBEncryptionKeyHex, "0102030405060708090a0b0c0d0e0f10")
	o, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if o == nil || len(o.EncryptionKey) != 16 {
		t.Fatalf("got %#v", o)
	}
}

func TestBuildHexxlaOpenOptions_keyAndPassphraseConflict(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "x")
	t.Setenv(config.EnvDBEncryptionKeyHex, "0102030405060708090a0b0c0d0e0f10")
	_, err := config.BuildHexxlaOpenOptions(config.HexxlaOpenParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyHexxlaEncryption_merge(t *testing.T) {
	t.Setenv(config.EnvDBPassphrase, "")
	opts := &hexxladb.Options{EnableMVCC: true}
	err := config.ApplyHexxlaEncryption(opts, config.HexxlaOpenParams{YAMLPassphrase: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Passphrase != "p" || !opts.EnableMVCC {
		t.Fatalf("got %#v", opts)
	}
}
