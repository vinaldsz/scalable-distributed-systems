output "alb_dns_name" {
  value = aws_lb.main.dns_name
}

output "sns_topic_arn" {
  value = aws_sns_topic.order_processing_events.arn
}

output "sqs_queue_url" {
  value = aws_sqs_queue.order_processing_queue.id
}

output "sqs_queue_arn" {
  value = aws_sqs_queue.order_processing_queue.arn
}
