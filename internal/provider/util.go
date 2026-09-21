package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	MAX_WIDTH = 24

	// Limits from the CloudWatch dashboard body structure.
	maxWidgetHeight = 1000
)

// ptrTo returns a pointer to v. It is used for dashboard body fields whose zero
// value is meaningful (`min = 0`, `showUnits = false`, ...) and therefore cannot
// be represented with a plain value plus `omitempty`.
func ptrTo[T any](v T) *T {
	return &v
}

// validateWidgetSize checks a widget's width and height against the dashboard grid.
// A widget wider than the grid can never be laid out correctly, so it is rejected
// rather than silently placed on a row of its own.
func validateWidgetSize(width, height int32) error {
	if width < 1 || width > MAX_WIDTH {
		return fmt.Errorf("width must be between 1 and %d, got: %d", MAX_WIDTH, width)
	}
	if height < 1 || height > maxWidgetHeight {
		return fmt.Errorf("height must be between 1 and %d, got: %d", maxWidgetHeight, height)
	}

	return nil
}

// layoutWidgets assigns X/Y coordinates to widgets, packing them from left to
// right and wrapping to a new row when the next widget no longer fits in the
// 24 column grid.
//
// The top of a new row is the bottom of the tallest widget in the row above it.
// Tracking the row height (rather than the height of the widget that triggered
// the wrap) is what keeps widgets of differing heights from overlapping.
func layoutWidgets(widgets []CWDashboardBodyWidget) {
	var x, y, rowHeight int32

	for i := range widgets {
		w := &widgets[i]

		// The `x > 0` guard keeps a widget wider than the grid from wrapping
		// forever: it is placed on its own row instead.
		if x > 0 && x+w.Width > MAX_WIDTH {
			x = 0
			y += rowHeight
			rowHeight = 0
		}

		w.X = x
		w.Y = y

		x += w.Width
		if w.Height > rowHeight {
			rowHeight = w.Height
		}
	}
}

// toStringSlice converts a list attribute's values to plain strings, dropping the
// slice entirely when it is empty so that `omitempty` can leave the field out.
func toStringSlice(values []types.String) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.ValueString())
	}

	return out
}
