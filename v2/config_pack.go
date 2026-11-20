// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-slug"
)

// PackConfigOption configures PackConfig.
type PackConfigOption func(*packConfigOptions)

type packConfigOptions struct {
	terraformIgnore     bool
	externalDereference bool
}

// go-slug v0.18.4 can mutate shared defaults while parsing ignore rules.
// Serialize ignore-enabled SDK calls until the dependency fixes that race.
var configIgnorePackSlot = make(chan struct{}, 1)

// WithTerraformIgnore controls whether PackConfig applies .terraformignore and
// go-slug's default exclusions. Both are enabled by default.
func WithTerraformIgnore(enabled bool) PackConfigOption {
	return func(o *packConfigOptions) { o.terraformIgnore = enabled }
}

// WithExternalSymlinkDereference lets PackConfig copy external symlink targets
// into the archive instead of rejecting those links. This can include files
// outside the selected directory. Use it only with trusted source trees.
func WithExternalSymlinkDereference() PackConfigOption {
	return func(o *packConfigOptions) { o.externalDereference = true }
}

// PackedConfig is a completed tar.gz archive kept in memory. The caller must
// close it when finished. Do not copy a PackedConfig.
type PackedConfig struct {
	mu   sync.Mutex
	data []byte
}

// PackConfig packs the contents of configDir without making an API request.
// The directory itself must not be a symlink. By default, .terraformignore
// rules and default exclusions apply, and external symlinks are rejected.
//
// Packing uses memory that grows with the archive and has no default size
// limit. Pack a stable, trusted source tree. Cancellation is checked before
// and after packing; it does not interrupt the underlying traversal. Calls
// with ignore handling enabled are serialized within the SDK.
func PackConfig(ctx context.Context, configDir string, opts ...PackConfigOption) (*PackedConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if configDir == "" {
		return nil, errors.New("pack config: config directory is required")
	}
	dir := filepath.Clean(configDir)
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("pack config: inspect config directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("pack config: config directory must not be a symbolic link")
	}
	if !info.IsDir() {
		return nil, errors.New("pack config: config path must be a directory")
	}

	options := packConfigOptions{terraformIgnore: true}
	for _, option := range opts {
		if option == nil {
			return nil, errors.New("pack config: invalid nil option")
		}
		option(&options)
	}
	var slugOptions []slug.PackerOption
	if options.terraformIgnore {
		slugOptions = append(slugOptions, slug.ApplyTerraformIgnore())
	}
	if options.externalDereference {
		slugOptions = append(slugOptions, slug.DereferenceSymlinks())
	}
	packer, err := slug.NewPacker(slugOptions...)
	if err != nil {
		return nil, fmt.Errorf("pack config: %w", err)
	}

	var buffer bytes.Buffer
	if options.terraformIgnore {
		select {
		case configIgnorePackSlot <- struct{}{}:
			defer func() { <-configIgnorePackSlot }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, err = packer.Pack(dir, &buffer)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, fmt.Errorf("pack config: %w", err)
	}
	return &PackedConfig{data: buffer.Bytes()}, nil
}

// Close releases the archive's storage for garbage collection and prevents
// subsequent uploads. It is safe to call Close more than once. An upload that
// already holds the archive bytes can continue until its context ends.
func (p *PackedConfig) Close() error {
	if p != nil {
		p.mu.Lock()
		p.data = nil
		p.mu.Unlock()
	}
	return nil
}

func (p *PackedConfig) archiveBytes() ([]byte, error) {
	if p == nil {
		return nil, errors.New("upload config: a packed config is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.data == nil {
		return nil, errors.New("upload config: packed config is closed or uninitialized")
	}
	return p.data, nil
}
