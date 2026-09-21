data "cwdashboard_alarm_widget" "this" {
  title = "Service health"

  alarms = [
    "arn:aws:cloudwatch:ap-northeast-1:123456789012:alarm:api-cpu-high",
    "arn:aws:cloudwatch:ap-northeast-1:123456789012:alarm:api-5xx-high",
  ]

  # Show alarms that are firing first, then the ones waiting for data.
  sort_by = "stateUpdatedTimestamp"

  width  = 24
  height = 4
}

data "cwdashboard" "this" {
  start = "-PT3H"
  widgets = [
    data.cwdashboard_alarm_widget.this.json,
  ]
}

# to create dashboard, use AWS Terraform Provider with the dashboard JSON
resource "aws_cloudwatch_dashboard" "this" {
  dashboard_name = "test-dashboard"
  dashboard_body = data.cwdashboard.this.json
}
