package gomtui

import (
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// ChangeTypeRGBColor returns the RGBColor for a given ChangeType.
// Maps gitutils.ChangeType to gomtui.RGBColor.
func ChangeTypeRGBColor(ct gitutils.ChangeType) RGBColor {
	switch ct {
	case gitutils.ModifiedChangeType:
		return YellowColor // Yellow - modified
	case gitutils.AddedChangeType:
		return GreenColor // Green - added
	case gitutils.DeletedChangeType:
		return RedColor // Red - deleted
	case gitutils.RenamedChangeType:
		return CyanColor // Cyan - renamed
	case gitutils.CopiedChangeType:
		return CyanColor // Cyan - copied
	case gitutils.UntrackedChangeType:
		return SilverColor // Silver - untracked/new
	default:
		return GrayColor // Gray - unknown
	}
}

// ChangeTypeColor returns the color string for a given ChangeType.
// Convenience wrapper around ChangeTypeRGBColor.
func ChangeTypeColor(ct gitutils.ChangeType) string {
	return string(ChangeTypeRGBColor(ct))
}

// StagingRGBColor returns the RGBColor for a given Staging state.
// Maps gitutils.Staging to gomtui.RGBColor.
func StagingRGBColor(s gitutils.Staging) RGBColor {
	switch s {
	case gitutils.IndexStaging:
		return GreenColor // Staged - ready to commit
	case gitutils.WorktreeStaging:
		return YellowColor // Unstaged - modified but not added
	case gitutils.BothStaging:
		return CyanColor // Both - staged and then modified again
	case gitutils.NoneStaging:
		return GrayColor // None - no changes
	default:
		return WhiteColor // Unknown
	}
}

// StagingColor returns the color string for a given Staging state.
// Convenience wrapper around StagingRGBColor.
func StagingColor(s gitutils.Staging) string {
	return string(StagingRGBColor(s))
}

// renderEnumValue renders a text value with a color using the existing renderRGBColor helper.
// Helper function for rendering colored enum values in tables.
func renderEnumValue(text string, color RGBColor) string {
	return renderRGBColor(text, color)
}

// renderChangeType renders a ChangeType with its associated color.
func renderChangeType(ct gitutils.ChangeType) string {
	return renderEnumValue(ct.Label(), ChangeTypeRGBColor(ct))
}

// renderStaging renders a Staging state with its associated color.
func renderStaging(s gitutils.Staging) string {
	return renderEnumValue(s.Label(), StagingRGBColor(s))
}

// renderDisposition renders a FileDisposition with its associated color.
func renderDisposition(disp FileDisposition) string {
	return renderEnumValue(disp.Key(), disp.RGBColor())
}
