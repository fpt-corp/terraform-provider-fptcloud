package fptcloud_tagging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validateColor(t *testing.T, color string) bool {
	t.Helper()
	_, errs := ResourceTagging().Schema["color"].ValidateFunc(color, "color")
	return len(errs) == 0
}

// The platform accepts the same twelve colors the portal offers.
func TestResourceTagging_ColorAcceptsThePalette(t *testing.T) {
	assert.Len(t, AllowedColors, 12)
	for _, color := range AllowedColors {
		assert.True(t, validateColor(t, color), "color %s must be accepted", color)
	}
}

func TestResourceTagging_ColorIsCaseInsensitive(t *testing.T) {
	assert.True(t, validateColor(t, "#5850EC"))
	assert.True(t, validateColor(t, "#DB4646"))
}

func TestResourceTagging_ColorCanBeEmpty(t *testing.T) {
	assert.True(t, validateColor(t, ""))
}

// Names and colors outside the palette are rejected at plan time instead of
// failing the apply with a generic "Data does not match the required format".
func TestResourceTagging_ColorRejectsValuesOutsideThePalette(t *testing.T) {
	for _, color := range []string{"red", "blue", "#ff0000", "5850ec"} {
		assert.False(t, validateColor(t, color), "color %s must be rejected", color)
	}
}

// The API returns the color in lowercase, so an uppercase config must not
// show a diff on every plan.
func TestResourceTagging_ColorDiffIgnoresCase(t *testing.T) {
	suppress := ResourceTagging().Schema["color"].DiffSuppressFunc
	assert.True(t, suppress("color", "#5850ec", "#5850EC", nil))
	assert.False(t, suppress("color", "#5850ec", "#6c2bd9", nil))
}
