package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hexxla/hexxladb"
)

// Passphrase and raw key material must never be logged.
const (
	// EnvDBPassphrase is an optional user passphrase for encrypted HexxlaDB files (Argon2id + per-DB salt).
	// Precedence: -db-passphrase flag > this env > policy YAML database.passphrase.
	EnvDBPassphrase = "MOSAIC_DB_PASSPHRASE" //nolint:gosec // G101: environment variable name, not a secret value
	// EnvDBEncryptionKeyHex is an optional hex-encoded raw key for Options.EncryptionKey (HKDF-stretched by HexxlaDB).
	// Mutually exclusive with passphrase sources. Example: 64 hex chars = 32 bytes.
	EnvDBEncryptionKeyHex = "MOSAIC_DB_ENCRYPTION_KEY_HEX" //nolint:gosec // G101: environment variable name, not a secret value
)

// HexxlaOpenParams holds optional credentials from the CLI, environment, and policy YAML.
// Do not log these values.
type HexxlaOpenParams struct {
	FlagPassphrase string
	YAMLPassphrase string
}

// BuildHexxlaOpenOptions returns options for [hexxladb.Open] when encryption credentials are present.
// It returns (nil, nil) when no passphrase and no key are configured (unencrypted open).
// Precedence for passphrase: FlagPassphrase, then [EnvDBPassphrase], then YAMLPassphrase.
// [EnvDBEncryptionKeyHex] overrides the passphrase path and must be used alone.
func BuildHexxlaOpenOptions(p HexxlaOpenParams) (*hexxladb.Options, error) {
	keyHex := strings.TrimSpace(os.Getenv(EnvDBEncryptionKeyHex))
	var keyBytes []byte
	if keyHex != "" {
		b, err := hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("config: %s must be hex-encoded: %w", EnvDBEncryptionKeyHex, err)
		}
		if len(b) == 0 {
			return nil, fmt.Errorf("config: %s decodes to empty key", EnvDBEncryptionKeyHex)
		}
		keyBytes = b
	}

	pass := strings.TrimSpace(p.FlagPassphrase)
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv(EnvDBPassphrase))
	}
	if pass == "" {
		pass = strings.TrimSpace(p.YAMLPassphrase)
	}

	if len(keyBytes) > 0 && pass != "" {
		return nil, errors.New("config: use either MOSAIC_DB_ENCRYPTION_KEY_HEX or a passphrase, not both")
	}
	if len(keyBytes) > 0 {
		return &hexxladb.Options{EncryptionKey: keyBytes}, nil
	}
	if pass != "" {
		return &hexxladb.Options{Passphrase: pass}, nil
	}
	return nil, nil
}

// ApplyHexxlaEncryption copies passphrase or encryption key from flag/env/yaml onto an existing
// non-nil [hexxladb.Options] (e.g. mosaic-seed MVCC + embedding options). It errors if both a raw key
// and a passphrase source are set, or if opts already has encryption fields populated.
func ApplyHexxlaEncryption(opts *hexxladb.Options, p HexxlaOpenParams) error {
	if opts == nil {
		return errors.New("config: ApplyHexxlaEncryption: nil options")
	}
	if len(opts.EncryptionKey) > 0 || opts.Passphrase != "" {
		return errors.New("config: ApplyHexxlaEncryption: options already set encryption fields")
	}
	extra, err := BuildHexxlaOpenOptions(p)
	if err != nil {
		return fmt.Errorf("config: apply hexxla encryption: %w", err)
	}
	if extra == nil {
		return nil
	}
	if len(extra.EncryptionKey) > 0 {
		opts.EncryptionKey = extra.EncryptionKey
	} else {
		opts.Passphrase = extra.Passphrase
	}
	return nil
}
