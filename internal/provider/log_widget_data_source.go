package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &logWidgetDataSource{}
)

type logWidgetDataSource struct {
}

func NewLogWidgetDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &logWidgetDataSource{}
	}
}

func (d *logWidgetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_widget"
}

func (d *logWidgetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A widget showing the results of a CloudWatch Logs Insights query.",
		Attributes: map[string]schema.Attribute{
			"query": schema.StringAttribute{
				Description: "The CloudWatch Logs Insights query. Separate each line with " + "`\\n|`" + ". " +
					"The log groups to search come from " + "`log_group_names`" + "; if you leave that empty " +
					"you must prefix the query with the " + "`SOURCE '<log group>' |`" + " clauses yourself.",
				Required: true,
			},
			"log_group_names": schema.ListAttribute{
				Description: "The log groups to query. Each one is turned into a " + "`SOURCE '<name>' |`" + " " +
					"clause in front of " + "`query`" + ", which is easier to compose from Terraform values " +
					"than writing the clauses by hand.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"region": schema.StringAttribute{
				Description: "The region of the logs query",
				Required:    true,
			},
			"account_id": schema.StringAttribute{
				Description: "The ID of the AWS account holding the logs, for a cross-account query",
				Optional:    true,
			},
			"title": schema.StringAttribute{
				Description: "Title for the widget",
				Optional:    true,
			},
			"view": schema.StringAttribute{
				Description: "How to display the results. Valid values: " + "`table`" + ", " +
					"`timeSeries`" + ", " + "`bar`" + ", " + "`pie`" + ". Defaults to " + "`table`" + ".",
				Optional: true,
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

type logWidgetDataSourceModel struct {
	Query         types.String   `tfsdk:"query"`
	LogGroupNames []types.String `tfsdk:"log_group_names"`
	Region        types.String   `tfsdk:"region"`
	AccountId     types.String   `tfsdk:"account_id"`
	Title         types.String   `tfsdk:"title"`
	View          types.String   `tfsdk:"view"`
	Width         types.Int32    `tfsdk:"width"`
	Height        types.Int32    `tfsdk:"height"`

	Json types.String `tfsdk:"json"`
}

const (
	typeLogWidget = "log"

	logWidgetSourceClausePrefix = "SOURCE"
)

func (d *logWidgetDataSourceModel) Validate() error {
	query := strings.TrimSpace(d.Query.ValueString())
	if query == "" {
		return fmt.Errorf("query must not be empty")
	}

	if len(d.LogGroupNames) > 0 {
		// Both forms produce SOURCE clauses, so accepting both at once would
		// build a query naming the log groups twice.
		if strings.HasPrefix(query, logWidgetSourceClausePrefix) {
			return fmt.Errorf("query must not start with SOURCE when log_group_names is set: the SOURCE clauses are generated from log_group_names")
		}

		for _, logGroupName := range d.LogGroupNames {
			name := logGroupName.ValueString()
			if name == "" {
				return fmt.Errorf("log_group_names must not contain an empty name")
			}
			if strings.Contains(name, "'") {
				return fmt.Errorf("log_group_names must not contain a single quote, which would break the SOURCE clause: %s", name)
			}
		}
	}

	if !d.View.IsNull() {
		view := d.View.ValueString()
		validViews := map[string]bool{
			"table":      true,
			"timeSeries": true,
			"bar":        true,
			"pie":        true,
		}
		if !validViews[view] {
			return fmt.Errorf("view must be one of 'table', 'timeSeries', 'bar' or 'pie', got: %s", view)
		}
	}

	return validateWidgetSize(d.Width.ValueInt32(), d.Height.ValueInt32())
}

type logWidgetDataSourceSettings struct {
	Type          string   `json:"type"`
	Query         string   `json:"query"`
	LogGroupNames []string `json:"log_group_names,omitempty"`
	Region        string   `json:"region"`
	AccountId     string   `json:"account_id,omitempty"`
	Title         string   `json:"title,omitempty"`
	View          string   `json:"view,omitempty"`
	Width         int32    `json:"width"`
	Height        int32    `json:"height"`
}

// buildQuery prepends one SOURCE clause per log group, which is the form
// CloudWatch expects a dashboard log widget query to take.
func (w logWidgetDataSourceSettings) buildQuery() string {
	if len(w.LogGroupNames) == 0 {
		return w.Query
	}

	var b strings.Builder
	for _, logGroupName := range w.LogGroupNames {
		fmt.Fprintf(&b, "SOURCE '%s' | ", logGroupName)
	}
	b.WriteString(w.Query)

	return b.String()
}

func (d *logWidgetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state logWidgetDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("failed to validate log widget data source", err.Error())
		return
	}

	settings := logWidgetDataSourceSettings{
		Type:          typeLogWidget,
		Query:         state.Query.ValueString(),
		LogGroupNames: toStringSlice(state.LogGroupNames),
		Region:        state.Region.ValueString(),
		AccountId:     state.AccountId.ValueString(),
		Title:         state.Title.ValueString(),
		View:          state.View.ValueString(),
		Width:         state.Width.ValueInt32(),
		Height:        state.Height.ValueInt32(),
	}

	b, err := json.Marshal(settings)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal widget settings", err.Error())
		return
	}

	tflog.Info(ctx, "log widget settings", map[string]interface{}{
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
func (w logWidgetDataSourceSettings) ToCWDashboardBodyWidget(ctx context.Context) (CWDashboardBodyWidget, error) {
	cwWidget := CWDashboardBodyWidget{
		Type:   typeLogWidget,
		Width:  w.Width,
		Height: w.Height,
		Properties: CWDashboardBodyWidgetPropertyLog{
			AccountId: w.AccountId,
			Region:    w.Region,
			Title:     w.Title,
			Query:     w.buildQuery(),
			View:      w.View,
		},
	}

	tflog.Debug(ctx, "built log widget", map[string]interface{}{
		"widget": cwWidget,
	})

	return cwWidget, nil
}
