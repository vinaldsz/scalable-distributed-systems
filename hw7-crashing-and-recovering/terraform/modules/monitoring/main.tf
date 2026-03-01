resource "aws_cloudwatch_metric_alarm" "health_check_failures" {
  alarm_name          = var.health_alarm_name
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "UnHealthyHostCount"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Average"
  threshold           = 0

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
    TargetGroup  = var.target_group_arn_suffix
  }

  alarm_description = "Alert when ECS tasks fail health checks"
  treat_missing_data = "notBreaching"

  tags = var.common_tags
}

resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = var.dashboard_name

  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric"
        properties = {
          metrics = [
            ["AWS/ApplicationELB", "TargetResponseTime", { stat = "Average" }],
            ["AWS/ApplicationELB", "RequestCount", { stat = "Sum" }],
            ["AWS/ApplicationELB", "HTTPCode_Target_2XX_Count", { stat = "Sum" }],
            ["AWS/ApplicationELB", "HTTPCode_Target_5XX_Count", { stat = "Sum" }],
            ["AWS/ECS", "CPUUtilization", { stat = "Average" }],
            ["AWS/ECS", "MemoryUtilization", { stat = "Average" }],
          ]
          period = 60
          stat   = "Average"
          region = var.aws_region
          title  = var.dashboard_title
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      }
    ]
  })
}
