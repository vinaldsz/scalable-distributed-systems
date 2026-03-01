variable "cluster_name" {
  description = "Name of the ECS cluster"
  type        = string
}

variable "task_family" {
  description = "Name of the ECS task definition family"
  type        = string
}

variable "task_cpu" {
  description = "CPU units for the task (256, 512, 1024, etc.)"
  type        = string
  default     = "512"
}

variable "task_memory" {
  description = "Memory for the task in MB (512, 1024, 2048, etc.)"
  type        = string
  default     = "1024"
}

variable "task_count" {
  description = "Initial number of ECS tasks"
  type        = number
  default     = 1
}

variable "execution_role_arn" {
  description = "ARN of the ECS task execution role"
  type        = string
}

variable "task_role_arn" {
  description = "ARN of the ECS task role"
  type        = string
}

variable "container_name" {
  description = "Name of the container"
  type        = string
}

variable "container_image" {
  description = "Docker image URI for the container"
  type        = string
}

variable "container_port" {
  description = "Port exposed by the container"
  type        = number
  default     = 8080
}

variable "container_environment" {
  description = "Environment variables for the container"
  type = list(object({
    name  = string
    value = string
  }))
  default = []
}

variable "log_group_name" {
  description = "CloudWatch log group name"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "ecs_security_group_id" {
  description = "Security group ID for ECS tasks"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs for ECS tasks"
  type        = list(string)
}

variable "target_group_arn" {
  description = "ARN of the target group"
  type        = string
}

variable "service_name" {
  description = "Name of the ECS service"
  type        = string
}

variable "alb_listener_dependency" {
  description = "ARN of the ALB listener to depend on"
  type        = string
}

variable "app_version" {
  description = "Application version (broken or fixed)"
  type        = string
  default     = "broken"
}

variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default     = {}
}
