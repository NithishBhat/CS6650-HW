# CS 6650 – Scalable Distributed Systems

This is my coursework from CS 6650, a graduate course on distributed systems at Northeastern University. Over the semester I built small backend services, put them on AWS, and then hammered them with simulated traffic to see how they behaved: where they slowed down, where they broke, and what it cost to fix that. Each folder is one assignment, usually with a short write-up of the measured results.

The piece I'd point to first is the Album Store, a photo-album web service that passed every test of the course's automated load-testing grader. It lives here in `Final_mastery/` and also as its own repo: [NithishBhat/album-store](https://github.com/NithishBhat/album-store).

Most of the code is Go, with Python for load tests and analysis. Infrastructure is Terraform and Docker on AWS (EC2, ECS Fargate, ALB, S3, DynamoDB, RDS MySQL, SNS/SQS, Lambda), and load testing is done with Locust.

## Assignments

| Folder | What it is |
|--------|------------|
| [`HW1`](HW1) | A small Go album API in Docker on EC2, plus a Python script that load tests it and plots response times. |
| [`HW2`](HW2) | EC2 setup with Terraform; a demo of two copies of a service drifting apart when each keeps its own in-memory state; a Go Lambda and a race condition. |
| [`HW3`](HW3) | Go concurrency experiments (atomics, mutexes, `sync.Map`, buffered vs. unbuffered file writes, context switching) and Locust tests looking at Amdahl's law and GET vs. POST. |
| [`HW4`](HW4) | MapReduce word count split into three Go services (splitter, mapper, reducer) on ECS Fargate, using S3 between stages. Run on *Hamlet*. |
| [`HW5`](HW5) | Product API in Go with a thread-safe in-memory store, deployed with Terraform and load tested with Locust. |
| [`HW6`](HW6) | Product search over 100,000 generated products behind a load balancer, tested with a search-heavy workload. |
| [`HW7`](HW7) | Order processing done synchronously vs. through queues (SNS/SQS with an ECS worker pool, and with Lambda). Compares queue depth, worker scaling, cold starts and cost. |
| [`CS6650_HW8_Team`](CS6650_HW8_Team) | Team project: the same shopping-cart service built on MySQL (RDS) and on DynamoDB, with Terraform modules and Python tests for performance and consistency. |
| [`HW9`](HW9) | Submodule pointer only; no files here. |
| [`HW10`](HW10) | Replicated key-value store in Go, run as leader-follower and leaderless clusters with tunable read/write quorums. Measures latency and stale reads at different write ratios. |
| [`Midterm_Mastery`](Midterm_Mastery) | Tracking down a deadlock from a lock that was never released, fixing it with `TryLock` and bulkheads, with before/after load-test numbers. |
| [`Midterm_mystery`](Midterm_mystery) | Debugging an existing media service: notes on four bugs found and fixed. |
| [`Final_mastery`](Final_mastery) | The Album Store: Go service for albums and photo uploads on DynamoDB and S3, running on ECS Fargate behind a load balancer. Includes the Terraform, a deploy script, tests, and logs from six benchmark runs. Also at [album-store](https://github.com/NithishBhat/album-store). |
| [`Final_Project`](Final_Project) | Submodule pointer only; no files here. |

## Running things

Each folder stands on its own. The usual pattern:

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

`HW10` runs locally with Docker Compose (`docker compose -f docker-compose-leader.yml up`, or `docker-compose-leaderless.yml`). `Final_mastery/deploy.sh` and `HW7/code/deploy-script.sh` build the image, push it to ECR and run Terraform.
