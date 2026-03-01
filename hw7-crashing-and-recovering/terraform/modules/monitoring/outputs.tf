output "dashboard_arn" {
  description = "ARN of the CloudWatch dashboard"
  value       = aws_cloudwatch_dashboard.main.dashboard_arn
}

output "alarm_arn" {
  description = "ARN of the health check alarm"
  value       = aws_cloudwatch_metric_alarm.health_check_failures.arn
}
