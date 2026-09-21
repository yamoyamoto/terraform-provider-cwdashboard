package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tj/assert"
)

func TestLogWidgetDataSourceModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		model   logWidgetDataSourceModel
		wantErr string
	}{
		{
			name: "valid minimum model",
			model: logWidgetDataSourceModel{
				Query:  types.StringValue("fields @timestamp, @message"),
				Region: types.StringValue("ap-northeast-1"),
			},
		},
		{
			name: "valid model with log group names",
			model: logWidgetDataSourceModel{
				Query:         types.StringValue("fields @timestamp, @message"),
				LogGroupNames: []types.String{types.StringValue("/aws/ecs/api")},
				Region:        types.StringValue("ap-northeast-1"),
				View:          types.StringValue("table"),
			},
		},
		{
			name: "accepts a hand written SOURCE clause when no log group names are given",
			model: logWidgetDataSourceModel{
				Query:  types.StringValue("SOURCE '/aws/ecs/api' | fields @message"),
				Region: types.StringValue("ap-northeast-1"),
			},
		},
		{
			name: "rejects an empty query",
			model: logWidgetDataSourceModel{
				Query:  types.StringValue("   "),
				Region: types.StringValue("ap-northeast-1"),
			},
			wantErr: "query must not be empty",
		},
		{
			name: "rejects SOURCE specified twice",
			model: logWidgetDataSourceModel{
				Query:         types.StringValue("SOURCE '/aws/ecs/api' | fields @message"),
				LogGroupNames: []types.String{types.StringValue("/aws/ecs/api")},
				Region:        types.StringValue("ap-northeast-1"),
			},
			wantErr: "query must not start with SOURCE when log_group_names is set: the SOURCE clauses are generated from log_group_names",
		},
		{
			name: "rejects a log group name containing a quote",
			model: logWidgetDataSourceModel{
				Query:         types.StringValue("fields @message"),
				LogGroupNames: []types.String{types.StringValue("/aws/ecs/a'pi")},
				Region:        types.StringValue("ap-northeast-1"),
			},
			wantErr: "log_group_names must not contain a single quote, which would break the SOURCE clause: /aws/ecs/a'pi",
		},
		{
			name: "rejects an unknown view",
			model: logWidgetDataSourceModel{
				Query:  types.StringValue("fields @message"),
				Region: types.StringValue("ap-northeast-1"),
				View:   types.StringValue("gauge"),
			},
			wantErr: "view must be one of 'table', 'timeSeries', 'bar' or 'pie', got: gauge",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.model.Width.IsNull() {
				tc.model.Width = types.Int32Value(24)
			}
			if tc.model.Height.IsNull() {
				tc.model.Height = types.Int32Value(8)
			}

			err := tc.model.Validate()
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestLogWidgetDataSourceSettings_BuildQuery(t *testing.T) {
	t.Run("prepends one SOURCE clause per log group", func(t *testing.T) {
		w := logWidgetDataSourceSettings{
			Query:         "fields @message",
			LogGroupNames: []string{"/aws/ecs/api", "/aws/lambda/worker"},
		}

		assert.Equal(t, "SOURCE '/aws/ecs/api' | SOURCE '/aws/lambda/worker' | fields @message", w.buildQuery())
	})

	t.Run("leaves the query alone when no log group is given", func(t *testing.T) {
		w := logWidgetDataSourceSettings{Query: "SOURCE '/aws/ecs/api' | fields @message"}

		assert.Equal(t, "SOURCE '/aws/ecs/api' | fields @message", w.buildQuery())
	})
}

func TestLogWidgetDataSourceSettings_ToCWDashboardBodyWidget(t *testing.T) {
	w := logWidgetDataSourceSettings{
		Type:          typeLogWidget,
		Query:         "fields @timestamp, @message",
		LogGroupNames: []string{"/aws/ecs/api"},
		Region:        "ap-northeast-1",
		AccountId:     "123456789012",
		Title:         "Recent errors",
		View:          "table",
		Width:         24,
		Height:        8,
	}

	cwWidget, err := w.ToCWDashboardBodyWidget(context.TODO())
	assert.NoError(t, err)

	assert.Equal(t, "log", cwWidget.Type)
	assert.Equal(t, int32(24), cwWidget.Width)
	assert.Equal(t, int32(8), cwWidget.Height)

	props, ok := cwWidget.Properties.(CWDashboardBodyWidgetPropertyLog)
	assert.True(t, ok)
	assert.Equal(t, "SOURCE '/aws/ecs/api' | fields @timestamp, @message", props.Query)
	assert.Equal(t, "ap-northeast-1", props.Region)
	assert.Equal(t, "123456789012", props.AccountId)
	assert.Equal(t, "Recent errors", props.Title)
	assert.Equal(t, "table", props.View)
}
