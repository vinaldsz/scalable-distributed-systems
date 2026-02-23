variable "service_name" {
  type        = string
  description = "Name of the service"
}

variable "retention_in_days" {
  type        = number
  default     = 7
  description = "CloudWatch log retention in days"
}
