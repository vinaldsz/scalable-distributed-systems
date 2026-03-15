locals {
  name_prefix = "cost-savings-${var.environment}"
}

data "aws_sns_topic" "order_processing_events" {
  name = var.existing_sns_topic_name
}

data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${local.name_prefix}-order-processor"
  retention_in_days = 7
}

resource "aws_lambda_function" "order_processor" {
  function_name = "${local.name_prefix}-order-processor"
  role          = data.aws_iam_role.lab_role.arn
  package_type  = "Image"
  image_uri     = var.lambda_image

  memory_size                    = var.lambda_memory_size
  timeout                        = var.lambda_timeout_seconds
  reserved_concurrent_executions = var.lambda_reserved_concurrency

  depends_on = [
    aws_cloudwatch_log_group.lambda
  ]
}

resource "aws_lambda_permission" "allow_sns_invoke" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.order_processor.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = data.aws_sns_topic.order_processing_events.arn
}

resource "aws_sns_topic_subscription" "lambda_subscription" {
  topic_arn = data.aws_sns_topic.order_processing_events.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.order_processor.arn
}
