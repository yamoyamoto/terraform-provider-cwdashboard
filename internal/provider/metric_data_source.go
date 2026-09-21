package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type metricDataSource struct {
}

func NewMetricDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &metricDataSource{}
	}
}

func (d *metricDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

func (d *metricDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"metric_name": schema.StringAttribute{
				Description: "Name of the metric",
				Required:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Namespace of the metric",
				Required:    true,
			},
			"account": schema.StringAttribute{
				Description: "The ID of the AWS account this metric comes from. Rendered as `accountId` in the dashboard body.",
				Optional:    true,
			},
			"color": schema.StringAttribute{
				Description: "The hex color code, prefixed with '#' (e.g. '#00ff00'), to use when this metric is rendered on a graph",
				Optional:    true,
			},
			"dimensions_map": schema.MapAttribute{
				Description: "Dimensions of the metric",
				Optional:    true,
				ElementType: types.StringType,
			},
			"label": schema.StringAttribute{
				Description: "Label for this metric when added to a Graph in a Dashboard",
				Optional:    true,
			},
			"period": schema.Int32Attribute{
				Description: "The period over which the specified statistic is applied",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region this metric comes from. Set it to graph metrics from another " +
					"region on the same dashboard (for example CloudFront and CLOUDFRONT-scoped WAF, " +
					"whose metrics live in `us-east-1`).",
				Optional: true,
			},
			"statistic": schema.StringAttribute{
				Description: "What function to use for aggregating",
				Optional:    true,
			},
			"unit": schema.StringAttribute{
				Description: "Unit used to filter the metric stream. " +
					"**Not rendered into the dashboard body**: the CloudWatch dashboard body structure " +
					"has no per-metric unit field, so this value is accepted but ignored. " +
					"Use `left_y_axis.show_units` / `right_y_axis.show_units` on the graph widget to " +
					"control unit suffixes on the axis labels.",
				Optional: true,
			},

			"json": schema.StringAttribute{
				Description: "The settings of the metric",
				Computed:    true,
			},
		},
	}
}

type metricDataSourceModel struct {
	MetricName    types.String `tfsdk:"metric_name"`
	Namespace     types.String `tfsdk:"namespace"`
	Account       types.String `tfsdk:"account"`
	Color         types.String `tfsdk:"color"`
	DimensionsMap types.Map    `tfsdk:"dimensions_map"`
	Label         types.String `tfsdk:"label"`
	Period        types.Int32  `tfsdk:"period"`
	Region        types.String `tfsdk:"region"`
	Statistic     types.String `tfsdk:"statistic"`
	Unit          types.String `tfsdk:"unit"`

	Json types.String `tfsdk:"json"`
}

var (
	validMetricStatistics = map[string]bool{
		"SampleCount": true,
		"Average":     true,
		"Sum":         true,
		"Minimum":     true,
		"Maximum":     true,
	}
)

func (d *metricDataSourceModel) Validate() error {
	// Period must be 60 or a multiple of 60
	if !d.Period.IsNull() {
		if period := d.Period.ValueInt32(); period < 60 || period%60 != 0 {
			return fmt.Errorf("period must be 60 or a multiple of 60, got: %d", period)
		}
	}

	// Validate CloudWatch statistics
	stat := d.Statistic.ValueString()
	if strings.HasPrefix(stat, "p") {
		percentile, err := strconv.ParseFloat(strings.TrimPrefix(stat, "p"), 64)
		if err != nil || percentile < 0 || percentile > 100 {
			return fmt.Errorf("invalid percentile statistic: %s, must be between p0 and p100", stat)
		}
	} else if !validMetricStatistics[stat] {
		return fmt.Errorf("invalid statistic: %s", stat)
	}

	color := d.Color.ValueString()
	if color != "" {
		colorPattern := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
		if !colorPattern.MatchString(color) {
			return fmt.Errorf("invalid color format: %s, must be a six-digit hex color code (e.g., #FF0000)", color)
		}
	}

	return nil
}

type metricDataSourceSettings struct {
	Type          string            `json:"type"`
	MetricName    string            `json:"metricName"`
	Namespace     string            `json:"namespace"`
	Account       string            `json:"account,omitempty"`
	Color         string            `json:"color,omitempty"`
	DimensionsMap map[string]string `json:"dimensionsMap,omitempty"`
	Label         string            `json:"label,omitempty"`
	Period        int32             `json:"period,omitempty"`
	Region        string            `json:"region,omitempty"`
	Statistic     string            `json:"statistic,omitempty"`
	Unit          string            `json:"unit,omitempty"`
}

const (
	typeNameOfMetricDataSource = "metric"
)

func (s *metricDataSourceSettings) GetType() string {
	return typeNameOfMetricDataSource
}

func (d *metricDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state metricDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("invalid settings", err.Error())
		return
	}

	dimensionsMap := make(map[string]string)
	for i, v := range state.DimensionsMap.Elements() {
		switch typedV := v.(type) {
		case types.String:
			dimensionsMap[i] = typedV.ValueString()
		default:
			resp.Diagnostics.AddError("invalid type for dimensions map", "expected string")
			return
		}
	}

	settings := metricDataSourceSettings{
		Type:          typeNameOfMetricDataSource,
		MetricName:    state.MetricName.ValueString(),
		Namespace:     state.Namespace.ValueString(),
		Account:       state.Account.ValueString(),
		Color:         state.Color.ValueString(),
		DimensionsMap: dimensionsMap,
		Label:         state.Label.ValueString(),
		Period:        state.Period.ValueInt32(),
		Region:        state.Region.ValueString(),
		Statistic:     state.Statistic.ValueString(),
		Unit:          state.Unit.ValueString(),
	}

	b, err := json.Marshal(settings)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal metric settings", err.Error())
		return
	}
	tflog.Info(ctx, "metric settings", map[string]interface{}{
		"settings": string(b),
	})

	state.Json = types.StringValue(string(b))

	resp.State.Set(ctx, state)
}

// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/CloudWatch-Dashboard-Body-Structure.html#CloudWatch-Dashboard-Properties-Metrics-Array-Format
func (s *metricDataSourceSettings) buildMetricWidgetMetricsSettings(left bool, extra map[string]interface{}) ([]interface{}, error) {
	settings := make([]interface{}, 0)

	settings = append(settings, s.Namespace)
	settings = append(settings, s.MetricName)

	// sort keys to make the order of dimensions deterministic. CloudWatch identifies a
	// metric by the set of its dimensions, not by their order, so sorting is free. Without
	// it Go's randomized map iteration produces a different dashboard body on every run,
	// which shows up as a permanent diff on aws_cloudwatch_dashboard.
	dimKeys := make([]string, 0, len(s.DimensionsMap))
	for dimKey := range s.DimensionsMap {
		dimKeys = append(dimKeys, dimKey)
	}
	sort.Strings(dimKeys)

	for _, dimKey := range dimKeys {
		settings = append(settings, dimKey)
		settings = append(settings, s.DimensionsMap[dimKey])
	}

	renderingProperties := map[string]interface{}{}

	if s.Account != "" {
		renderingProperties["accountId"] = s.Account
	}
	if s.Color != "" {
		renderingProperties["color"] = s.Color
	}
	if s.Label != "" {
		renderingProperties["label"] = s.Label
	}
	if s.Period != 0 {
		renderingProperties["period"] = s.Period
	}
	if s.Region != "" {
		renderingProperties["region"] = s.Region
	}
	if s.Statistic != "" {
		renderingProperties["stat"] = s.Statistic
	}
	// NOTE: s.Unit is intentionally not rendered. The CloudWatch dashboard body
	// structure has no per-metric unit field.

	if left {
		renderingProperties["yAxis"] = "left"
	} else {
		renderingProperties["yAxis"] = "right"
	}

	for k, v := range extra {
		renderingProperties[k] = v
	}

	settings = append(settings, renderingProperties)

	return settings, nil
}
