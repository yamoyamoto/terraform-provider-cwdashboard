package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &alarmWidgetDataSource{}

	// Alarm ARNs, including composite alarms, share this shape.
	alarmArnPattern = regexp.MustCompile(`^arn:aws[a-zA-Z-]*:cloudwatch:[a-z0-9-]+:\d{12}:alarm:.+$`)
)

type alarmWidgetDataSource struct {
}

func NewAlarmWidgetDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &alarmWidgetDataSource{}
	}
}

func (d *alarmWidgetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alarm_widget"
}

func (d *alarmWidgetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An alarm status widget, which lists the current state of the alarms you give it. " +
			"To draw a threshold on a graph instead, use the `annotations` argument of " +
			"`cwdashboard_graph_widget`.",
		Attributes: map[string]schema.Attribute{
			"alarms": schema.ListAttribute{
				Description: "The ARNs of the alarms to include in the widget. Between 1 and 100.",
				Required:    true,
				ElementType: types.StringType,
			},
			"sort_by": schema.StringAttribute{
				Description: "How to sort the alarms. " + "`default`" + " sorts alphabetically by alarm name, " +
					"`stateUpdatedTimestamp`" + " groups by state (ALARM, then INSUFFICIENT_DATA, then OK) and " +
					"sorts each group by when it last changed, and " + "`timestamp`" + " sorts by the last state " +
					"change regardless of state. Defaults to alphabetical order.",
				Optional: true,
			},
			"states": schema.ListAttribute{
				Description: "Show only the alarms currently in these states. " +
					"Valid values: " + "`ALARM`" + ", " + "`INSUFFICIENT_DATA`" + ", " + "`OK`" + ". " +
					"When omitted, every alarm in " + "`alarms`" + " is shown.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"title": schema.StringAttribute{
				Description: "Title for the widget",
				Optional:    true,
			},
			"width": schema.Int32Attribute{
				Description: "The width of the widget, in a grid of 24 units wide",
				Required:    true,
			},
			"height": schema.Int32Attribute{
				Description: "The height of the widget",
				Required:    true,
			},

			"json": schema.StringAttribute{
				Description: "The settings of the widget",
				Computed:    true,
			},
		},
	}
}

type alarmWidgetDataSourceModel struct {
	Alarms []types.String `tfsdk:"alarms"`
	SortBy types.String   `tfsdk:"sort_by"`
	States []types.String `tfsdk:"states"`
	Title  types.String   `tfsdk:"title"`
	Width  types.Int32    `tfsdk:"width"`
	Height types.Int32    `tfsdk:"height"`

	Json types.String `tfsdk:"json"`
}

const (
	typeAlarmWidget = "alarm"

	alarmWidgetMaxAlarms = 100
)

func (d *alarmWidgetDataSourceModel) Validate() error {
	if len(d.Alarms) < 1 || len(d.Alarms) > alarmWidgetMaxAlarms {
		return fmt.Errorf("alarms must contain between 1 and %d ARNs, got: %d", alarmWidgetMaxAlarms, len(d.Alarms))
	}

	for _, alarm := range d.Alarms {
		if !alarmArnPattern.MatchString(alarm.ValueString()) {
			return fmt.Errorf("invalid alarm ARN: %s", alarm.ValueString())
		}
	}

	if !d.SortBy.IsNull() {
		sortBy := d.SortBy.ValueString()
		validSortBy := map[string]bool{
			"default":               true,
			"stateUpdatedTimestamp": true,
			"timestamp":             true,
		}
		if !validSortBy[sortBy] {
			return fmt.Errorf("sort_by must be one of 'default', 'stateUpdatedTimestamp' or 'timestamp', got: %s", sortBy)
		}
	}

	validStates := map[string]bool{
		"ALARM":             true,
		"INSUFFICIENT_DATA": true,
		"OK":                true,
	}
	seenStates := map[string]bool{}
	for _, state := range d.States {
		value := state.ValueString()
		if !validStates[value] {
			return fmt.Errorf("states must only contain 'ALARM', 'INSUFFICIENT_DATA' or 'OK', got: %s", value)
		}
		if seenStates[value] {
			return fmt.Errorf("states must not contain duplicates, got: %s twice", value)
		}
		seenStates[value] = true
	}

	return validateWidgetSize(d.Width.ValueInt32(), d.Height.ValueInt32())
}

type alarmWidgetDataSourceSettings struct {
	Type   string   `json:"type"`
	Alarms []string `json:"alarms"`
	SortBy string   `json:"sort_by,omitempty"`
	States []string `json:"states,omitempty"`
	Title  string   `json:"title,omitempty"`
	Width  int32    `json:"width"`
	Height int32    `json:"height"`
}

func (d *alarmWidgetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state alarmWidgetDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("failed to validate alarm widget data source", err.Error())
		return
	}

	settings := alarmWidgetDataSourceSettings{
		Type:   typeAlarmWidget,
		Alarms: toStringSlice(state.Alarms),
		SortBy: state.SortBy.ValueString(),
		States: toStringSlice(state.States),
		Title:  state.Title.ValueString(),
		Width:  state.Width.ValueInt32(),
		Height: state.Height.ValueInt32(),
	}

	b, err := json.Marshal(settings)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal widget settings", err.Error())
		return
	}

	tflog.Info(ctx, "alarm widget settings", map[string]interface{}{
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
func (w alarmWidgetDataSourceSettings) ToCWDashboardBodyWidget(ctx context.Context) (CWDashboardBodyWidget, error) {
	cwWidget := CWDashboardBodyWidget{
		Type:   typeAlarmWidget,
		Width:  w.Width,
		Height: w.Height,
		Properties: CWDashboardBodyWidgetPropertyAlarm{
			Alarms: w.Alarms,
			SortBy: w.SortBy,
			States: w.States,
			Title:  w.Title,
		},
	}

	tflog.Debug(ctx, "built alarm widget", map[string]interface{}{
		"widget": cwWidget,
	})

	return cwWidget, nil
}
