output "alb_dns_name" {
  description = "DNS name of the ALB for accessing the search service"
  value       = module.alb.alb_dns_name
}

output "alb_url" {
  description = "URL to access the search service"
  value       = "http://${module.alb.alb_dns_name}"
}

output "ecr_repository_url" {
  description = "ECR repository URL for pushing Docker images"
  value       = module.ecr.repository_url
}

output "ecs_cluster_name" {
  description = "Name of the ECS cluster"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "Name of the ECS service"
  value       = module.ecs.service_name
}

output "cloudwatch_log_group" {
  description = "CloudWatch log group for ECS logs"
  value       = module.logging.log_group_name
}

output "cloudwatch_dashboard_url" {
  description = "URL to CloudWatch dashboard for monitoring"
  value       = "https://console.aws.amazon.com/cloudwatch/home?region=${var.aws_region}#dashboards:name=hw7-${var.app_version}"
}

output "target_group_arn" {
  description = "ARN of the target group"
  value       = module.alb.target_group_arn
}

output "deployment_info" {
  description = "Summary of deployment"
  value = {
    version              = var.app_version
    region               = var.aws_region
    cluster_name         = module.ecs.cluster_name
    service_name         = module.ecs.service_name
    alb_dns              = module.alb.alb_dns_name
    search_endpoint      = "http://${module.alb.alb_dns_name}/products/search?q=laptop"
    health_endpoint      = "http://${module.alb.alb_dns_name}/health"
    ecr_repository       = module.ecr.repository_url
    log_group            = module.logging.log_group_name
  }
}
