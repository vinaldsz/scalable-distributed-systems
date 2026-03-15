output "lambda_function_name" {
  value = aws_lambda_function.order_processor.function_name
}

output "lambda_function_arn" {
  value = aws_lambda_function.order_processor.arn
}

output "lambda_log_group_name" {
  value = aws_cloudwatch_log_group.lambda.name
}

output "sns_topic_arn" {
  value = data.aws_sns_topic.order_processing_events.arn
}
