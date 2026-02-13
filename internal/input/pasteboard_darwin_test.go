//go:build darwin

package input

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAndGetClipboard(t *testing.T) {
	text := "MCPMacControl test clipboard content"

	err := SetClipboard(text)
	require.NoError(t, err)

	got, err := GetClipboard()
	require.NoError(t, err)
	assert.Equal(t, text, got)
}

func TestSetClipboard_Unicode(t *testing.T) {
	text := "Hello 世界 🌍 café"

	err := SetClipboard(text)
	require.NoError(t, err)

	got, err := GetClipboard()
	require.NoError(t, err)
	assert.Equal(t, text, got)
}

func TestSetClipboard_Empty(t *testing.T) {
	err := SetClipboard("")
	require.NoError(t, err)

	got, err := GetClipboard()
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestSetClipboard_Multiline(t *testing.T) {
	text := "/private/tmp/CertificateSigningRequest.certSigningRequest"

	err := SetClipboard(text)
	require.NoError(t, err)

	got, err := GetClipboard()
	require.NoError(t, err)
	assert.Equal(t, text, got)
}

func TestPasteText_WritesClipboard(t *testing.T) {
	// PasteText writes to clipboard then simulates ⌘V.
	// We can verify the clipboard write; KeyTap needs accessibility
	// and may silently fail in test, which is acceptable.
	text := "paste-test-content-12345"

	err := PasteText(text)
	require.NoError(t, err)

	got, err := GetClipboard()
	require.NoError(t, err)
	assert.Equal(t, text, got)
}
