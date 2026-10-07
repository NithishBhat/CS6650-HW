# CS 6650 – Scalable Distributed Systems (Northeastern University)

Coursework for CS 6650 Scalable Distributed Systems. Each folder is a self-contained assignment: Go services deployed to AWS with Terraform and Docker, load tested with Locust, and written up with measured results.

> **Highlight:** `Final_mastery/` is an Album Store service that scored **190/190** on the ChaosArena load-test leaderboard. It is also published as its own repo: [NithishBhat/album-store](https://github.com/NithishBhat/album-store).

## Tech Stack

- **Languages:** Go (net/http, Gin, gorilla/mux), Python
- **AWS:** EC2, ECS Fargate, ECR, ALB, S3, DynamoDB, RDS MySQL, SNS, SQS, Lambda, CloudWatch, VPC
- **Infrastructure / tooling:** Terraform, Docker, Docker Compose, Locust

## Assignments

| Folder | What it covers |
|--------|----------------|
| [`HW1`](HW1) | Gin REST API for albums, containerized with Docker and deployed to EC2; Python script that load tests the endpoint and plots the response-time distribution. |
| [`HW2`](HW2) | Terraform for EC2 instances and security groups; script showing that in-memory state diverges across two independently deployed instances; deploying a Go Lambda and analyzing a race condition. |
| [`HW3`](HW3) | Go concurrency experiments (atomic vs. non-atomic counters, Mutex, RWMutex, `sync.Map`, buffered vs. unbuffered file I/O, context switching) plus Locust load tests in Docker Compose (Amdahl's law, `HttpUser` vs. `FastHttpUser`, GET vs. POST). |
| [`HW4`](HW4) | MapReduce word count as three Go microservices (splitter, mapper, reducer) on ECS Fargate with S3 for intermediate and final results, run on *Hamlet*. |
| [`HW5`](HW5) | Product API in Go with an `RWMutex`-protected in-memory store, deployed to AWS using Terraform from the course demo repo (submodule) and load tested with Locust (`HttpUser` vs. `FastHttpUser`). |
| [`HW6`](HW6) | Product search service over 100,000 generated products in a `sync.Map` with bounded search, ALB health checks, and a Locust search-heavy workload. |
| [`HW7`](HW7) | Order processing comparing synchronous vs. asynchronous handling: API publishes to SNS, SQS-backed ECS worker pool vs. SNS-triggered Lambda, with queue-depth, worker-scaling, cold-start and cost analysis. All infrastructure in Terraform. |
| [`CS6650_HW8_Team`](CS6650_HW8_Team) | Team project: shopping-cart service implemented on both RDS MySQL (normalized schema, documented index design) and DynamoDB, with modular Terraform (network, ALB, ECS, ECR, RDS, logging), SNS/SQS/Lambda, and Python tests for performance and consistency. |
| [`HW9`](HW9) | Submodule pointer only; no files in this repo. |
| [`HW10`](HW10) | Replicated in-memory key-value store in Go, run as Leader-Follower and Leaderless clusters (Docker Compose) with configurable W/R quorums; Locust tests across write ratios measuring latency and stale reads, with generated graphs and a report. |
| [`Midterm_Mastery`](Midterm_Mastery) | Diagnosing a deadlock caused by a lock that was never released, then fixing it with fail-fast `TryLock` and bulkhead patterns; before/after load-test metrics. |
| [`Midterm_mystery`](Midterm_mystery) | Debugging an existing media service ("Hummingbird"): summary of four bug fixes (port fallback, missing metadata field, redirect URL, download redirect logic). |
| [`Final_mastery`](Final_mastery) | **Album Store** – Go REST service for albums and photo uploads backed by DynamoDB and S3, running on ECS Fargate behind an ALB, provisioned with Terraform and a deploy script. Includes Locust and verification tests and logs from six benchmark runs (EC2 instance sizes, Fargate, streaming and multipart uploads). Scored 190/190 on ChaosArena; see [album-store](https://github.com/NithishBhat/album-store). |

## Running an Assignment

Each folder runs on its own. Common patterns:

```bash
# Go service locally
cd <folder>/<service>
go run .

# AWS infrastructure (needs AWS credentials; creates billable resources)
cd <folder>/terraform
terraform init
terraform apply

# Load test
locust -f locustfile.py --host http://<service-host>
```

`HW10` runs locally with Docker Compose (`docker compose -f docker-compose-leader.yml up` or `docker-compose-leaderless.yml`). `Final_mastery/deploy.sh` and `HW7/code/deploy-script.sh` handle build, push to ECR, and Terraform apply for those projects.
