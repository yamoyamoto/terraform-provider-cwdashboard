package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Code-Hex/synchro/iso8601"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &dashboardDataSource{}
)

type dashboardDataSource struct {
}

func NewDashboardDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &dashboardDataSource{}
	}
}

func (d *dashboardDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName
}

func (d *dashboardDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"widgets": schema.ListAttribute{
				Description: `The list of widgets in the dashboard.`,
				Required:    true,
				ElementType: types.StringType,
			},
			"end": schema.StringAttribute{
				Description: `The end of the time range to use for each widget on the dashboard when the dashboard loads. ` +
					`If you specify a value for end, you must also specify a value for ` + "`start`" + `. ` +
					`For each of these values, specify an absolute time in the ISO 8601 format. ` +
					`For example, ` + "`2018-12-17T06:00:00.000Z`" + `.`,
				Optional: true,
			},
			"start": schema.StringAttribute{
				Description: `The start of the time range to use for each widget on the dashboard. ` +
					`You can specify ` + "`start`" + ` without specifying end to specify a relative time range that ends with the current time. ` +
					`In this case, the value of ` + "`start`" + ` must begin with ` + "`-PT`" + ` if you specify a time range in minutes or hours, and must begin with ` + "`-P`" + ` if you specify a time range in days, weeks, or months. ` +
					`You can then use M, H, D, W and M as abbreviations for minutes, hours, days, weeks and months. ` +
					`For example, ` + "`-PT5M`" + ` shows the last 5 minutes, ` + "`-PT8H`" + ` shows the last 8 hours, and ` + "`-P3M`" + ` shows the last three months. ` +
					`You can also use ` + "`start`" + ` along with an end field, to specify an absolute time range. ` +
					`When specifying an absolute time range, use the ISO 8601 format. ` +
					`For example, ` + "`2018-12-17T06:00:00.000Z`" + `. ` +
					`If you omit ` + "`start`" + `, the dashboard shows the default time range when it loads.`,
				Optional: true,
			},
			"period_override": schema.StringAttribute{
				Description: `Use this field to specify the period for the graphs when the dashboard loads. ` +
					`Specifying ` + "`auto`" + ` causes the period of all graphs on the dashboard to automatically adapt to the time range of the dashboard. ` +
					`Specifying ` + "`inherit`" + ` ensures that the period set for each graph is always obeyed. ` +
					`Valid Values: ` + "`auto`" + ` |` + "`inherit`",
				Optional: true,
			},
			"json": schema.StringAttribute{
				Description: "The json of the dashboard body",
				Computed:    true,
			},
		},
	}
}

type dashboardDataSourceModel struct {
	Start          types.String `tfsdk:"start"`
	End            types.String `tfsdk:"end"`
	PeriodOverride types.String `tfsdk:"period_override"`
	Widgets        types.List   `tfsdk:"widgets"`
	Json           types.String `tfsdk:"json"`
}

const (
	dashboardMaxWidgets = 500

	periodOverrideAuto    = "auto"
	periodOverrideInherit = "inherit"
)

var (
	// -PTxx[MH] （minute、hour）
	relativeMinutesHoursPattern = regexp.MustCompile(`^-PT\d+[MH]$`)

	// -Pxx[DWM] （day、week、month）
	relativeDaysWeeksMonthsPattern = regexp.MustCompile(`^-P\d+[DWM]$`)
)

func (d *dashboardDataSourceModel) Validate() error {
	if len(d.Widgets.Elements()) > dashboardMaxWidgets {
		return fmt.Errorf("maximum number of widgets is %d. Got %d", dashboardMaxWidgets, len(d.Widgets.Elements()))
	}

	// check if start is a valid ISO8601 date
	if !d.Start.IsNull() {
		_, err := iso8601.ParseDateTime(d.Start.ValueString())
		if err != nil {
			// check if start is a valid relative time
			if strings.HasPrefix(d.Start.ValueString(), "-PT") {
				if !relativeMinutesHoursPattern.MatchString(d.Start.ValueString()) {
					return fmt.Errorf("start must be a valid ISO8601 date or a valid relative time: %w", err)
				}
			} else if strings.HasPrefix(d.Start.ValueString(), "-P") {
				if !relativeDaysWeeksMonthsPattern.MatchString(d.Start.ValueString()) {
					return fmt.Errorf("start must be a valid ISO8601 date or a valid relative time: %w", err)
				}
			} else {
				return fmt.Errorf("start must be a valid ISO8601 date or a valid relative time: %w", err)
			}
		}
	}

	// check if end is a valid ISO8601 date
	if !d.End.IsNull() {
		_, err := iso8601.ParseDateTime(d.End.ValueString())
		if err != nil {
			return fmt.Errorf("end must be a valid ISO8601 date: %w", err)
		}
	}

	// check if period_override is a valid value
	if !d.PeriodOverride.IsNull() {
		if d.PeriodOverride.ValueString() != periodOverrideAuto && d.PeriodOverride.ValueString() != periodOverrideInherit {
			return fmt.Errorf("period_override must be either 'auto' or 'inherit'")
		}
	}

	return nil
}

func (d *dashboardDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state dashboardDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("failed to validate dashboard data source", err.Error())
		return
	}

	widgets, err := d.parseToWidgetSettings(ctx, state.Widgets.Elements())
	if err != nil {
		resp.Diagnostics.AddError("failed to parse widgets", err.Error())
		return
	}

	dashboardJson, err := buildDashboardBodyJson(ctx, state, widgets)
	if err != nil {
		resp.Diagnostics.AddError("failed to build dashboard json", err.Error())
		return
	}

	tflog.Info(ctx, "built dashboard json", map[string]interface{}{
		"dashboard_json": dashboardJson,
	})

	state.Json = types.StringValue(dashboardJson)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// parseToWidgetSettings decodes each widget's JSON into its settings struct. It does
// not assign positions: layout is done once, over the whole list, in
// buildDashboardBodyJson.
func (d *dashboardDataSource) parseToWidgetSettings(_ context.Context, elements []attr.Value) ([]interface{}, error) {
	widgets := make([]interface{}, 0)
	for _, elem := range elements {
		// NOTE: Unmarshal twice because of double escaping by Terraform
		var escaped string
		if err := json.Unmarshal([]byte(elem.String()), &escaped); err != nil {
			return nil, fmt.Errorf("failed to unmarshal widget json: %w", err)
		}

		w := map[string]interface{}{}
		if err := json.Unmarshal([]byte(escaped), &w); err != nil {
			return nil, fmt.Errorf("failed to unmarshal widget json: %w", err)
		}

		widgetType, ok := w["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing widget type")
		}

		switch widgetType {
		case typeTextWidget:
			var w textWidgetDataSourceSettings
			if err := json.Unmarshal([]byte(escaped), &w); err != nil {
				return nil, fmt.Errorf("failed to unmarshal text widget json: %w", err)
			}
			widgets = append(widgets, w)
		case typeGraphWidget:
			var w graphWidgetDataSourceSettings
			if err := json.Unmarshal([]byte(escaped), &w); err != nil {
				return nil, fmt.Errorf("failed to unmarshal graph widget json: %w", err)
			}
			widgets = append(widgets, w)
		default:
			return nil, fmt.Errorf("unsupported widget type: %s", widgetType)
		}
	}

	return widgets, nil
}
