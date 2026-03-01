variable "alb_sg_name" {
  description = "Name of the ALB security group"
  type        = string
}

variable "ecs_sg_name" {
  description = "Name of the ECS tasks security group"
  type        = string
}

variable "container_port" {
  description = "Container port"
  type        = number
  default     = 8080
}

variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default     = {}
}
