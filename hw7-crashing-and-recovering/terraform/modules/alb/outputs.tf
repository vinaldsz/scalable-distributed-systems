output "alb_arn" {
  description = "ARN of the ALB"
  value       = aws_lb.main.arn
}

output "alb_dns_name" {
  description = "DNS name of the ALB"
  value       = aws_lb.main.dns_name
}

output "alb_arn_suffix" {
  description = "ARN suffix of the ALB for use with CloudWatch"
  value       = aws_lb.main.arn_suffix
}

output "target_group_arn" {
  description = "ARN of the target group"
  value       = aws_lb_target_group.search_service.arn
}

output "target_group_arn_suffix" {
  description = "ARN suffix of the target group for use with CloudWatch"
  value       = aws_lb_target_group.search_service.arn_suffix
}

output "alb_listener_arn" {
  description = "ARN of the ALB listener"
  value       = aws_lb_listener.http.arn
}
