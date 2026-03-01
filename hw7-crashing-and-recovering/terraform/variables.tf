variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "app_version" {
  description = "Application version: 'broken' or 'fixed'"
  type        = string
  default     = "broken"

  validation {
    condition     = contains(["broken", "fixed"], var.app_version)
    error_message = "Version must be either 'broken' or 'fixed'."
  }
}

# Note: VPC and Subnets are automatically fetched from your default VPC
# via the network module data sources. No manual configuration needed!

variable "task_count" {
  description = "Number of ECS tasks to run"
  type        = number
  default     = 2
}

variable "task_max_count" {
  description = "Maximum number of ECS tasks for auto-scaling"
  type        = number
  default     = 5
}

variable "task_cpu" {
  description = "CPU units for ECS task (256, 512, 1024, 2048, 4096)"
  type        = string
  default     = "512"
}

variable "task_memory" {
  description = "Memory in MB for ECS task"
  type        = string
  default     = "1024"
}
