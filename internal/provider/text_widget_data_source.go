package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &textWidgetDataSource{}
)

type textWidgetDataSource struct {
}

func NewTextWidgetDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &textWidgetDataSource{}
	}
}

func (d *textWidgetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_text_widget"
}

func (d *textWidgetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"markdown": schema.StringAttribute{
				Description: "The text to be displayed by the widget. Use this parameter only for text widgets.",
				Required:    true,
			},
			"background": schema.StringAttribute{
				Description: "Specifies whether the text widget has a solid or transparent background. The value " + "`transparent`" + " makes the widget transparent. The value " + "`solid`" + " is the default.",
				Optional:    true,
			},
			"width": schema.Int32Attribute{
				Description: "The width of the widget",
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

type textWidgetDataSourceModel struct {
	Markdown   types.String `tfsdk:"markdown"`
	Background types.String `tfsdk:"background"`
	Width      types.Int32  `tfsdk:"width"`
	Height     types.Int32  `tfsdk:"height"`

	Json types.String `tfsdk:"json"`
}

func (d *textWidgetDataSourceModel) Validate() error {
	return validateWidgetSize(d.Width.ValueInt32(), d.Height.ValueInt32())
}

type textWidgetDataSourceSettings struct {
	Type       string `json:"type"`
	Markdown   string `json:"markdown"`
	Background string `json:"background"`
	Width      int32  `json:"width"`
	Height     int32  `json:"height"`
}

const (
	typeTextWidget = "text"
)

func (d *textWidgetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state textWidgetDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("failed to validate text widget data source", err.Error())
		return
	}

	settings := textWidgetDataSourceSettings{
		Type:       typeTextWidget,
		Markdown:   state.Markdown.ValueString(),
		Background: state.Background.ValueString(),
		Width:      state.Width.ValueInt32(),
		Height:     state.Height.ValueInt32(),
	}

	b, err := json.Marshal(settings)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal widget settings", err.Error())
		return
	}

	tflog.Info(ctx, "text widget settings", map[string]interface{}{
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
func (w textWidgetDataSourceSettings) ToCWDashboardBodyWidget(ctx context.Context) (CWDashboardBodyWidget, error) {
	cwWidget := CWDashboardBodyWidget{
		Type:   "text",
		Width:  w.Width,
		Height: w.Height,
		Properties: CWDashboardBodyWidgetPropertyText{
			Markdown:   w.Markdown,
			Background: w.Background,
		},
	}

	tflog.Debug(ctx, "built text widget", map[string]interface{}{
		"widget": cwWidget,
	})

	return cwWidget, nil
}
