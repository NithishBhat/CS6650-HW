# Lambda function that subscribes directly to SNS (no SQS needed)
resource "aws_lambda_function" "order_processor" {
  count = var.enable_order_processor_lambda ? 1 : 0

  function_name = "${var.project_name}-order-processor"
  role          = data.aws_iam_role.lab.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  memory_size   = 512
  timeout       = 30

  filename         = "${path.module}/../lambda/function.zip"
  source_code_hash = var.enable_order_processor_lambda ? filebase64sha256("${path.module}/../lambda/function.zip") : null

  tags = { Name = "${var.project_name}-order-processor" }
}

# Allow SNS to invoke the Lambda function
resource "aws_lambda_permission" "sns_invoke" {
  count = var.enable_order_processor_lambda ? 1 : 0

  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.order_processor[0].function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.order_processing_events.arn
}

# Subscribe Lambda to the SNS topic
resource "aws_sns_topic_subscription" "lambda_subscription" {
  count = var.enable_order_processor_lambda ? 1 : 0

  topic_arn = aws_sns_topic.order_processing_events.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.order_processor[0].arn
}

# CloudWatch log group for Lambda
resource "aws_cloudwatch_log_group" "lambda_logs" {
  count = var.enable_order_processor_lambda ? 1 : 0

  name              = "/aws/lambda/${aws_lambda_function.order_processor[0].function_name}"
  retention_in_days = 7

  tags = { Name = "${var.project_name}-lambda-logs" }
}

# LabRole data source (reuse from ECS module scope)
data "aws_iam_role" "lab" {
  name = "LabRole"
}
