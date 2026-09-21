package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Code-Hex/synchro/iso8601"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &graphWidgetDataSource{}
)

type graphWidgetDataSource struct {
}

func NewGraphWidgetDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &graphWidgetDataSource{}
	}
}

func (d *graphWidgetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_graph_widget"
}

// Schema defines the schema for the data source.
func (d *graphWidgetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"height": schema.Int32Attribute{
				Description: "Height of the widget",
				Required:    true,
			},
			"annotations": schema.SingleNestedAttribute{
				Description: "Lines drawn on top of the graph. Use `horizontal` for a threshold, " +
					"`vertical` to mark a point in time, and `alarms` to render a single alarm " +
					"instead of metrics. CloudWatch does not allow `alarms` together with metrics " +
					"or with the other annotation kinds.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"alarms": schema.ListAttribute{
						Description: "The ARN of a single alarm to render. At most one, and only when " +
							"`left`, `right`, `horizontal` and `vertical` are all empty.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"horizontal": schema.ListNestedAttribute{
						Description: "Horizontal lines, typically alarm thresholds",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.Float64Attribute{
									Description: "The value on the Y axis to draw the line at",
									Required:    true,
								},
								"label": schema.StringAttribute{
									Description: "Label for the line",
									Optional:    true,
								},
								"color": schema.StringAttribute{
									Description: "The hex color code, prefixed with '#' (e.g. '#00ff00')",
									Optional:    true,
								},
								"fill": schema.StringAttribute{
									Description: "Shade the area on one side of the line. " +
										"Valid values: `above`, `below`, `none`.",
									Optional: true,
								},
								"visible": schema.BoolAttribute{
									Description: "Whether the line is shown. Defaults to true.",
									Optional:    true,
								},
								"y_axis": schema.StringAttribute{
									Description: "Which axis the value belongs to. Valid values: `left`, `right`.",
									Optional:    true,
								},
							},
						},
					},
					"vertical": schema.ListNestedAttribute{
						Description: "Vertical lines, typically marking a deploy or an incident",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Description: "The point in time to draw the line at, in ISO 8601 format",
									Required:    true,
								},
								"label": schema.StringAttribute{
									Description: "Label for the line",
									Optional:    true,
								},
								"color": schema.StringAttribute{
									Description: "The hex color code, prefixed with '#' (e.g. '#00ff00')",
									Optional:    true,
								},
								"fill": schema.StringAttribute{
									Description: "Shade the area on one side of the line. " +
										"Valid values: `before`, `after`, `none`.",
									Optional: true,
								},
								"visible": schema.BoolAttribute{
									Description: "Whether the line is shown. Defaults to true.",
									Optional:    true,
								},
							},
						},
					},
				},
			},
			"left": schema.ListAttribute{
				Description: "Metrics to display on left Y axis",
				Optional:    true,
				ElementType: types.StringType,
			},
			"left_y_axis": schema.SingleNestedAttribute{
				Description: "Settings for the left Y axis",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"label": schema.StringAttribute{
						Description: "The label",
						Optional:    true,
					},
					"max": schema.Float64Attribute{
						Description: "The maximum value",
						Optional:    true,
					},
					"min": schema.Float64Attribute{
						Description: "The minimum value",
						Optional:    true,
					},
					"show_units": schema.BoolAttribute{
						Description: "Whether to show units",
						Optional:    true,
					},
				},
			},
			"legend_position": schema.StringAttribute{
				Description: "Position of the legend",
				Optional:    true,
			},
			"live_data": schema.BoolAttribute{
				Description: "Whether the graph should show live data",
				Optional:    true,
			},
			"period": schema.Int32Attribute{
				Description: "The default period for all metrics in this widget",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region the metrics of this graph should be taken from",
				Optional:    true,
			},
			"right": schema.ListAttribute{
				Description: "Metrics to display on right Y axis",
				Optional:    true,
				ElementType: types.StringType,
			},
			"right_y_axis": schema.SingleNestedAttribute{
				Description: "Settings for the right Y axis",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"label": schema.StringAttribute{
						Description: "The label",
						Optional:    true,
					},
					"max": schema.Float64Attribute{
						Description: "The maximum value",
						Optional:    true,
					},
					"min": schema.Float64Attribute{
						Description: "The minimum value",
						Optional:    true,
					},
					"show_units": schema.BoolAttribute{
						Description: "Whether to show units",
						Optional:    true,
					},
				},
			},
			"sparkline": schema.BoolAttribute{
				Description: "Whether the graph should be shown as a sparkline",
				Optional:    true,
			},
			"stacked": schema.BoolAttribute{
				Description: "Whether the graph should be shown as stacked lines",
				Optional:    true,
			},
			"statistic": schema.StringAttribute{
				Description: "The default statistic to be displayed for each metric",
				Optional:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "The timezone to use for the widget",
				Optional:    true,
			},
			"title": schema.StringAttribute{
				Description: "Title for the graph",
				Optional:    true,
			},
			"view": schema.StringAttribute{
				Description: "Display this metric",
				Optional:    true,
			},
			"width": schema.Int32Attribute{
				Description: "Width of the widget, in a grid of 24 units wide",
				Required:    true,
			},
			"json": schema.StringAttribute{
				Description: "The settings of the widget",
				Computed:    true,
			},
		},
	}
}

type graphWidgetYAxisDataSourceModel struct {
	Label     types.String  `tfsdk:"label"`
	Max       types.Float64 `tfsdk:"max"`
	Min       types.Float64 `tfsdk:"min"`
	ShowUnits types.Bool    `tfsdk:"show_units"`
}

type graphWidgetAnnotationsDataSourceModel struct {
	Alarms     []types.String                                   `tfsdk:"alarms"`
	Horizontal []graphWidgetHorizontalAnnotationDataSourceModel `tfsdk:"horizontal"`
	Vertical   []graphWidgetVerticalAnnotationDataSourceModel   `tfsdk:"vertical"`
}

type graphWidgetHorizontalAnnotationDataSourceModel struct {
	Value   types.Float64 `tfsdk:"value"`
	Label   types.String  `tfsdk:"label"`
	Color   types.String  `tfsdk:"color"`
	Fill    types.String  `tfsdk:"fill"`
	Visible types.Bool    `tfsdk:"visible"`
	YAxis   types.String  `tfsdk:"y_axis"`
}

type graphWidgetVerticalAnnotationDataSourceModel struct {
	Value   types.String `tfsdk:"value"`
	Label   types.String `tfsdk:"label"`
	Color   types.String `tfsdk:"color"`
	Fill    types.String `tfsdk:"fill"`
	Visible types.Bool   `tfsdk:"visible"`
}

type graphWidgetDataSourceModel struct {
	Annotations    *graphWidgetAnnotationsDataSourceModel `tfsdk:"annotations"`
	Height         types.Int32                            `tfsdk:"height"`
	Left           []types.String                         `tfsdk:"left"` // JSON string containing array of metrics
	LeftYAxis      *graphWidgetYAxisDataSourceModel       `tfsdk:"left_y_axis"`
	LegendPosition types.String                           `tfsdk:"legend_position"`
	LiveData       types.Bool                             `tfsdk:"live_data"`
	Period         types.Int32                            `tfsdk:"period"`
	Region         types.String                           `tfsdk:"region"`
	Right          []types.String                         `tfsdk:"right"` // JSON string containing array of metrics
	RightYAxis     *graphWidgetYAxisDataSourceModel       `tfsdk:"right_y_axis"`
	Sparkline      types.Bool                             `tfsdk:"sparkline"`
	Stacked        types.Bool                             `tfsdk:"stacked"`
	Statistic      types.String                           `tfsdk:"statistic"`
	Timezone       types.String                           `tfsdk:"timezone"`
	Title          types.String                           `tfsdk:"title"`
	View           types.String                           `tfsdk:"view"`
	Width          types.Int32                            `tfsdk:"width"`
	Json           types.String                           `tfsdk:"json"`
}

func (d *graphWidgetDataSourceModel) Validate() error {
	// Period must be 60 or a multiple of 60
	if !d.Period.IsNull() {
		if period := d.Period.ValueInt32(); period < 60 || period%60 != 0 {
			return fmt.Errorf("period must be 60 or a multiple of 60, got: %d", period)
		}
	}

	// Validate LegendPosition
	if !d.LegendPosition.IsNull() {
		legendPos := d.LegendPosition.ValueString()
		validPositions := map[string]bool{
			"right":  true,
			"bottom": true,
			"hidden": true,
		}
		if !validPositions[legendPos] {
			return fmt.Errorf("legend_position must be one of 'right', 'bottom', or 'hidden', got: %s", legendPos)
		}
	}

	// Validate Statistic
	if !d.Statistic.IsNull() {
		stat := d.Statistic.ValueString()
		validStats := map[string]bool{
			"SampleCount": true,
			"Average":     true,
			"Sum":         true,
			"Minimum":     true,
			"Maximum":     true,
		}

		if !validStats[stat] {
			// Check if it's a percentile statistic (p??)
			if strings.HasPrefix(stat, "p") {
				percentile, err := strconv.ParseFloat(strings.TrimPrefix(stat, "p"), 64)
				if err != nil || percentile < 0 || percentile > 100 {
					return fmt.Errorf("invalid percentile statistic: %s, must be between p0 and p100", stat)
				}
			} else {
				return fmt.Errorf("statistic must be one of 'SampleCount', 'Average', 'Sum', 'Minimum', 'Maximum', or a percentile (p0-p100), got: %s", stat)
			}
		}
	}

	// Validate Timezone format if present
	if !d.Timezone.IsNull() {
		timezone := d.Timezone.ValueString()
		if timezone != "" {
			// Timezone format: [+-]HHMM
			timezonePattern := regexp.MustCompile(`^[+-][0-9]{4}$`)
			if !timezonePattern.MatchString(timezone) {
				return fmt.Errorf("invalid timezone format: %s. Must be in format +/-HHMM (e.g., +0130)", timezone)
			}

			// Validate hours (00-23) and minutes (00-59)
			hours, _ := strconv.Atoi(timezone[1:3])
			minutes, _ := strconv.Atoi(timezone[3:])
			if hours > 23 {
				return fmt.Errorf("invalid timezone hours: %02d. Must be between 00 and 23", hours)
			}
			if minutes > 59 {
				return fmt.Errorf("invalid timezone minutes: %02d. Must be between 00 and 59", minutes)
			}
		}
	}

	// Validate View. The full set of values accepted by CloudWatch is allowed: the
	// value is passed straight through to the dashboard body, so restricting it here
	// would only reject configurations that CloudWatch itself renders fine.
	if !d.View.IsNull() {
		view := d.View.ValueString()
		validViews := map[string]bool{
			"timeSeries":  true,
			"singleValue": true,
			"gauge":       true,
			"bar":         true,
			"pie":         true,
			"table":       true,
		}
		if !validViews[view] {
			return fmt.Errorf("view must be one of 'timeSeries', 'singleValue', 'gauge', 'bar', 'pie' or 'table', got: %s", view)
		}
	}

	if err := validateWidgetSize(d.Width.ValueInt32(), d.Height.ValueInt32()); err != nil {
		return err
	}

	if err := d.Annotations.Validate(len(d.Left) > 0 || len(d.Right) > 0); err != nil {
		return err
	}

	return nil
}

func (a *graphWidgetAnnotationsDataSourceModel) Validate(hasMetrics bool) error {
	if a == nil {
		return nil
	}

	if len(a.Alarms) > 0 {
		// CloudWatch renders an alarm annotation instead of metrics, not on top
		// of them: the widget carries either one alarm or a metrics array.
		if len(a.Alarms) > 1 {
			return fmt.Errorf("annotations.alarms accepts at most one alarm ARN, got: %d", len(a.Alarms))
		}
		if hasMetrics {
			return fmt.Errorf("annotations.alarms cannot be combined with left or right metrics")
		}
		if len(a.Horizontal) > 0 || len(a.Vertical) > 0 {
			return fmt.Errorf("annotations.alarms cannot be combined with horizontal or vertical annotations")
		}
		if !alarmArnPattern.MatchString(a.Alarms[0].ValueString()) {
			return fmt.Errorf("invalid alarm ARN: %s", a.Alarms[0].ValueString())
		}
	}

	validHorizontalFill := map[string]bool{"above": true, "below": true, "none": true}
	for _, h := range a.Horizontal {
		if fill := h.Fill.ValueString(); fill != "" && !validHorizontalFill[fill] {
			return fmt.Errorf("annotations.horizontal.fill must be one of 'above', 'below' or 'none', got: %s", fill)
		}
		if yAxis := h.YAxis.ValueString(); yAxis != "" && yAxis != "left" && yAxis != "right" {
			return fmt.Errorf("annotations.horizontal.y_axis must be either 'left' or 'right', got: %s", yAxis)
		}
		if color := h.Color.ValueString(); color != "" && !hexColorPattern.MatchString(color) {
			return fmt.Errorf("invalid color format: %s, must be a six-digit hex color code (e.g., #FF0000)", color)
		}
	}

	validVerticalFill := map[string]bool{"before": true, "after": true, "none": true}
	for _, v := range a.Vertical {
		if fill := v.Fill.ValueString(); fill != "" && !validVerticalFill[fill] {
			return fmt.Errorf("annotations.vertical.fill must be one of 'before', 'after' or 'none', got: %s", fill)
		}
		if color := v.Color.ValueString(); color != "" && !hexColorPattern.MatchString(color) {
			return fmt.Errorf("invalid color format: %s, must be a six-digit hex color code (e.g., #FF0000)", color)
		}
		if _, err := iso8601.ParseDateTime(v.Value.ValueString()); err != nil {
			return fmt.Errorf("annotations.vertical.value must be a valid ISO8601 date: %w", err)
		}
	}

	return nil
}

// NOTE: Value carries no omitempty and Visible is a pointer, so that a threshold
// line at 0 and an explicit `visible = false` survive this hop too. The settings
// JSON is a second place where omitempty can silently drop them.
type graphWidgetAnnotationsDataSourceSettings struct {
	Alarms     []string                                            `json:"alarms,omitempty"`
	Horizontal []graphWidgetHorizontalAnnotationDataSourceSettings `json:"horizontal,omitempty"`
	Vertical   []graphWidgetVerticalAnnotationDataSourceSettings   `json:"vertical,omitempty"`
}

type graphWidgetHorizontalAnnotationDataSourceSettings struct {
	Value   float64 `json:"value"`
	Label   string  `json:"label,omitempty"`
	Color   string  `json:"color,omitempty"`
	Fill    string  `json:"fill,omitempty"`
	Visible *bool   `json:"visible,omitempty"`
	YAxis   string  `json:"y_axis,omitempty"`
}

type graphWidgetVerticalAnnotationDataSourceSettings struct {
	Value   string `json:"value"`
	Label   string `json:"label,omitempty"`
	Color   string `json:"color,omitempty"`
	Fill    string `json:"fill,omitempty"`
	Visible *bool  `json:"visible,omitempty"`
}

func (a *graphWidgetAnnotationsDataSourceModel) toSettings() *graphWidgetAnnotationsDataSourceSettings {
	if a == nil {
		return nil
	}

	settings := &graphWidgetAnnotationsDataSourceSettings{
		Alarms: toStringSlice(a.Alarms),
	}

	for _, h := range a.Horizontal {
		horizontal := graphWidgetHorizontalAnnotationDataSourceSettings{
			Value: h.Value.ValueFloat64(),
			Label: h.Label.ValueString(),
			Color: h.Color.ValueString(),
			Fill:  h.Fill.ValueString(),
			YAxis: h.YAxis.ValueString(),
		}
		if !h.Visible.IsNull() {
			horizontal.Visible = ptrTo(h.Visible.ValueBool())
		}
		settings.Horizontal = append(settings.Horizontal, horizontal)
	}

	for _, v := range a.Vertical {
		vertical := graphWidgetVerticalAnnotationDataSourceSettings{
			Value: v.Value.ValueString(),
			Label: v.Label.ValueString(),
			Color: v.Color.ValueString(),
			Fill:  v.Fill.ValueString(),
		}
		if !v.Visible.IsNull() {
			vertical.Visible = ptrTo(v.Visible.ValueBool())
		}
		settings.Vertical = append(settings.Vertical, vertical)
	}

	return settings
}

func (s *graphWidgetAnnotationsDataSourceSettings) toCWDashboardBodyAnnotations() *CWDashboardBodyWidgetPropertyMetricAnnotations {
	if s == nil {
		return nil
	}

	annotations := &CWDashboardBodyWidgetPropertyMetricAnnotations{
		Alarms: s.Alarms,
	}

	// The settings structs and the body structs hold the same fields in the same
	// order and differ only in their JSON tags (snake_case for the intermediate
	// settings, camelCase for the dashboard body), so a conversion is enough.
	// It also fails to compile if the two ever drift apart.
	for _, h := range s.Horizontal {
		annotations.Horizontal = append(annotations.Horizontal, CWDashboardBodyWidgetPropertyMetricAnnotationsHorizontal(h))
	}

	for _, v := range s.Vertical {
		annotations.Vertical = append(annotations.Vertical, CWDashboardBodyWidgetPropertyMetricAnnotationsVertical(v))
	}

	return annotations
}

// NOTE: Max / Min / ShowUnits are pointers so that the meaningful zero values
// (`min = 0`, `max = 0`, `show_units = false`) are not dropped by `omitempty`
// on the way to the dashboard body.
type graphWidgetYAxisDataSourceSettings struct {
	Label     string   `json:"label,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	ShowUnits *bool    `json:"show_units,omitempty"`
}

func (a *graphWidgetYAxisDataSourceModel) toSettings() *graphWidgetYAxisDataSourceSettings {
	if a == nil {
		return nil
	}

	settings := &graphWidgetYAxisDataSourceSettings{
		Label: a.Label.ValueString(),
	}
	if !a.Max.IsNull() {
		settings.Max = ptrTo(a.Max.ValueFloat64())
	}
	if !a.Min.IsNull() {
		settings.Min = ptrTo(a.Min.ValueFloat64())
	}
	if !a.ShowUnits.IsNull() {
		settings.ShowUnits = ptrTo(a.ShowUnits.ValueBool())
	}

	return settings
}

func (s *graphWidgetYAxisDataSourceSettings) toCWDashboardBodyYAxisSide() *CWDashboardBodyWidgetPropertyMetricYAxisSide {
	if s == nil {
		return nil
	}

	return &CWDashboardBodyWidgetPropertyMetricYAxisSide{
		Label:     s.Label,
		Max:       s.Max,
		Min:       s.Min,
		ShowUnits: s.ShowUnits,
	}
}

const (
	typeGraphWidget = "graph"
)

type graphWidgetDataSourceSettings struct {
	Type           string                                    `json:"type"`
	Annotations    *graphWidgetAnnotationsDataSourceSettings `json:"annotations,omitempty"`
	Height         int32                                     `json:"height"`
	Left           []IMetricSettings                         `json:"left,omitempty"`
	LeftYAxis      *graphWidgetYAxisDataSourceSettings       `json:"left_y_axis,omitempty"`
	LegendPosition string                                    `json:"legend_position,omitempty"`
	LiveData       bool                                      `json:"live_data,omitempty"`
	Period         int32                                     `json:"period,omitempty"`
	Region         string                                    `json:"region,omitempty"`
	Right          []IMetricSettings                         `json:"right,omitempty"`
	RightYAxis     *graphWidgetYAxisDataSourceSettings       `json:"right_y_axis,omitempty"`
	Sparkline      bool                                      `json:"sparkline,omitempty"`
	Stacked        bool                                      `json:"stacked,omitempty"`
	Statistic      string                                    `json:"statistic,omitempty"`
	Timezone       string                                    `json:"timezone,omitempty"`
	Title          string                                    `json:"title,omitempty"`
	View           string                                    `json:"view,omitempty"`
	Width          int32                                     `json:"width"`
}

func (s *graphWidgetDataSourceSettings) UnmarshalJSON(data []byte) error {
	var intermediate struct {
		Type           string                                    `json:"type"`
		Annotations    *graphWidgetAnnotationsDataSourceSettings `json:"annotations,omitempty"`
		Height         int32                                     `json:"height"`
		LeftYAxis      *graphWidgetYAxisDataSourceSettings       `json:"left_y_axis,omitempty"`
		LegendPosition string                                    `json:"legend_position,omitempty"`
		LiveData       bool                                      `json:"live_data,omitempty"`
		Period         int32                                     `json:"period,omitempty"`
		Region         string                                    `json:"region,omitempty"`
		RightYAxis     *graphWidgetYAxisDataSourceSettings       `json:"right_y_axis,omitempty"`
		Sparkline      bool                                      `json:"sparkline,omitempty"`
		Stacked        bool                                      `json:"stacked,omitempty"`
		Statistic      string                                    `json:"statistic,omitempty"`
		Timezone       string                                    `json:"timezone,omitempty"`
		Title          string                                    `json:"title,omitempty"`
		View           string                                    `json:"view,omitempty"`
		Width          int32                                     `json:"width"`
		// Left/Right has multiple types, so we need to unmarshal them separately
		Left  []interface{} `json:"left"`
		Right []interface{} `json:"right"`
	}

	if err := json.Unmarshal(data, &intermediate); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	s.Type = intermediate.Type
	s.Annotations = intermediate.Annotations
	s.Height = intermediate.Height
	s.LeftYAxis = intermediate.LeftYAxis
	s.LegendPosition = intermediate.LegendPosition
	s.LiveData = intermediate.LiveData
	s.Period = intermediate.Period
	s.Region = intermediate.Region
	s.RightYAxis = intermediate.RightYAxis
	s.Sparkline = intermediate.Sparkline
	s.Stacked = intermediate.Stacked
	s.Statistic = intermediate.Statistic
	s.Timezone = intermediate.Timezone
	s.Title = intermediate.Title
	s.View = intermediate.View
	s.Width = intermediate.Width

	// Process left and right metrics separately
	left, err := processMetrics(intermediate.Left)
	if err != nil {
		return err
	}
	right, err := processMetrics(intermediate.Right)
	if err != nil {
		return err
	}

	s.Left = left
	s.Right = right

	return nil
}

func processMetrics(metrics []interface{}) ([]IMetricSettings, error) {
	var result []IMetricSettings

	for _, m := range metrics {
		m2, ok := m.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid metric")
		}

		t, ok := m2["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing metric type")
		}

		switch t {
		case typeNameOfMetricDataSource:
			b, err := json.Marshal(m)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metric settings: %w", err)
			}

			m := &metricDataSourceSettings{}
			if err := json.Unmarshal(b, m); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metric settings: %w", err)
			}
			result = append(result, m)

		case typeNameOfMetricExpressionDataSource:
			b, err := json.Marshal(m)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metric expression settings: %w", err)
			}

			m := &metricExpressionDataSourceSettings{}
			if err := json.Unmarshal(b, m); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metric expression settings: %w", err)
			}
			result = append(result, m)

		default:
			return nil, fmt.Errorf("unsupported metric type")
		}
	}

	return result, nil
}

func (d *graphWidgetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state graphWidgetDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("failed to validate graph widget data source", err.Error())
		return
	}

	// Parse left metrics from JSON
	leftMetrics := make([]IMetricSettings, len(state.Left))
	for i, metricJson := range state.Left {
		metric := &metricDataSourceSettings{}
		if err := json.Unmarshal([]byte(metricJson.ValueString()), &metric); err != nil {
			resp.Diagnostics.AddError("failed to unmarshal left metric", err.Error())
			return
		}

		if metric.Type != typeNameOfMetricDataSource {
			metric := &metricExpressionDataSourceSettings{}
			if err := json.Unmarshal([]byte(metricJson.ValueString()), &metric); err != nil {
				resp.Diagnostics.AddError("failed to unmarshal left metric", err.Error())
				return
			}
			leftMetrics[i] = metric
		} else {
			leftMetrics[i] = metric
		}
	}

	// Parse right metrics from JSON
	rightMetrics := make([]IMetricSettings, len(state.Right))
	for i, metricJson := range state.Right {
		var metric *metricDataSourceSettings
		if err := json.Unmarshal([]byte(metricJson.ValueString()), &metric); err != nil {
			resp.Diagnostics.AddError("failed to unmarshal right metric", err.Error())
			return
		}

		if metric.Type != typeNameOfMetricDataSource {
			metric := &metricExpressionDataSourceSettings{}
			if err := json.Unmarshal([]byte(metricJson.ValueString()), &metric); err != nil {
				resp.Diagnostics.AddError("failed to unmarshal right metric", err.Error())
				return
			}
			rightMetrics[i] = metric
		} else {
			rightMetrics[i] = metric
		}
	}

	settings := graphWidgetDataSourceSettings{
		Type:           typeGraphWidget,
		Height:         state.Height.ValueInt32(),
		Left:           leftMetrics,
		LegendPosition: state.LegendPosition.ValueString(),
		LiveData:       state.LiveData.ValueBool(),
		Period:         state.Period.ValueInt32(),
		Region:         state.Region.ValueString(),
		Right:          rightMetrics,
		Sparkline:      state.Sparkline.ValueBool(),
		Stacked:        state.Stacked.ValueBool(),
		Statistic:      state.Statistic.ValueString(),
		Timezone:       state.Timezone.ValueString(),
		Title:          state.Title.ValueString(),
		View:           state.View.ValueString(),
		Width:          state.Width.ValueInt32(),
	}

	settings.Annotations = state.Annotations.toSettings()
	settings.LeftYAxis = state.LeftYAxis.toSettings()
	settings.RightYAxis = state.RightYAxis.toSettings()

	b, err := json.Marshal(settings)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal widget settings", err.Error())
		return
	}

	tflog.Info(ctx, "graph widget settings", map[string]interface{}{
		"settings": string(b),
	})

	state.Json = types.StringValue(string(b))

	stateDiags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ToCWDashboardBodyWidget builds the widget without its position. X / Y are assigned
// later by layoutWidgets, which sees the whole widget list.
func (w graphWidgetDataSourceSettings) ToCWDashboardBodyWidget(ctx context.Context) (CWDashboardBodyWidget, error) {
	leftYAxis := w.LeftYAxis.toCWDashboardBodyYAxisSide()
	rightYAxis := w.RightYAxis.toCWDashboardBodyYAxisSide()

	var yAxis *CWDashboardBodyWidgetPropertyMetricYAxis
	if leftYAxis != nil || rightYAxis != nil {
		yAxis = &CWDashboardBodyWidgetPropertyMetricYAxis{
			Left:  leftYAxis,
			Right: rightYAxis,
		}
	}

	metrics := make([][]interface{}, 0)
	for _, metric := range w.Left {
		switch m := metric.(type) {
		case *metricDataSourceSettings:
			settings, err := m.buildMetricWidgetMetricsSettings(true, nil)
			if err != nil {
				return CWDashboardBodyWidget{}, fmt.Errorf("failed to build metric settings: %w", err)
			}
			metrics = append(metrics, settings)
		case *metricExpressionDataSourceSettings:
			settingsList, err := m.buildMetricWidgetMetricSettingsList(true)
			if err != nil {
				return CWDashboardBodyWidget{}, fmt.Errorf("failed to build metric settings: %w", err)
			}
			metrics = append(metrics, settingsList...)
		default:
			return CWDashboardBodyWidget{}, fmt.Errorf("unsupported metric type: %T", metric)

		}
	}

	for _, metric := range w.Right {
		switch m := metric.(type) {
		case *metricDataSourceSettings:
			settings, err := m.buildMetricWidgetMetricsSettings(false, nil)
			if err != nil {
				return CWDashboardBodyWidget{}, fmt.Errorf("failed to build metric settings: %w", err)
			}
			metrics = append(metrics, settings)
		case *metricExpressionDataSourceSettings:
			settingsList, err := m.buildMetricWidgetMetricSettingsList(false)
			if err != nil {
				return CWDashboardBodyWidget{}, fmt.Errorf("failed to build metric settings: %w", err)
			}
			metrics = append(metrics, settingsList...)
		default:
			return CWDashboardBodyWidget{}, fmt.Errorf("unsupported metric type: %T", metric)
		}
	}

	cwWidget := CWDashboardBodyWidget{
		Type:   "metric",
		Width:  w.Width,
		Height: w.Height,
		Properties: CWDashboardBodyWidgetPropertyMetric{
			Annotations: w.Annotations.toCWDashboardBodyAnnotations(),
			LiveData:    w.LiveData,
			Legend: &CWDashboardBodyWidgetPropertyMetricLegend{
				Position: w.LegendPosition,
			},
			Metrics:   metrics,
			Period:    w.Period,
			Region:    w.Region,
			Stat:      w.Statistic,
			Title:     w.Title,
			View:      w.View,
			Stacked:   w.Stacked,
			Sparkline: w.Sparkline,
			Timezone:  w.Timezone,
			YAxis:     yAxis,
			// NOTE: unnecessary to set because it's not used in the graph widget
			Table: nil,
		},
	}

	tflog.Debug(ctx, "built graph widget", map[string]interface{}{
		"widget": cwWidget,
	})

	return cwWidget, nil
}
