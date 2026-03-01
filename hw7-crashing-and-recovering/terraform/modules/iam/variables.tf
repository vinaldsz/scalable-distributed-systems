variable "execution_role_name" {
  description = "Name of the ECS task execution role"
  type        = string
}

variable "task_role_name" {
  description = "Name of the ECS task role"
  type        = string
}

variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default     = {}
}
