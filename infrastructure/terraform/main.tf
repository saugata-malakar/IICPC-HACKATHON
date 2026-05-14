# ─────────────────────────────────────────────────────────────────────────────
# IICPC Platform v2 — Terraform Infrastructure as Code
# Provider: AWS (EKS + MSK + ElastiCache + RDS)
# ─────────────────────────────────────────────────────────────────────────────

terraform {
  required_version = ">= 1.7.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
  }

  backend "s3" {
    bucket = "iicpc-terraform-state"
    key    = "platform/terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project     = "IICPC-Platform-v2"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# ─── Variables ────────────────────────────────────────────────────────────────

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
  default     = "iicpc-platform-cluster"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

# ─── VPC ──────────────────────────────────────────────────────────────────────

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.cluster_name}-vpc"
  cidr = var.vpc_cidr

  azs             = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = false
  enable_dns_hostnames = true
  enable_dns_support   = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = "1"
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = "1"
  }
}

# ─── EKS Cluster ──────────────────────────────────────────────────────────────

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.29"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  cluster_endpoint_public_access = true

  # Enable IRSA (IAM Roles for Service Accounts)
  enable_irsa = true

  # Cluster addons
  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
    aws-ebs-csi-driver = {
      most_recent = true
    }
  }

  # Node groups
  eks_managed_node_groups = {
    # General purpose nodes
    general = {
      name           = "general-purpose"
      instance_types = ["m6i.2xlarge"]
      
      min_size     = 3
      max_size     = 10
      desired_size = 5

      disk_size = 100

      labels = {
        role = "general"
      }

      tags = {
        NodeGroup = "general-purpose"
      }
    }

    # Bot fleet nodes (CPU-optimized)
    bot_fleet = {
      name           = "bot-fleet"
      instance_types = ["c6i.4xlarge"]
      
      min_size     = 2
      max_size     = 20
      desired_size = 4

      disk_size = 100

      labels = {
        role = "bot-fleet"
      }

      taints = [{
        key    = "workload"
        value  = "bot-fleet"
        effect = "NoSchedule"
      }]

      tags = {
        NodeGroup = "bot-fleet"
      }
    }

    # Sandbox nodes (gVisor-enabled)
    sandbox = {
      name           = "sandbox"
      instance_types = ["m6i.xlarge"]
      
      min_size     = 2
      max_size     = 10
      desired_size = 3

      disk_size = 50

      labels = {
        role = "sandbox"
      }

      taints = [{
        key    = "workload"
        value  = "sandbox"
        effect = "NoSchedule"
      }]

      tags = {
        NodeGroup = "sandbox"
      }
    }
  }
}

# ─── Amazon MSK (Kafka) ───────────────────────────────────────────────────────

resource "aws_msk_cluster" "kafka" {
  cluster_name           = "${var.cluster_name}-kafka"
  kafka_version          = "3.6.0"
  number_of_broker_nodes = 3

  broker_node_group_info {
    instance_type   = "kafka.m5.xlarge"
    client_subnets  = module.vpc.private_subnets
    security_groups = [aws_security_group.kafka.id]

    storage_info {
      ebs_storage_info {
        volume_size            = 500
        provisioned_throughput {
          enabled           = true
          volume_throughput = 250
        }
      }
    }
  }

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  configuration_info {
    arn      = aws_msk_configuration.kafka_config.arn
    revision = aws_msk_configuration.kafka_config.latest_revision
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.kafka.name
      }
    }
  }

  tags = {
    Name = "${var.cluster_name}-kafka"
  }
}

resource "aws_msk_configuration" "kafka_config" {
  name              = "${var.cluster_name}-kafka-config"
  kafka_versions    = ["3.6.0"]
  server_properties = <<PROPERTIES
auto.create.topics.enable=true
default.replication.factor=3
min.insync.replicas=2
num.partitions=12
log.retention.hours=24
compression.type=snappy
PROPERTIES
}

resource "aws_security_group" "kafka" {
  name        = "${var.cluster_name}-kafka-sg"
  description = "Security group for MSK cluster"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 9092
    to_port     = 9092
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_cloudwatch_log_group" "kafka" {
  name              = "/aws/msk/${var.cluster_name}"
  retention_in_days = 7
}

# ─── ElastiCache Redis ────────────────────────────────────────────────────────

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "${var.cluster_name}-redis"
  replication_group_description = "Redis cluster for IICPC platform"
  
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = "cache.r6g.xlarge"
  num_cache_clusters   = 3
  parameter_group_name = "default.redis7"
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.redis.name
  security_group_ids = [aws_security_group.redis.id]

  automatic_failover_enabled = true
  multi_az_enabled          = true

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true

  snapshot_retention_limit = 5
  snapshot_window         = "03:00-05:00"
  maintenance_window      = "sun:05:00-sun:07:00"

  tags = {
    Name = "${var.cluster_name}-redis"
  }
}

resource "aws_elasticache_subnet_group" "redis" {
  name       = "${var.cluster_name}-redis-subnet"
  subnet_ids = module.vpc.private_subnets
}

resource "aws_security_group" "redis" {
  name        = "${var.cluster_name}-redis-sg"
  description = "Security group for Redis cluster"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ─── RDS TimescaleDB ──────────────────────────────────────────────────────────

resource "aws_db_instance" "timescaledb" {
  identifier     = "${var.cluster_name}-timescaledb"
  engine         = "postgres"
  engine_version = "16.1"
  instance_class = "db.r6g.2xlarge"

  allocated_storage     = 500
  max_allocated_storage = 2000
  storage_type          = "io2"
  iops                  = 16000
  storage_encrypted     = true

  db_name  = "telemetry"
  username = "telemetry"
  password = random_password.db_password.result

  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.rds.name

  multi_az               = true
  publicly_accessible    = false
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "sun:04:00-sun:05:00"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  performance_insights_enabled = true

  tags = {
    Name = "${var.cluster_name}-timescaledb"
  }
}

resource "aws_db_subnet_group" "rds" {
  name       = "${var.cluster_name}-rds-subnet"
  subnet_ids = module.vpc.private_subnets
}

resource "aws_security_group" "rds" {
  name        = "${var.cluster_name}-rds-sg"
  description = "Security group for RDS instance"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "random_password" "db_password" {
  length  = 32
  special = true
}

# ─── Outputs ──────────────────────────────────────────────────────────────────

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "kafka_bootstrap_brokers" {
  description = "MSK bootstrap brokers"
  value       = aws_msk_cluster.kafka.bootstrap_brokers_tls
}

output "redis_endpoint" {
  description = "Redis primary endpoint"
  value       = aws_elasticache_replication_group.redis.primary_endpoint_address
}

output "timescaledb_endpoint" {
  description = "TimescaleDB endpoint"
  value       = aws_db_instance.timescaledb.endpoint
}

output "timescaledb_password" {
  description = "TimescaleDB password"
  value       = random_password.db_password.result
  sensitive   = true
}
