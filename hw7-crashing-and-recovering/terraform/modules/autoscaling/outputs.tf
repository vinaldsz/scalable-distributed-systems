output "autoscaling_target_id" {
  description = "ID of the autoscaling target"
  value       = aws_appautoscaling_target.ecs_service.id
}

output "cpu_policy_arn" {
  description = "ARN of the CPU scaling policy"
  value       = aws_appautoscaling_policy.ecs_cpu_scaling.arn
}

output "memory_policy_arn" {
  description = "ARN of the memory scaling policy"
  value       = aws_appautoscaling_policy.ecs_memory_scaling.arn
}
