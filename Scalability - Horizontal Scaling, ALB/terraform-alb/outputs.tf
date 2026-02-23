output "ecs_cluster_name" {
  description = "Name of the created ECS cluster"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "Name of the running ECS service"
  value       = module.ecs.service_name
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = module.alb.alb_dns_name
}

output "alb_url" {
  description = "Full URL to access the service via ALB"
  value       = "http://${module.alb.alb_dns_name}"
}

output "target_group_arn" {
  description = "ARN of the target group"
  value       = module.alb.target_group_arn
}
