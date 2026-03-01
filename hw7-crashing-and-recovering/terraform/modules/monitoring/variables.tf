variable "health_alarm_name" {
  description = "Name of the health check alarm"
  type        = string
}

variable "alb_arn_suffix" {
  description = "ARN suffix of the ALB"
  type        = string
}

variable "target_group_arn_suffix" {
  description = "ARN suffix of the target group"
  type        = string
}

variable "dashboard_name" {
  description = "Name of the CloudWatch dashboard"
  type        = string
}

variable "dashboard_title" {
  description = "Title displayed in the CloudWatch dashboard"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default     = {}
}
