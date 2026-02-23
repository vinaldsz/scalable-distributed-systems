output "autoscaling_target_id" {
  value       = aws_appautoscaling_target.ecs.id
  description = "ID of the autoscaling target"
}

output "cpu_policy_arn" {
  value       = aws_appautoscaling_policy.cpu.arn
  description = "ARN of the CPU-based autoscaling policy"
}

output "memory_policy_arn" {
  value       = aws_appautoscaling_policy.memory.arn
  description = "ARN of the memory-based autoscaling policy"
}
