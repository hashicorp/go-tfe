// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type configArchiveEntry struct {
	kind byte
	body string
	link string
}

func readConfigArchive(t *testing.T, pack *PackedConfig) map[string]configArchiveEntry {
	t.Helper()
	data, err := pack.archiveBytes()
	require.NoError(t, err)
	gz, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, gz.Close()) })
	reader := tar.NewReader(gz)
	entries := make(map[string]configArchiveEntry)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		entries[header.Name] = configArchiveEntry{kind: header.Typeflag, body: string(body), link: header.Linkname}
	}
	return entries
}

func TestPackConfigIgnore(t *testing.T) {
	dir := t.TempDir()
	writeConfigFixture(t, dir, "main.tf", "test config")
	writeConfigFixture(t, dir, ".terraformignore", "ignored.tf\n")
	writeConfigFixture(t, dir, "ignored.tf", "ignored")
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o700))
	writeConfigFixture(t, dir, ".git/config", "default exclusion")

	pack, err := PackConfig(t.Context(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pack.Close()) })
	entries := readConfigArchive(t, pack)
	require.Equal(t, configArchiveEntry{kind: tar.TypeReg, body: "test config"}, entries["main.tf"])
	require.NotContains(t, entries, "ignored.tf")
	require.NotContains(t, entries, ".git/config")

	unfiltered, err := PackConfig(t.Context(), dir, WithTerraformIgnore(false))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, unfiltered.Close()) })
	entries = readConfigArchive(t, unfiltered)
	require.Contains(t, entries, "ignored.tf")
	require.Contains(t, entries, ".git/config")
}

func TestPackConfigSymlinks(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, "config")
	require.NoError(t, os.Mkdir(dir, 0o700))
	writeConfigFixture(t, repo, "shared.tf", "shared contents")
	writeConfigFixture(t, dir, "main.tf", "config contents")
	makeConfigSymlink(t, "main.tf", filepath.Join(dir, "local.tf"))
	makeConfigSymlink(t, "../shared.tf", filepath.Join(dir, "external.tf"))
	makeConfigSymlink(t, "config", filepath.Join(repo, "current"))

	pack, err := PackConfig(t.Context(), dir)
	require.Error(t, err)
	require.Nil(t, pack)

	pack, err = PackConfig(t.Context(), dir, WithExternalSymlinkDereference())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pack.Close()) })
	entries := readConfigArchive(t, pack)
	require.Equal(t, configArchiveEntry{kind: tar.TypeSymlink, link: "main.tf"}, entries["local.tf"])
	require.Equal(t, configArchiveEntry{kind: tar.TypeReg, body: "shared contents"}, entries["external.tf"])
	target, err := os.Readlink(filepath.Join(dir, "external.tf"))
	require.NoError(t, err)
	require.Equal(t, "../shared.tf", filepath.ToSlash(target))

	rootPack, err := PackConfig(t.Context(), filepath.Join(repo, "current")+string(os.PathSeparator), WithExternalSymlinkDereference())
	require.ErrorContains(t, err, "symbolic link")
	require.Nil(t, rootPack)
}

func TestPackConfigValidationAndClose(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	pack, err := PackConfig(ctx, dir)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, pack)

	pack, err = PackConfig(t.Context(), "")
	require.Error(t, err)
	require.Nil(t, pack)

	pack, err = PackConfig(t.Context(), dir)
	require.NoError(t, err)
	require.Empty(t, readConfigArchive(t, pack))
	require.NoError(t, pack.Close())
	require.NoError(t, pack.Close())
	_, err = pack.archiveBytes()
	require.Error(t, err)
}

func TestPackConfigConcurrentIgnore(t *testing.T) {
	withRules, withDefaults := t.TempDir(), t.TempDir()
	writeConfigFixture(t, withRules, ".terraformignore", "!main.tf\n")
	writeConfigFixture(t, withRules, "main.tf", "test config")
	writeConfigFixture(t, withDefaults, "main.tf", "test config")
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, dir := range []string{withRules, withDefaults} {
		workers.Go(func() {
			<-start
			pack, err := PackConfig(t.Context(), dir)
			if err != nil {
				t.Error(err)
				return
			}
			if err := pack.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	close(start)
	workers.Wait()
}

func writeConfigFixture(t *testing.T, dir, name, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600))
}

func makeConfigSymlink(t *testing.T, target, link string) {
	t.Helper()
	err := os.Symlink(filepath.FromSlash(target), link)
	if os.IsPermission(err) {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	require.NoError(t, err)
}
