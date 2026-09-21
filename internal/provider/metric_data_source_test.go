package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tj/assert"
)

func TestMetricDataSourceModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		model   metricDataSourceModel
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid metric with all fields",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("Average"),
				Color:     types.StringValue("#FF0000"),
			},
			wantErr: false,
		},
		{
			name: "valid metric without optional color",
			model: metricDataSourceModel{
				Period:    types.Int32Value(300),
				Statistic: types.StringValue("Maximum"),
			},
			wantErr: false,
		},
		{
			name: "valid percentile statistic",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("p95"),
			},
			wantErr: false,
		},
		{
			name: "invalid period - less than 60",
			model: metricDataSourceModel{
				Period:    types.Int32Value(30),
				Statistic: types.StringValue("Average"),
			},
			wantErr: true,
			errMsg:  "period must be 60 or a multiple of 60, got: 30",
		},
		{
			name: "invalid period - not multiple of 60",
			model: metricDataSourceModel{
				Period:    types.Int32Value(90),
				Statistic: types.StringValue("Average"),
			},
			wantErr: true,
			errMsg:  "period must be 60 or a multiple of 60, got: 90",
		},
		{
			name: "invalid statistic",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("InvalidStat"),
			},
			wantErr: true,
			errMsg:  "invalid statistic: InvalidStat",
		},
		{
			name: "invalid percentile - negative",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("p-1"),
			},
			wantErr: true,
			errMsg:  "invalid percentile statistic: p-1, must be between p0 and p100",
		},
		{
			name: "invalid percentile - over 100",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("p101"),
			},
			wantErr: true,
			errMsg:  "invalid percentile statistic: p101, must be between p0 and p100",
		},
		{
			name: "invalid color format - missing #",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("Average"),
				Color:     types.StringValue("FF0000"),
			},
			wantErr: true,
			errMsg:  "invalid color format: FF0000, must be a six-digit hex color code (e.g., #FF0000)",
		},
		{
			name: "invalid color format - wrong length",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("Average"),
				Color:     types.StringValue("#FF00"),
			},
			wantErr: true,
			errMsg:  "invalid color format: #FF00, must be a six-digit hex color code (e.g., #FF0000)",
		},
		{
			name: "invalid color format - invalid characters",
			model: metricDataSourceModel{
				Period:    types.Int32Value(60),
				Statistic: types.StringValue("Average"),
				Color:     types.StringValue("#GG0000"),
			},
			wantErr: true,
			errMsg:  "invalid color format: #GG0000, must be a six-digit hex color code (e.g., #FF0000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestMetricDataSourceSettings_BuildMetricWidgetMetricsSettings(t *testing.T) {
	t.Run("emits dimensions in a deterministic order", func(t *testing.T) {
		s := &metricDataSourceSettings{
			Type:       typeNameOfMetricDataSource,
			Namespace:  "AWS/ECS",
			MetricName: "CPUUtilization",
			DimensionsMap: map[string]string{
				"ServiceName": "api",
				"ClusterName": "shared",
				"Env":         "production",
			},
			Statistic: "Average",
		}

		// Go randomizes map iteration, so a single run proves nothing. Anything
		// other than one distinct serialization means the dashboard body changes
		// between plans, which shows up as a permanent Terraform diff.
		seen := map[string]struct{}{}
		for i := 0; i < 200; i++ {
			got, err := s.buildMetricWidgetMetricsSettings(true, nil)
			assert.NoError(t, err)

			b, err := json.Marshal(got)
			assert.NoError(t, err)
			seen[string(b)] = struct{}{}
		}

		assert.Len(t, seen, 1, "the metrics array must serialize identically on every run")
	})

	t.Run("sorts dimensions by name", func(t *testing.T) {
		s := &metricDataSourceSettings{
			Type:       typeNameOfMetricDataSource,
			Namespace:  "AWS/ECS",
			MetricName: "CPUUtilization",
			DimensionsMap: map[string]string{
				"ServiceName": "api",
				"ClusterName": "shared",
			},
		}

		got, err := s.buildMetricWidgetMetricsSettings(true, nil)
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{
			"AWS/ECS", "CPUUtilization",
			"ClusterName", "shared",
			"ServiceName", "api",
			map[string]interface{}{"yAxis": "left"},
		}, got)
	})

	t.Run("renders region and accountId, but not unit", func(t *testing.T) {
		s := &metricDataSourceSettings{
			Type:       typeNameOfMetricDataSource,
			Namespace:  "AWS/CloudFront",
			MetricName: "Requests",
			Account:    "123456789012",
			Region:     "us-east-1",
			// The CloudWatch dashboard body has no per-metric unit field, so this
			// must not leak into the rendering properties.
			Unit: "Count",
		}

		got, err := s.buildMetricWidgetMetricsSettings(false, nil)
		assert.NoError(t, err)

		props, ok := got[len(got)-1].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "us-east-1", props["region"])
		assert.Equal(t, "123456789012", props["accountId"])
		assert.Nil(t, props["account"])
		assert.Nil(t, props["unit"])
		assert.Equal(t, "right", props["yAxis"])
	})
}
