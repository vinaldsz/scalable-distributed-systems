resource "aws_cloudwatch_log_group" "ecs_logs" {
  name              = var.log_group_name
  retention_in_days = var.retention_in_days

  tags = merge(
    var.common_tags,
    {
      Name = var.log_group_name
    }
  )
}
