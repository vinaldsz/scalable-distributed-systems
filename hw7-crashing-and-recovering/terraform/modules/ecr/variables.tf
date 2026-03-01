variable "repository_name" {
  description = "Name of the ECR repository"
  type        = string
}

variable "force_delete" {
  description = "Delete repository even if it contains images"
  type        = bool
  default     = false
}

variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default     = {}
}
