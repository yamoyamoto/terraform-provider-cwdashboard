data "cwdashboard_log_widget" "this" {
  title  = "Recent errors"
  region = "ap-northeast-1"

  # Each name becomes a `SOURCE '<name>' |` clause in front of the query.
  log_group_names = ["/aws/ecs/api"]

  query = join("\n| ", [
    "filter @message like /\\[ERROR\\]/",
    "fields @timestamp, @message",
    "sort @timestamp desc",
    "limit 20",
  ])

  width  = 24
  height = 8
}

data "cwdashboard" "this" {
  start = "-PT3H"
  widgets = [
    data.cwdashboard_log_widget.this.json,
  ]
}

# to create dashboard, use AWS Terraform Provider with the dashboard JSON
resource "aws_cloudwatch_dashboard" "this" {
  dashboard_name = "test-dashboard"
  dashboard_body = data.cwdashboard.this.json
}
