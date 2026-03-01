variable "cluster_name" {
  description = "Name of the ECS cluster"
  type        = string
}

variable "service_name" {
  description = "Name of the ECS service"
  type        = string
}

variable "min_capacity" {
  description = "Minimum number of tasks"
  type        = number
  default     = 1
}

variable "max_capacity" {
  description = "Maximum number of tasks"
  type        = number
  default     = 5
}

variable "cpu_policy_name" {
  description = "Name of the CPU scaling policy"
  type        = string
}

variable "memory_policy_name" {
  description = "Name of the memory scaling policy"
  type        = string
}

variable "cpu_target_value" {
  description = "Target CPU utilization percentage"
  type        = number
  default     = 70.0
}

variable "memory_target_value" {
  description = "Target memory utilization percentage"
  type        = number
  default     = 80.0
}
