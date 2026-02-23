variable "service_name" {
  type        = string
  description = "Name of the service for resource naming"
}

variable "cluster_name" {
  type        = string
  description = "Name of the ECS cluster"
}

variable "service_name_ecs" {
  type        = string
  description = "Name of the ECS service"
}

variable "min_capacity" {
  type        = number
  default     = 2
  description = "Minimum number of tasks (start with capacity for baseline test)"
}

variable "max_capacity" {
  type        = number
  default     = 4
  description = "Maximum number of tasks (cap cost while handling breaking point)"
}

variable "cpu_target" {
  type        = number
  default     = 70.0
  description = "Target CPU utilization percentage for auto-scaling"
}

variable "memory_target" {
  type        = number
  default     = 80.0
  description = "Target memory utilization percentage for auto-scaling"
}

variable "scale_out_cooldown" {
  type        = number
  default     = 300
  description = "Cooldown period (seconds) after scaling out"
}

variable "scale_in_cooldown" {
  type        = number
  default     = 300
  description = "Cooldown period (seconds) after scaling in"
}
