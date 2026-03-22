resource "aws_sns_topic" "order_processing_events" {
  name = "${var.project_name}-order-processing-events"
}

resource "aws_sqs_queue" "order_processing_queue" {
  name                       = "${var.project_name}-order-processing-queue"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 4 * 24 * 60 * 60
  receive_wait_time_seconds  = 20
}

resource "aws_sns_topic_subscription" "order_processing_subscription" {
  topic_arn            = aws_sns_topic.order_processing_events.arn
  protocol             = "sqs"
  endpoint             = aws_sqs_queue.order_processing_queue.arn
  raw_message_delivery = true
}

resource "aws_sqs_queue_policy" "order_processing_queue_policy" {
  queue_url = aws_sqs_queue.order_processing_queue.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "*" }
      Action    = "SQS:SendMessage"
      Resource  = aws_sqs_queue.order_processing_queue.arn
      Condition = {
        ArnEquals = {
          "aws:SourceArn" = aws_sns_topic.order_processing_events.arn
        }
      }
    }]
  })
}