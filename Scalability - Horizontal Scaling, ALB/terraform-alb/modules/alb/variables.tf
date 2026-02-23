variable "service_name" {
  type        = string
  description = "Name of the service for resource naming"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID where ALB will be created"
}

variable "subnet_ids" {
  type        = list(string)
  description = "List of subnet IDs for ALB (must be in different AZs)"
}

variable "container_port" {
  type        = number
  description = "Port where container listens for traffic"
}
