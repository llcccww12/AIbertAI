package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var cloudReleaseVersion = regexp.MustCompile(`^[0-9]{1,8}\.[0-9]{1,8}\.[0-9]{1,8}$`)

func prepareCloudMigration(args []string, out io.Writer) error {
	if runtime.GOOS != "linux" {
		return errors.New("cloud_runtime_requires_linux")
	}
	flags := flag.NewFlagSet("prepare-cloud-migration", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("state-dir", "", "existing stopped cloud runtime state")
	keyFile := flags.String("key-file", "", "protected original Ed25519 PKCS#8 PEM file or raw 32-byte seed")
	publicKey := flags.String("public-key", "", "original registered device public key")
	environment := flags.String("environment", "", "existing cloud environment ID")
	generation := flags.Int64("generation", 0, "confirmed stopped generation")
	release := flags.String("version", "", "exact source runtime version")
	var identity cloudIdentity
	flags.StringVar(&identity.Server, "server", "", "private backend origin")
	flags.StringVar(&identity.ServerID, "server-id", "", "private backend identity")
	flags.StringVar(&identity.UserID, "user", "", "cloud environment owner")
	flags.StringVar(&identity.MachineID, "machine", "", "original cloud device ID")
	flags.StringVar(&identity.Name, "name", "AIbertAI Cloud", "existing device display name")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return errors.New("cloud_migration_arguments_invalid")
	}
	return stageCloudMigration(*dir, *keyFile, *publicKey, *environment, *generation, *release, identity, out)
}

func stageCloudMigration(dir, keyFile, publicKey, environment string, generation int64, release string, identity cloudIdentity, out io.Writer) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || !filepath.IsAbs(keyFile) || !validUUID(environment) ||
		!validUUID(identity.ServerID) || !validUUID(identity.UserID) || !validUUID(identity.MachineID) ||
		generation < 1 || !cloudReleaseVersion.MatchString(release) || strings.TrimSpace(identity.Name) == "" || len(identity.Name) > 128 {
		return errors.New("cloud_migration_arguments_invalid")
	}
	c, err := newBackendClient(identity.Server, true)
	if err != nil {
		return err
	}
	identity.Server = c.server
	expected, err := base64.RawURLEncoding.Strict().DecodeString(publicKey)
	if err != nil || len(expected) != ed25519.PublicKeySize {
		return errors.New("cloud_migration_public_key_invalid")
	}
	private, err := readCloudMigrationKey(keyFile)
	if err != nil {
		return err
	}
	defer clear(private)
	if !bytes.Equal(private.Public().(ed25519.PublicKey), expected) {
		return errors.New("cloud_migration_key_mismatch")
	}
	seed := private.Seed()
	defer clear(seed)
	identity.DataKey = base64.StdEncoding.EncodeToString(seed)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	install := func() error {
		current, err := readCloudIdentity(root)
		if err == nil {
			if current != identity {
				return errors.New("cloud_identity_already_bound")
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		// Credentials from a different source runtime must not be adopted.
		if _, err := root.Lstat(sourceGrantFilename); !errors.Is(err, os.ErrNotExist) {
			return errors.New("cloud_identity_already_bound")
		}
		encoded, err := json.Marshal(identity)
		if err != nil {
			return err
		}
		defer clear(encoded)
		return persistSourceState(root, cloudIdentityFile, encoded, true)
	}
	if err := install(); err != nil {
		return err
	}
	// The proof authorizes one exact upgrade and carries no reusable credential.
	payload := fmt.Sprintf("aibertai.cloud-runtime-migration.v1\n%s\n%d\n%s\n%s\n%s\n%s",
		environment, generation, release, identity.ServerID, identity.UserID, identity.MachineID)
	key, err := newUUID()
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(map[string]any{
		"generation": generation, "version": release, "request_key": key,
		"identity_proof": base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(payload))),
	})
}

func readCloudMigrationKey(path string) (ed25519.PrivateKey, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 4096 {
		return nil, errors.New("cloud_migration_key_file_invalid")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cloud_migration_key_file_invalid")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("cloud_migration_key_file_invalid")
	}
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil || len(data) > 4096 {
		return nil, errors.New("cloud_migration_key_file_invalid")
	}
	defer clear(data)
	if len(data) == ed25519.SeedSize {
		return ed25519.NewKeyFromSeed(data), nil
	}
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("cloud_migration_key_format_invalid")
	}
	defer clear(block.Bytes)
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	key, ok := parsed.(ed25519.PrivateKey)
	if err != nil || !ok {
		return nil, errors.New("cloud_migration_key_format_invalid")
	}
	return key, nil
}
