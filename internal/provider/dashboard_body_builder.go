package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

/*
	variables, metrics explorer not yet supported
*/

// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/CloudWatch-Dashboard-Body-Structure.html
type CWDashboardBody struct {
	Widgets []CWDashboardBodyWidget `json:"widgets"`
	// Variables []CWDashboardBodyVariable `json:"variables"`
	Start          string `json:"start,omitempty"`
	End            string `json:"end,omitempty"`
	PeriodOverride string `json:"periodOverride,omitempty"`
}

type CWDashboardBodyWidget struct {
	Type       string      `json:"type"`
	X          int32       `json:"x"`
	Y          int32       `json:"y"`
	Width      int32       `json:"width"`
	Height     int32       `json:"height"`
	Properties interface{} `json:"properties"`
}

type CWDashboardBodyWidgetPropertyText struct {
	Markdown   string `json:"markdown"`
	Background string `json:"background,omitempty"`
}

type CWDashboardBodyWidgetPropertyMetric struct {
	// NOTE: Widget level settings are not supported yet
	// AccountId string `json:"accountId,omitempty"`
	// NOTE: annotations are not supported yet
	// Annotations *CWDashboardBodyWidgetPropertyMetricAnnotations `json:"annotations,omitempty"`
	LiveData  bool                                       `json:"liveData,omitempty"`
	Legend    *CWDashboardBodyWidgetPropertyMetricLegend `json:"legend,omitempty"`
	Metrics   [][]interface{}                            `json:"metrics"`
	Period    int32                                      `json:"period,omitempty"`
	Region    string                                     `json:"region"`
	Sparkline bool                                       `json:"sparkline,omitempty"`
	Stacked   bool                                       `json:"stacked,omitempty"`
	Stat      string                                     `json:"stat,omitempty"`
	Table     *CWDashboardBodyWidgetPropertyMetricTable  `json:"table,omitempty"`
	Timezone  string                                     `json:"timezone,omitempty"`
	Title     string                                     `json:"title,omitempty"`
	View      string                                     `json:"view,omitempty"`
	YAxis     *CWDashboardBodyWidgetPropertyMetricYAxis  `json:"yAxis,omitempty"`
}

type CWDashboardBodyWidgetPropertyMetricAnnotations struct {
	Alarms     []string                                                   `json:"alarms,omitempty"`
	Horizontal []CWDashboardBodyWidgetPropertyMetricAnnotationsHorizontal `json:"horizontal,omitempty"`
	Vertical   []CWDashboardBodyWidgetPropertyMetricAnnotationsVertical   `json:"vertical,omitempty"`
}

type CWDashboardBodyWidgetPropertyMetricAnnotationsHorizontal struct {
	Value   float64 `json:"value"`
	Label   string  `json:"label,omitempty"`
	Color   string  `json:"color,omitempty"`
	Fill    string  `json:"fill,omitempty"`
	Visible bool    `json:"visible,omitempty"`
	YAxis   string  `json:"yAxis,omitempty"`
}

type CWDashboardBodyWidgetPropertyMetricAnnotationsVertical struct {
	Value   string `json:"value"`
	Label   string `json:"label,omitempty"`
	Color   string `json:"color,omitempty"`
	Fill    string `json:"fill,omitempty"`
	Visible bool   `json:"visible,omitempty"`
}

type CWDashboardBodyWidgetPropertyMetricLegend struct {
	Position string `json:"position"`
}

type CWDashboardBodyWidgetPropertyMetricYAxis struct {
	Left  *CWDashboardBodyWidgetPropertyMetricYAxisSide `json:"left,omitempty"`
	Right *CWDashboardBodyWidgetPropertyMetricYAxisSide `json:"right,omitempty"`
}

// NOTE: Min / Max / ShowUnits are pointers so that the meaningful zero values
// (`min = 0`, `max = 0`, `showUnits = false`) survive `omitempty`.
type CWDashboardBodyWidgetPropertyMetricYAxisSide struct {
	Label     string   `json:"label,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	ShowUnits *bool    `json:"showUnits,omitempty"`
}

type CWDashboardBodyWidgetPropertyMetricTable struct {
	Layout             string   `json:"layout,omitempty"`
	StickySummary      bool     `json:"stickySummary,omitempty"`
	ShowTimeSeriesData bool     `json:"showTimeSeriesData,omitempty"`
	SummaryColumns     []string `json:"summaryColumns,omitempty"`
}

type CWDashboardBodyWidgetPropertyLog struct {
	AccountId string `json:"accountId,omitempty"`
	Region    string `json:"region"`
	Title     string `json:"title,omitempty"`
	Query     string `json:"query"`
	View      string `json:"view,omitempty"`
}

type CWDashboardBodyWidgetPropertyExplorer struct {
	// AggregateBy   *CWDashboardBodyWidgetPropertyExplorerAggregateBy `json:"aggregateBy,omitempty"`
	// Labels        []CWDashboardBodyWidgetPropertyExplorerLabel      `json:"labels"`
	// Metrics       []CWDashboardBodyWidgetPropertyExplorerMetric     `json:"metrics"`
	Period  int32  `json:"period,omitempty"`
	SplitBy string `json:"splitBy,omitempty"`
	Title   string `json:"title,omitempty"`
	// WidgetOptions *CWDashboardBodyWidgetPropertyExplorerOptions     `json:"widgetOptions,omitempty"`
}

// type CWDashboardBodyWidgetPropertyExplorerAggregateBy struct {
// 	Key  string `json:"key"`
// 	Func string `json:"func"`
// }

// type CWDashboardBodyWidgetPropertyExplorerLabel struct {
// 	Key   string `json:"key"`
// 	Value string `json:"value,omitempty"`
// }

// type CWDashboardBodyWidgetPropertyExplorerMetric struct {
// 	MetricName   string `json:"metricName"`
// 	ResourceType string `json:"resourceType"`
// 	Stat         string `json:"stat"`
// }

// type CWDashboardBodyWidgetPropertyExplorerOptions struct {
// 	Legend        *CWDashboardBodyWidgetPropertyMetricLegend `json:"legend,omitempty"`
// 	RowsPerPage   int                                        `json:"rowsPerPage,omitempty"`
// 	Stacked       bool                                       `json:"stacked,omitempty"`
// 	View          string                                     `json:"view,omitempty"`
// 	WidgetsPerRow int                                        `json:"widgetsPerRow,omitempty"`
// }

// Alarm widget properties are planned for future implementation
// type CWDashboardBodyWidgetPropertyAlarm struct {
//     Alarms  []string `json:"alarms"`
//     SortBy  string   `json:"sortBy,omitempty"`
//     States  []string `json:"states,omitempty"`
//     Title   string   `json:"title,omitempty"`
// }

func buildDashboardBodyJson(ctx context.Context, state dashboardDataSourceModel, rawWidgets []interface{}) (string, error) {
	widgets := make([]CWDashboardBodyWidget, 0)
	for _, rawWidget := range rawWidgets {
		switch w := rawWidget.(type) {
		case textWidgetDataSourceSettings:
			widget, err := w.ToCWDashboardBodyWidget(ctx)
			if err != nil {
				return "", fmt.Errorf("failed to parse text widget: %w", err)
			}
			widgets = append(widgets, widget)
		case graphWidgetDataSourceSettings:
			widget, err := w.ToCWDashboardBodyWidget(ctx)
			if err != nil {
				return "", fmt.Errorf("failed to parse graph widget: %w", err)
			}
			widgets = append(widgets, widget)
		default:
			return "", fmt.Errorf("unsupported widget type")
		}
	}

	// Positions are assigned in a single pass once every widget is built, so that
	// each row can be laid out with the height of its tallest widget.
	layoutWidgets(widgets)

	body := CWDashboardBody{
		Widgets:        widgets,
		Start:          state.Start.ValueString(),
		End:            state.End.ValueString(),
		PeriodOverride: state.PeriodOverride.ValueString(),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal dashboard body: %w", err)
	}

	return string(bodyBytes), nil
}
