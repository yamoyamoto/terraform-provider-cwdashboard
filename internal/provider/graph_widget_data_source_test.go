package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tj/assert"
)

func TestGraphWidgetDataSourceModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		model   graphWidgetDataSourceModel
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid complete model",
			model: graphWidgetDataSourceModel{
				Period:         types.Int32Value(60),
				LegendPosition: types.StringValue("right"),
				Statistic:      types.StringValue("Average"),
				Timezone:       types.StringValue("+0900"),
				View:           types.StringValue("timeSeries"),
			},
			wantErr: false,
		},
		{
			name: "valid model with minimum fields",
			model: graphWidgetDataSourceModel{
				Period: types.Int32Value(60),
			},
			wantErr: false,
		},
		{
			name: "invalid period - not allowed value",
			model: graphWidgetDataSourceModel{
				Period: types.Int32Value(45),
			},
			wantErr: true,
			errMsg:  "period must be 60 or a multiple of 60, got: 45",
		},
		{
			name: "invalid legend position",
			model: graphWidgetDataSourceModel{
				Period:         types.Int32Value(60),
				LegendPosition: types.StringValue("top"),
			},
			wantErr: true,
			errMsg:  "legend_position must be one of 'right', 'bottom', or 'hidden', got: top",
		},
		{
			name: "valid percentile statistic",
			model: graphWidgetDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("p95"),
			},
			wantErr: false,
		},
		{
			name: "invalid percentile statistic",
			model: graphWidgetDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("p101"),
			},
			wantErr: true,
			errMsg:  "invalid percentile statistic: p101, must be between p0 and p100",
		},
		{
			name: "invalid statistic",
			model: graphWidgetDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("InvalidStat"),
			},
			wantErr: true,
			errMsg:  "statistic must be one of 'SampleCount', 'Average', 'Sum', 'Minimum', 'Maximum', or a percentile (p0-p100), got: InvalidStat",
		},
		{
			name: "invalid timezone format",
			model: graphWidgetDataSourceModel{
				Period:   types.Int32Value(60),
				Timezone: types.StringValue("0900"),
			},
			wantErr: true,
			errMsg:  "invalid timezone format: 0900. Must be in format +/-HHMM (e.g., +0130)",
		},
		{
			name: "invalid timezone hours",
			model: graphWidgetDataSourceModel{
				Period:   types.Int32Value(60),
				Timezone: types.StringValue("+2500"),
			},
			wantErr: true,
			errMsg:  "invalid timezone hours: 25. Must be between 00 and 23",
		},
		{
			name: "invalid timezone minutes",
			model: graphWidgetDataSourceModel{
				Period:   types.Int32Value(60),
				Timezone: types.StringValue("+0060"),
			},
			wantErr: true,
			errMsg:  "invalid timezone minutes: 60. Must be between 00 and 59",
		},
		{
			name: "invalid view",
			model: graphWidgetDataSourceModel{
				Period: types.Int32Value(60),
				View:   types.StringValue("invalid"),
			},
			wantErr: true,
			errMsg:  "view must be one of 'timeSeries', 'singleValue', 'gauge', 'bar', 'pie' or 'table', got: invalid",
		},
		{
			name: "accepts every view CloudWatch supports",
			model: graphWidgetDataSourceModel{
				Period: types.Int32Value(60),
				View:   types.StringValue("gauge"),
			},
			wantErr: false,
		},
		{
			name: "invalid width",
			model: graphWidgetDataSourceModel{
				Period: types.Int32Value(60),
				Width:  types.Int32Value(25),
				Height: types.Int32Value(6),
			},
			wantErr: true,
			errMsg:  "width must be between 1 and 24, got: 25",
		},
		{
			name: "invalid height",
			model: graphWidgetDataSourceModel{
				Period: types.Int32Value(60),
				Width:  types.Int32Value(8),
				Height: types.Int32Value(0),
			},
			wantErr: true,
			errMsg:  "height must be between 1 and 1000, got: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// width / height are required by the schema, so every real model has
			// them. Fill in a valid size for the cases that are about something else.
			if tt.model.Width.IsNull() {
				tt.model.Width = types.Int32Value(8)
			}
			if tt.model.Height.IsNull() {
				tt.model.Height = types.Int32Value(6)
			}

			err := tt.model.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, want error")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestGraphWidgetDatasourceSettings_ToCWDashboardBodyWidget(t *testing.T) {
	t.Run("should successfully parse when all fields are specified", func(t *testing.T) {
		input := graphWidgetDataSourceSettings{
			Height: 6,
			Left: []IMetricSettings{
				&metricDataSourceSettings{
					MetricName: "CPUUtilization",
					Namespace:  "AWS/EC2",
					Account:    "123456789012",
					Color:      "#ff0000",
					DimensionsMap: map[string]string{
						"InstanceId": "i-1234567890abcdef0",
					},
					Label:     "CPU Utilization",
					Period:    300,
					Region:    "us-east-1",
					Statistic: "Average",
					Unit:      "Percent",
				},
			},
			LeftYAxis: &graphWidgetYAxisDataSourceSettings{
				Label:     "percentage",
				Min:       ptrTo(0.0),
				Max:       ptrTo(100.0),
				ShowUnits: ptrTo(true),
			},
			LegendPosition: "bottom",
			LiveData:       true,
			Period:         300,
			Region:         "us-east-1",
			Right: []IMetricSettings{
				&metricDataSourceSettings{
					MetricName: "NetworkIn",
					Namespace:  "AWS/EC2",
					Account:    "123456789012",
					Color:      "#0000ff",
					DimensionsMap: map[string]string{
						"InstanceId": "i-1234567890abcdef0",
					},
					Label:     "Network In",
					Period:    300,
					Region:    "us-east-1",
					Statistic: "Average",
					Unit:      "Bytes",
				},
			},
			RightYAxis: &graphWidgetYAxisDataSourceSettings{
				Label:     "bytes",
				Min:       ptrTo(0.0),
				ShowUnits: ptrTo(true),
			},
			Sparkline: true,
			Stacked:   true,
			Statistic: "Average",
			Timezone:  "+0000",
			Title:     "EC2 Instance Monitoring",
			View:      "timeSeries",
			Width:     12,
		}
		cwWidget, err := input.ToCWDashboardBodyWidget(context.TODO())

		assert.NoError(t, err)

		assert.Equal(t, "metric", cwWidget.Type)
		// X / Y are assigned later by layoutWidgets, not by the widget itself.
		assert.Equal(t, int32(0), cwWidget.X)
		assert.Equal(t, int32(0), cwWidget.Y)
		assert.Equal(t, int32(12), cwWidget.Width)
		assert.Equal(t, int32(6), cwWidget.Height)

		cwWidgetProperties, ok := cwWidget.Properties.(CWDashboardBodyWidgetPropertyMetric)
		assert.True(t, ok)
		assert.Equal(t, true, cwWidgetProperties.LiveData)
		assert.Equal(t, "bottom", cwWidgetProperties.Legend.Position)
		assert.Equal(t, int32(300), cwWidgetProperties.Period)
		assert.Equal(t, "us-east-1", cwWidgetProperties.Region)
		assert.Equal(t, "Average", cwWidgetProperties.Stat)
		assert.Equal(t, "EC2 Instance Monitoring", cwWidgetProperties.Title)
		assert.Equal(t, "timeSeries", cwWidgetProperties.View)
		assert.Equal(t, true, cwWidgetProperties.Stacked)
		assert.Equal(t, true, cwWidgetProperties.Sparkline)
		assert.Equal(t, "+0000", cwWidgetProperties.Timezone)
		assert.Equal(t, &CWDashboardBodyWidgetPropertyMetricYAxisSide{
			Label:     "percentage",
			Min:       ptrTo(0.0),
			Max:       ptrTo(100.0),
			ShowUnits: ptrTo(true),
		}, cwWidgetProperties.YAxis.Left)
		assert.Equal(t, &CWDashboardBodyWidgetPropertyMetricYAxisSide{
			Label:     "bytes",
			Min:       ptrTo(0.0),
			ShowUnits: ptrTo(true),
		}, cwWidgetProperties.YAxis.Right)
		assert.Nil(t, cwWidgetProperties.Table)

		// Metrics assertions
		assert.Len(t, cwWidgetProperties.Metrics, 2)
		assert.Equal(t, []interface{}{
			"AWS/EC2",
			"CPUUtilization",
			"InstanceId",
			"i-1234567890abcdef0",
			map[string]interface{}{
				"accountId": "123456789012",
				"color":     "#ff0000",
				"label":     "CPU Utilization",
				"period":    int32(300),
				"region":    "us-east-1",
				"stat":      "Average",
				"yAxis":     "left",
			},
		}, cwWidgetProperties.Metrics[0])
		assert.Equal(t, []interface{}{
			"AWS/EC2",
			"NetworkIn",
			"InstanceId",
			"i-1234567890abcdef0",
			map[string]interface{}{
				"accountId": "123456789012",
				"color":     "#0000ff",
				"label":     "Network In",
				"period":    int32(300),
				"region":    "us-east-1",
				"stat":      "Average",
				"yAxis":     "right",
			},
		}, cwWidgetProperties.Metrics[1])

	})
}

// The intermediate settings JSON is produced by the data source and re-parsed by
// the dashboard data source. UnmarshalJSON is hand written, so a field added to
// the struct but forgotten in the intermediate struct is dropped silently. This
// round trip fails whenever that happens.
func TestGraphWidgetDataSourceSettings_JSONRoundTrip(t *testing.T) {
	original := graphWidgetDataSourceSettings{
		Type: typeGraphWidget,
		Annotations: &graphWidgetAnnotationsDataSourceSettings{
			Horizontal: []graphWidgetHorizontalAnnotationDataSourceSettings{
				{Value: 0, Label: "floor", Fill: "below", Visible: ptrTo(false), YAxis: "left"},
				{Value: 45, Label: "cpu threshold"},
			},
			Vertical: []graphWidgetVerticalAnnotationDataSourceSettings{
				{Value: "2026-09-21T00:00:00Z", Label: "deploy"},
			},
		},
		Height: 6,
		Left: []IMetricSettings{
			&metricDataSourceSettings{
				Type:          typeNameOfMetricDataSource,
				MetricName:    "CPUUtilization",
				Namespace:     "AWS/ECS",
				Account:       "123456789012",
				Color:         "#ff0000",
				DimensionsMap: map[string]string{"ClusterName": "shared"},
				Label:         "CPU",
				Period:        300,
				Region:        "ap-northeast-1",
				Statistic:     "Average",
				Unit:          "Percent",
			},
		},
		LeftYAxis: &graphWidgetYAxisDataSourceSettings{
			Label:     "percentage",
			Min:       ptrTo(0.0),
			Max:       ptrTo(100.0),
			ShowUnits: ptrTo(false),
		},
		LegendPosition: "bottom",
		LiveData:       true,
		Period:         300,
		Region:         "ap-northeast-1",
		Right: []IMetricSettings{
			&metricExpressionDataSourceSettings{
				Type:       typeNameOfMetricExpressionDataSource,
				Expression: "ANOMALY_DETECTION_BAND(m1, 2)",
				Label:      "band",
				Period:     300,
			},
		},
		RightYAxis: &graphWidgetYAxisDataSourceSettings{
			Label:     "bytes",
			ShowUnits: ptrTo(true),
		},
		Sparkline: true,
		Stacked:   true,
		Statistic: "Average",
		Timezone:  "+0900",
		Title:     "round trip",
		View:      "timeSeries",
		Width:     12,
	}

	encoded, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded graphWidgetDataSourceSettings
	assert.NoError(t, json.Unmarshal(encoded, &decoded))

	reEncoded, err := json.Marshal(decoded)
	assert.NoError(t, err)

	assert.JSONEq(t, string(encoded), string(reEncoded))
}

func TestGraphWidgetDataSourceSettings_YAxisZeroValuesSurvive(t *testing.T) {
	w := graphWidgetDataSourceSettings{
		Type:   typeGraphWidget,
		Width:  12,
		Height: 6,
		Region: "ap-northeast-1",
		LeftYAxis: &graphWidgetYAxisDataSourceSettings{
			// Both of these are meaningful zero values and used to be dropped by
			// omitempty, silently ignoring the configuration.
			Min:       ptrTo(0.0),
			ShowUnits: ptrTo(false),
		},
	}

	cwWidget, err := w.ToCWDashboardBodyWidget(context.TODO())
	assert.NoError(t, err)

	encoded, err := json.Marshal(cwWidget)
	assert.NoError(t, err)

	assert.Contains(t, string(encoded), `"min":0`)
	assert.Contains(t, string(encoded), `"showUnits":false`)
}

func TestGraphWidgetAnnotations_ZeroValuesSurvive(t *testing.T) {
	w := graphWidgetDataSourceSettings{
		Type:   typeGraphWidget,
		Width:  12,
		Height: 6,
		Region: "ap-northeast-1",
		Annotations: &graphWidgetAnnotationsDataSourceSettings{
			Horizontal: []graphWidgetHorizontalAnnotationDataSourceSettings{
				// A threshold of 0 and an explicit "hide this line" are both
				// meaningful, and both look like empty values to omitempty.
				{Value: 0, Visible: ptrTo(false)},
			},
			Vertical: []graphWidgetVerticalAnnotationDataSourceSettings{
				{Value: "2026-09-21T00:00:00Z", Visible: ptrTo(false)},
			},
		},
	}

	cwWidget, err := w.ToCWDashboardBodyWidget(context.TODO())
	assert.NoError(t, err)

	encoded, err := json.Marshal(cwWidget)
	assert.NoError(t, err)

	assert.Contains(t, string(encoded), `"value":0`)
	assert.Contains(t, string(encoded), `"visible":false`)
	assert.Contains(t, string(encoded), `"annotations"`)
}

func TestGraphWidgetAnnotations_OmittedWhenUnset(t *testing.T) {
	w := graphWidgetDataSourceSettings{
		Type: typeGraphWidget, Width: 12, Height: 6, Region: "ap-northeast-1",
	}

	cwWidget, err := w.ToCWDashboardBodyWidget(context.TODO())
	assert.NoError(t, err)

	encoded, err := json.Marshal(cwWidget)
	assert.NoError(t, err)

	assert.NotContains(t, string(encoded), `"annotations"`)
}

func TestGraphWidgetAnnotationsDataSourceModel_Validate(t *testing.T) {
	alarmArn := types.StringValue("arn:aws:cloudwatch:ap-northeast-1:123456789012:alarm:api-cpu-high")

	tests := []struct {
		name        string
		annotations *graphWidgetAnnotationsDataSourceModel
		hasMetrics  bool
		wantErr     string
	}{
		{
			name: "valid horizontal and vertical annotations",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Horizontal: []graphWidgetHorizontalAnnotationDataSourceModel{
					{Value: types.Float64Value(45), Fill: types.StringValue("above"), YAxis: types.StringValue("left"), Color: types.StringValue("#FF0000")},
				},
				Vertical: []graphWidgetVerticalAnnotationDataSourceModel{
					{Value: types.StringValue("2026-09-21T00:00:00Z"), Fill: types.StringValue("before")},
				},
			},
			hasMetrics: true,
		},
		{
			name:        "valid alarm annotation on a widget without metrics",
			annotations: &graphWidgetAnnotationsDataSourceModel{Alarms: []types.String{alarmArn}},
		},
		{
			name:        "rejects more than one alarm",
			annotations: &graphWidgetAnnotationsDataSourceModel{Alarms: []types.String{alarmArn, alarmArn}},
			wantErr:     "annotations.alarms accepts at most one alarm ARN, got: 2",
		},
		{
			name:        "rejects an alarm annotation alongside metrics",
			annotations: &graphWidgetAnnotationsDataSourceModel{Alarms: []types.String{alarmArn}},
			hasMetrics:  true,
			wantErr:     "annotations.alarms cannot be combined with left or right metrics",
		},
		{
			name: "rejects an alarm annotation alongside a threshold line",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Alarms:     []types.String{alarmArn},
				Horizontal: []graphWidgetHorizontalAnnotationDataSourceModel{{Value: types.Float64Value(1)}},
			},
			wantErr: "annotations.alarms cannot be combined with horizontal or vertical annotations",
		},
		{
			name:        "rejects an alarm name that is not an ARN",
			annotations: &graphWidgetAnnotationsDataSourceModel{Alarms: []types.String{types.StringValue("api-cpu-high")}},
			wantErr:     "invalid alarm ARN: api-cpu-high",
		},
		{
			name: "rejects an unknown horizontal fill",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Horizontal: []graphWidgetHorizontalAnnotationDataSourceModel{{Value: types.Float64Value(1), Fill: types.StringValue("before")}},
			},
			wantErr: "annotations.horizontal.fill must be one of 'above', 'below' or 'none', got: before",
		},
		{
			name: "rejects an unknown y_axis",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Horizontal: []graphWidgetHorizontalAnnotationDataSourceModel{{Value: types.Float64Value(1), YAxis: types.StringValue("middle")}},
			},
			wantErr: "annotations.horizontal.y_axis must be either 'left' or 'right', got: middle",
		},
		{
			name: "rejects a malformed color",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Horizontal: []graphWidgetHorizontalAnnotationDataSourceModel{{Value: types.Float64Value(1), Color: types.StringValue("red")}},
			},
			wantErr: "invalid color format: red, must be a six-digit hex color code (e.g., #FF0000)",
		},
		{
			name: "rejects an unknown vertical fill",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Vertical: []graphWidgetVerticalAnnotationDataSourceModel{{Value: types.StringValue("2026-09-21T00:00:00Z"), Fill: types.StringValue("above")}},
			},
			wantErr: "annotations.vertical.fill must be one of 'before', 'after' or 'none', got: above",
		},
		{
			name: "rejects a vertical value that is not a timestamp",
			annotations: &graphWidgetAnnotationsDataSourceModel{
				Vertical: []graphWidgetVerticalAnnotationDataSourceModel{{Value: types.StringValue("yesterday")}},
			},
			wantErr: "annotations.vertical.value must be a valid ISO8601 date",
		},
		{
			name:        "accepts no annotations at all",
			annotations: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.annotations.Validate(tc.hasMetrics)
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
