package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestDashboardDataSourceModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		model   dashboardDataSourceModel
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid model with all fields",
			model: dashboardDataSourceModel{
				Start:          types.StringValue("2024-01-01T00:00:00Z"),
				End:            types.StringValue("2024-01-02T00:00:00Z"),
				PeriodOverride: types.StringValue("auto"),
				Widgets:        types.ListValueMust(types.StringType, []attr.Value{}),
				Json:           types.StringValue("{}"),
			},
			wantErr: false,
		},
		{
			name: "valid model with minutes relative time for start",
			model: dashboardDataSourceModel{
				Start:          types.StringValue("-PT15M"),
				End:            types.StringValue("2024-01-02T00:00:00Z"),
				PeriodOverride: types.StringValue("auto"),
				Widgets:        types.ListValueMust(types.StringType, []attr.Value{}),
			},
			wantErr: false,
		},
		{
			name: "valid model with hours relative time for start",
			model: dashboardDataSourceModel{
				Start:          types.StringValue("-PT2H"),
				End:            types.StringValue("2024-01-02T00:00:00Z"),
				PeriodOverride: types.StringValue("auto"),
			},
			wantErr: false,
		},
		{
			name: "valid model with days relative time for start",
			model: dashboardDataSourceModel{
				Start:          types.StringValue("-P7D"),
				End:            types.StringValue("2024-01-02T00:00:00Z"),
				PeriodOverride: types.StringValue("auto"),
			},
			wantErr: false,
		},
		{
			name: "valid model with weeks relative time for start",
			model: dashboardDataSourceModel{
				Start:          types.StringValue("-P2W"),
				End:            types.StringValue("2024-01-02T00:00:00Z"),
				PeriodOverride: types.StringValue("auto"),
			},
			wantErr: false,
		},
		{
			name: "valid model with months relative time for start",
			model: dashboardDataSourceModel{
				Start:          types.StringValue("-P3M"),
				End:            types.StringValue("2024-01-02T00:00:00Z"),
				PeriodOverride: types.StringValue("auto"),
			},
			wantErr: false,
		},
		{
			name: "invalid relative time format for start (invalid minutes)",
			model: dashboardDataSourceModel{
				Start: types.StringValue("-PT15X"),
			},
			wantErr: true,
			errMsg:  "start must be a valid ISO8601 date or a valid relative time",
		},
		{
			name: "invalid relative time format for start (invalid days)",
			model: dashboardDataSourceModel{
				Start: types.StringValue("-P7X"),
			},
			wantErr: true,
			errMsg:  "start must be a valid ISO8601 date or a valid relative time",
		},
		{
			name: "invalid start date format",
			model: dashboardDataSourceModel{
				Start: types.StringValue("invalid-date"),
			},
			wantErr: true,
			errMsg:  "start must be a valid ISO8601 date or a valid relative time",
		},
		{
			name: "invalid end date format",
			model: dashboardDataSourceModel{
				End: types.StringValue("invalid-date"),
			},
			wantErr: true,
			errMsg:  "end must be a valid ISO8601 date",
		},
		{
			name: "invalid period override value",
			model: dashboardDataSourceModel{
				PeriodOverride: types.StringValue("invalid"),
			},
			wantErr: true,
			errMsg:  "period_override must be either 'auto' or 'inherit'",
		},
		{
			name: "too many widgets",
			model: dashboardDataSourceModel{
				Widgets: createWidgetsList(501),
			},
			wantErr: true,
			errMsg:  "maximum number of widgets is 500",
		},
		{
			name: "valid model with inherit period override",
			model: dashboardDataSourceModel{
				PeriodOverride: types.StringValue("inherit"),
			},
			wantErr: false,
		},
		{
			name:    "empty model",
			model:   dashboardDataSourceModel{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to create a list with specified number of widgets
func createWidgetsList(count int) types.List {
	elements := make([]attr.Value, count)
	for i := 0; i < count; i++ {
		elements[i] = types.StringValue("")
	}
	return types.ListValueMust(types.StringType, elements)
}

func TestBuildDashboardBodyJson(t *testing.T) {
	state := dashboardDataSourceModel{
		Start:          types.StringValue("-PT3H"),
		PeriodOverride: types.StringValue("auto"),
		Widgets:        types.ListValueMust(types.StringType, []attr.Value{}),
	}

	// A full width heading followed by two half width graphs: the layout that
	// used to place the graphs below the height of the graph rather than below
	// the heading.
	rawWidgets := []interface{}{
		textWidgetDataSourceSettings{
			Type:     typeTextWidget,
			Markdown: "# overview",
			Width:    24,
			Height:   2,
		},
		graphWidgetDataSourceSettings{
			Type:   typeGraphWidget,
			Width:  12,
			Height: 6,
			Region: "ap-northeast-1",
			Title:  "ECS",
			Left: []IMetricSettings{
				&metricDataSourceSettings{
					Type:       typeNameOfMetricDataSource,
					Namespace:  "AWS/ECS",
					MetricName: "CPUUtilization",
					DimensionsMap: map[string]string{
						"ClusterName": "shared",
						"ServiceName": "api",
					},
					Statistic: "Average",
				},
			},
		},
		graphWidgetDataSourceSettings{
			Type:   typeGraphWidget,
			Width:  12,
			Height: 6,
			Region: "us-east-1",
			Title:  "CloudFront",
			Left: []IMetricSettings{
				&metricDataSourceSettings{
					Type:       typeNameOfMetricDataSource,
					Namespace:  "AWS/CloudFront",
					MetricName: "Requests",
					DimensionsMap: map[string]string{
						"DistributionId": "E123",
						"Region":         "Global",
					},
					Region:    "us-east-1",
					Statistic: "Sum",
				},
			},
		},
	}

	body, err := buildDashboardBodyJson(context.Background(), state, rawWidgets)
	assert.NoError(t, err)

	var decoded CWDashboardBody
	assert.NoError(t, json.Unmarshal([]byte(body), &decoded))

	assert.Len(t, decoded.Widgets, 3)
	assert.Equal(t, []int32{0, 0}, []int32{decoded.Widgets[0].X, decoded.Widgets[0].Y})
	assert.Equal(t, []int32{0, 2}, []int32{decoded.Widgets[1].X, decoded.Widgets[1].Y})
	assert.Equal(t, []int32{12, 2}, []int32{decoded.Widgets[2].X, decoded.Widgets[2].Y})

	// Per-metric region is what lets a single dashboard mix regions.
	assert.Contains(t, body, `"region":"us-east-1"`)

	// The whole body must be byte for byte identical on every build, otherwise
	// aws_cloudwatch_dashboard shows a diff on every plan.
	for i := 0; i < 50; i++ {
		again, err := buildDashboardBodyJson(context.Background(), state, rawWidgets)
		assert.NoError(t, err)
		assert.Equal(t, body, again, "dashboard body must be deterministic")
	}
}
