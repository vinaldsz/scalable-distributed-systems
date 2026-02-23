variable "aws_region" {
  type        = string
  default     = "us-west-2"
  description = "AWS region"
}

variable "service_name" {
  type        = string
  default     = "product-search"
  description = "Name of the service"
}

variable "container_port" {
  type        = number
  default     = 8080
  description = "Container port"
}

variable "ecr_repository_name" {
  type        = string
  default     = "product-search"
  description = "ECR repository name"
}

variable "log_retention_days" {
  type        = number
  default     = 7
  description = "CloudWatch log retention in days"
}

variable "ecs_count" {
  type        = number
  default     = 1
  description = "Number of ECS tasks"
}
