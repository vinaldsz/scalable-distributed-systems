output "vpc_id" {
  description = "ID of the default VPC"
  value       = data.aws_vpc.default.id
}

output "subnet_ids" {
  description = "List of subnet IDs in the default VPC"
  value       = data.aws_subnets.default.ids
}

output "alb_security_group_id" {
  description = "ID of the ALB security group"
  value       = aws_security_group.alb.id
}

output "ecs_security_group_id" {
  description = "ID of the ECS tasks security group"
  value       = aws_security_group.ecs_tasks.id
}
