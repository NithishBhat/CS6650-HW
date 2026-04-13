variable "region" {
  default = "us-west-2"
}

variable "prefix" {
  default = "album-store"
}

variable "vpc_id" {
  default = "vpc-0122801c190321a21"
}

variable "subnet_ids" {
  default = [
    "subnet-0a2ed0352058ea7b7",
    "subnet-095621e9587cfcf02",
  ]
}

variable "cpu" {
  default = "2048"
}

variable "memory" {
  default = "4096"
}

variable "desired_count" {
  default = 2
}
