# ecr module
module "ecr" {
  source = "./modules/ecr"

  project_name = var.project_name
}

# network module
module "network" {
  source = "./modules/network"

  project_name = var.project_name
}

# rds module
module "rds_mysql" {
  source = "./modules/rds-mysql"

  project_name          = var.project_name
  vpc_id                = module.network.vpc_id
  subnet_ids            = module.network.private_subnet_ids
  ecs_security_group_id = module.network.ecs_security_group_id

  db_name     = var.rds_db_name
  db_username = var.rds_db_username
  db_password = var.rds_db_password

  depends_on = [module.network]
}

# logging module
module "logging" {
  source = "./modules/logging"

  project_name = var.project_name
}

# alb module
module "alb" {
  source = "./modules/alb"

  project_name          = var.project_name
  vpc_id                = module.network.vpc_id
  subnet_ids            = module.network.public_subnet_ids
  alb_security_group_id = module.network.alb_security_group_id

  depends_on = [module.network]
}

# ecs module
module "ecs" {
  source = "./modules/ecs"

  project_name          = var.project_name
  aws_region            = var.aws_region
  image_tag             = var.image_tag
  ecr_cart_url          = module.ecr.cart_repository_url
  log_group_name        = module.logging.cart_log_group_name
  subnet_ids            = module.network.private_subnet_ids
  ecs_security_group_id = module.network.ecs_security_group_id
  desired_count         = var.cart_min_count
  min_count             = var.cart_min_count
  max_count             = var.cart_max_count
  target_group_arn      = module.alb.target_group_arn
  cpu_target_value      = var.cpu_target_value

  depends_on = [module.ecr, module.network, module.logging, module.alb, module.rds_mysql]

  worker_count               = var.worker_count
  order_processing_topic_arn = aws_sns_topic.order_processing_events.arn
  order_processing_queue_url = aws_sqs_queue.order_processing_queue.id

  db_host     = module.rds_mysql.db_endpoint
  db_port     = module.rds_mysql.db_port
  db_name     = module.rds_mysql.db_name
  db_username = var.rds_db_username
  db_password = var.rds_db_password

  dynamodb_table_name = aws_dynamodb_table.shopping_carts.name
}
