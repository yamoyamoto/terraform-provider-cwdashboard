package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &metricExpressionDataSource{}
)

type metricExpressionDataSource struct {
}

func NewMetricExpressionDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &metricExpressionDataSource{}
	}
}

func (d *metricExpressionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric_expression"
}

func (d *metricExpressionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"expression": schema.StringAttribute{
				Description: "The expression of the metric",
				Required:    true,
			},
			"color": schema.StringAttribute{
				Description: "The color of the metric",
				Optional:    true,
			},
			"label": schema.StringAttribute{
				Description: "The label of the metric",
				Optional:    true,
			},
			"period": schema.Int32Attribute{
				Description: "The period of the metric",
				Optional:    true,
			},
			"using_metrics": schema.MapAttribute{
				Description: "The metrics used in the expression",
				Optional:    true,
				ElementType: types.StringType,
			},
			"json": schema.StringAttribute{
				Description: "The settings of the metric",
				Computed:    true,
			},
		},
	}
}

type metricExpressionDataSourceModel struct {
	Expression   types.String `tfsdk:"expression"`
	Color        types.String `tfsdk:"color"`
	Label        types.String `tfsdk:"label"`
	Period       types.Int32  `tfsdk:"period"`
	UsingMetrics types.Map    `tfsdk:"using_metrics"`
	Json         types.String `tfsdk:"json"`
}

func (m *metricExpressionDataSourceModel) Validate() error {
	if !m.Period.IsNull() {
		if period := m.Period.ValueInt32(); period < 60 || period%60 != 0 {
			return fmt.Errorf("period must be 60 or a multiple of 60, got: %d", period)
		}
	}

	color := m.Color.ValueString()
	if color != "" {
		if !hexColorPattern.MatchString(color) {
			return fmt.Errorf("invalid color format: %s, must be a six-digit hex color code (e.g., #FF0000)", color)
		}
	}

	// Validate metric variable names in usingMetrics
	var invalidVarNames []string
	usingMetrics := m.UsingMetrics.Elements()
	for k := range usingMetrics {
		if !isValidVariableName(k) {
			invalidVarNames = append(invalidVarNames, k)
		}
	}
	if len(invalidVarNames) > 0 {
		return fmt.Errorf("invalid variable names in expression: %v. Must start with lowercase letter and only contain alphanumerics", invalidVarNames)
	}

	// Validate expression syntax and references
	expr := m.Expression.ValueString()
	if expr == "" {
		return fmt.Errorf("expression cannot be empty")
	}

	// Check for unknown identifiers in expression
	identifiers := findIdentifiersInExpression(expr)
	var missingIds []string
	for _, id := range identifiers {
		if _, exists := usingMetrics[id]; !exists {
			missingIds = append(missingIds, id)
		}
	}

	isSpecialCommand := regexp.MustCompile(`(?i)\s*INSIGHT_RULE_METRIC|SELECT|SEARCH|METRICS\s.*`).MatchString(expr)
	if !isSpecialCommand && len(missingIds) > 0 {
		return fmt.Errorf("missing metrics in using_metrics: %v", missingIds)
	}

	return nil
}

func isValidVariableName(name string) bool {
	if len(name) == 0 {
		return false
	}
	if !unicode.IsLower(rune(name[0])) {
		return false
	}
	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

func findIdentifiersInExpression(expr string) []string {
	// This is a simplified implementation
	pattern := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	matches := pattern.FindAllString(expr, -1)

	// Filter out common math functions and keywords
	keywords := map[string]bool{
		"SELECT": true,
		"FROM":   true,
		"WHERE":  true,
		"GROUP":  true,
		"BY":     true,
		// https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/using-metric-math.html#metric-math-syntax
		"ABS":                    true,
		"ANOMALY_DETECTION_BAND": true,
		"AVG":                    true,
		"CEIL":                   true,
		"DATAPOINT_COUNT":        true,
		"DB_PERF_INSIGHTS":       true,
		"DIFF":                   true,
		"DIFF_TIME":              true,
		"FILL":                   true,
		"FIRST":                  true,
		"LAST":                   true,
		"FLOOR":                  true,
		"IF":                     true,
		"INSIGHT_RULE_METRIC":    true,
		"LAMBDA":                 true,
		"LOG":                    true,
		"LOG10":                  true,
		"MAX":                    true,
		"METRIC_COUNT":           true,
		"METRICS":                true,
		"MIN":                    true,

		"MINUTE": true,
		"HOUR":   true,
		"DAY":    true,
		"DATE":   true,
		"MONTH":  true,
		"YEAR":   true,
		"EPOCH":  true,

		"PERIOD":        true,
		"RATE":          true,
		"REMOVE_EMPTY":  true,
		"RUNNING_SUM":   true,
		"SEARCH":        true,
		"SERVICE_QUOTA": true,
		"SLICE":         true,
		"SORT":          true,
		"STDDEV":        true,
		"SUM":           true,
		"TIME_SERIES":   true,
	}

	var identifiers []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if !keywords[match] && !seen[match] {
			identifiers = append(identifiers, match)
			seen[match] = true
		}
	}

	return identifiers
}

type metricExpressionDataSourceSettings struct {
	Type         string            `json:"type"`
	Expression   string            `json:"expression"`
	Color        string            `json:"color"`
	Label        string            `json:"label"`
	Period       int32             `json:"period"`
	UsingMetrics map[string]string `json:"using_metrics"`
}

const (
	typeNameOfMetricExpressionDataSource = "metric_expression"
)

func (s *metricExpressionDataSourceSettings) GetType() string {
	return typeNameOfMetricExpressionDataSource
}

func (d *metricExpressionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state metricExpressionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := state.Validate(); err != nil {
		resp.Diagnostics.AddError("invalid settings", err.Error())
		return
	}

	usingMetrics := make(map[string]string)
	for k, v := range state.UsingMetrics.Elements() {
		vs, ok := v.(types.String)
		if !ok {
			resp.Diagnostics.AddError("failed to convert using_metrics", "failed to convert using_metrics")
			return
		}

		usingMetrics[k] = vs.ValueString()
	}

	settings := metricExpressionDataSourceSettings{
		Type:         typeNameOfMetricExpressionDataSource,
		Expression:   state.Expression.ValueString(),
		Color:        state.Color.ValueString(),
		Label:        state.Label.ValueString(),
		Period:       state.Period.ValueInt32(),
		UsingMetrics: usingMetrics,
	}

	b, err := json.Marshal(settings)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal settings", err.Error())
		return
	}

	state.Json = types.StringValue(string(b))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (s *metricExpressionDataSourceSettings) buildMetricWidgetMetricSettingsList(left bool) ([][]interface{}, error) {
	settings := make([][]interface{}, 0)

	// sort keys to make the order of metrics deterministic
	keys := make([]string, 0, len(s.UsingMetrics))
	for id := range s.UsingMetrics {
		keys = append(keys, id)
	}
	sort.Strings(keys)

	for _, id := range keys {
		usingMetric := s.UsingMetrics[id]
		var m metricDataSourceSettings
		if err := json.Unmarshal([]byte(usingMetric), &m); err != nil {
			return nil, err
		}

		ms, err := m.buildMetricWidgetMetricsSettings(left, map[string]interface{}{
			"id":      id,
			"visible": false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build metric widget metric settings: %w", err)
		}

		settings = append(settings, ms)
	}

	metricExpressionSettings := []interface{}{
		map[string]interface{}{
			"expression": s.Expression,
		},
	}

	if s.Color != "" {
		metricExpressionSettings[0].(map[string]interface{})["color"] = s.Color
	}

	if s.Label != "" {
		metricExpressionSettings[0].(map[string]interface{})["label"] = s.Label
	}

	settings = append(settings, metricExpressionSettings)

	return settings, nil
}
