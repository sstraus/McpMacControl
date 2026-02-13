package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAppBundle_InsideApp(t *testing.T) {
	// Simulate: /path/to/MCPMacControl.app/Contents/MacOS/mcpmaccontrol
	tmpDir := t.TempDir()
	// Resolve symlinks on tmpDir itself (macOS: /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	appDir := filepath.Join(tmpDir, "MCPMacControl.app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	binaryPath := filepath.Join(appDir, "mcpmaccontrol")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake"), 0o755))

	got, err := findAppBundle(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "MCPMacControl.app"), got)
}

func TestFindAppBundle_SiblingApp(t *testing.T) {
	// Simulate: build/mcpmaccontrol alongside build/MCPMacControl.app/
	// This is the case when Claude Code launches the bare binary directly.
	tmpDir := t.TempDir()
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create sibling .app bundle
	appDir := filepath.Join(tmpDir, "MCPMacControl.app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "mcpmaccontrol"), []byte("fake"), 0o755))

	// Create bare binary in same directory
	binaryPath := filepath.Join(tmpDir, "mcpmaccontrol")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake"), 0o755))

	got, err := findAppBundle(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "MCPMacControl.app"), got)
}

func TestFindAppBundle_NoAppAnywhere(t *testing.T) {
	// Simulate: /usr/local/bin/mcpmaccontrol (no .app anywhere)
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "mcpmaccontrol")
	require.NoError(t, os.WriteFile(binaryPath, []byte("fake"), 0o755))

	_, err := findAppBundle(binaryPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .app bundle found")
}

func TestFindAppBundle_SymlinkIntoApp(t *testing.T) {
	// Binary is a symlink pointing into .app bundle
	tmpDir := t.TempDir()
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	appDir := filepath.Join(tmpDir, "MCPMacControl.app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	realBinary := filepath.Join(appDir, "mcpmaccontrol")
	require.NoError(t, os.WriteFile(realBinary, []byte("fake"), 0o755))

	symlink := filepath.Join(tmpDir, "mcpmaccontrol-link")
	require.NoError(t, os.Symlink(realBinary, symlink))

	got, err := findAppBundle(symlink)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "MCPMacControl.app"), got)
}

func TestSocketPath(t *testing.T) {
	assert.Equal(t, "/tmp/mcpmaccontrol.sock", SocketPath)
}
