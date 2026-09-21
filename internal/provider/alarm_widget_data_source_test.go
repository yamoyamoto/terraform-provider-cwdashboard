package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tj/assert"
)

func alarmArns(n int) []types.String {
	arns := make([]types.String, 0, n)
	for i := 0; i < n; i++ {
		arns = append(arns, types.StringValue("arn:aws:cloudwatch:ap-northeast-1:123456789012:alarm:a"))
	}

	return arns
}

func TestAlarmWidgetDataSourceModel_Validate(t *testing.T) {
	validArn := types.StringValue("arn:aws:cloudwatch:ap-northeast-1:123456789012:alarm:api-cpu-high")

	tests := []struct {
		name    string
		model   alarmWidgetDataSourceModel
		wantErr string
	}{
		{
			name:  "valid minimum model",
			model: alarmWidgetDataSourceModel{Alarms: []types.String{validArn}},
		},
		{
			name: "valid complete model",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{validArn},
				SortBy: types.StringValue("stateUpdatedTimestamp"),
				States: []types.String{types.StringValue("ALARM"), types.StringValue("INSUFFICIENT_DATA")},
				Title:  types.StringValue("Service health"),
			},
		},
		{
			name: "accepts a composite alarm ARN",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{types.StringValue("arn:aws:cloudwatch:us-east-1:123456789012:alarm:composite-api")},
			},
		},
		{
			name:    "rejects an empty alarm list",
			model:   alarmWidgetDataSourceModel{Alarms: []types.String{}},
			wantErr: "alarms must contain between 1 and 100 ARNs, got: 0",
		},
		{
			name:    "rejects more than 100 alarms",
			model:   alarmWidgetDataSourceModel{Alarms: alarmArns(101)},
			wantErr: "alarms must contain between 1 and 100 ARNs, got: 101",
		},
		{
			name:    "rejects an alarm name that is not an ARN",
			model:   alarmWidgetDataSourceModel{Alarms: []types.String{types.StringValue("api-cpu-high")}},
			wantErr: "invalid alarm ARN: api-cpu-high",
		},
		{
			name: "rejects an ARN for another service",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{types.StringValue("arn:aws:sns:ap-northeast-1:123456789012:alerts")},
			},
			wantErr: "invalid alarm ARN: arn:aws:sns:ap-northeast-1:123456789012:alerts",
		},
		{
			name: "rejects an unknown sort_by",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{validArn},
				SortBy: types.StringValue("name"),
			},
			wantErr: "sort_by must be one of 'default', 'stateUpdatedTimestamp' or 'timestamp', got: name",
		},
		{
			name: "rejects an unknown state",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{validArn},
				States: []types.String{types.StringValue("FIRING")},
			},
			wantErr: "states must only contain 'ALARM', 'INSUFFICIENT_DATA' or 'OK', got: FIRING",
		},
		{
			name: "rejects duplicate states",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{validArn},
				States: []types.String{types.StringValue("OK"), types.StringValue("OK")},
			},
			wantErr: "states must not contain duplicates, got: OK twice",
		},
		{
			name: "rejects a widget wider than the grid",
			model: alarmWidgetDataSourceModel{
				Alarms: []types.String{validArn},
				Width:  types.Int32Value(25),
				Height: types.Int32Value(4),
			},
			wantErr: "width must be between 1 and 24, got: 25",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// width / height are required by the schema; fill in a valid size for
			// the cases that are about something else.
			if tc.model.Width.IsNull() {
				tc.model.Width = types.Int32Value(24)
			}
			if tc.model.Height.IsNull() {
				tc.model.Height = types.Int32Value(4)
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

func TestAlarmWidgetDataSourceSettings_ToCWDashboardBodyWidget(t *testing.T) {
	w := alarmWidgetDataSourceSettings{
		Type:   typeAlarmWidget,
		Alarms: []string{"arn:aws:cloudwatch:ap-northeast-1:123456789012:alarm:api-cpu-high"},
		SortBy: "stateUpdatedTimestamp",
		States: []string{"ALARM"},
		Title:  "Service health",
		Width:  24,
		Height: 4,
	}

	cwWidget, err := w.ToCWDashboardBodyWidget(context.TODO())
	assert.NoError(t, err)

	assert.Equal(t, "alarm", cwWidget.Type)
	assert.Equal(t, int32(24), cwWidget.Width)
	assert.Equal(t, int32(4), cwWidget.Height)
	// X / Y are assigned later by layoutWidgets.
	assert.Equal(t, int32(0), cwWidget.X)
	assert.Equal(t, int32(0), cwWidget.Y)

	props, ok := cwWidget.Properties.(CWDashboardBodyWidgetPropertyAlarm)
	assert.True(t, ok)
	assert.Equal(t, w.Alarms, props.Alarms)
	assert.Equal(t, "stateUpdatedTimestamp", props.SortBy)
	assert.Equal(t, []string{"ALARM"}, props.States)
	assert.Equal(t, "Service health", props.Title)
}
