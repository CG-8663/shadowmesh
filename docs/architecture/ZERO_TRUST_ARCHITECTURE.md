# Zero-Trust Network Architecture for ShadowMesh

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="80" align="right"/>

**Chronara Group ShadowMesh - Zero-Trust Architecture**

## Highly Available, Secure Cloud Infrastructure

---

## Executive Summary

This document outlines a zero-trust security architecture for a highly available application handling sensitive customer data. The design implements defense-in-depth with network segmentation, end-to-end encryption, identity-based access control, and comprehensive monitoring.

**Key Principles:**
- Never trust, always verify
- Assume breach mentality
- Verify explicitly
- Use least privilege access
- Encrypt everything

---

## Architecture Overview

```
                        ┌─────────────────────────────────────┐
                        │     External Users/Internet         │
                        └──────────────┬──────────────────────┘
                                       │
                        ┌──────────────▼──────────────────────┐
                        │   CDN/WAF (Cloudflare/CloudFront)   │
                        │   - DDoS Protection                 │
                        │   - TLS Termination                 │
                        │   - Bot Detection                   │
                        └──────────────┬──────────────────────┘
                                       │
                        ┌──────────────▼──────────────────────┐
                        │     Load Balancer (ALB/NLB)         │
                        │     - TLS 1.3                       │
                        │     - Mutual TLS (mTLS)             │
                        └──────────────┬──────────────────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              │                   VPC/VNet                       │
              │         CIDR: 10.0.0.0/16                       │
              │                                                  │
              │  ┌──────────────────────────────────────────┐  │
              │  │        DMZ Subnet (Public)               │  │
              │  │        CIDR: 10.0.1.0/24                 │  │
              │  │                                          │  │
              │  │  ┌──────────────────────────────────┐   │  │
              │  │  │    Web Tier (Stateless)          │   │  │
              │  │  │    - NGINX/Envoy Proxy           │   │  │
              │  │  │    - WAF Rules                   │   │  │
              │  │  │    - Rate Limiting               │   │  │
              │  │  └──────────────┬───────────────────┘   │  │
              │  └─────────────────┼───────────────────────┘  │
              │                    │                           │
              │  ┌─────────────────▼───────────────────────┐  │
              │  │     Application Subnet (Private)        │  │
              │  │     CIDR: 10.0.10.0/24                  │  │
              │  │                                         │  │
              │  │  ┌──────────────────────────────────┐  │  │
              │  │  │   Application Tier               │  │  │
              │  │  │   - Microservices (K8s/ECS)      │  │  │
              │  │  │   - Service Mesh (Istio/Linkerd) │  │  │
              │  │  │   - mTLS Between Services        │  │  │
              │  │  └──────────────┬───────────────────┘  │  │
              │  └─────────────────┼───────────────────────┘  │
              │                    │                           │
              │  ┌─────────────────▼───────────────────────┐  │
              │  │     Data Subnet (Isolated)              │  │
              │  │     CIDR: 10.0.20.0/24                  │  │
              │  │                                         │  │
              │  │  ┌──────────────────────────────────┐  │  │
              │  │  │   Database Tier                  │  │  │
              │  │  │   - RDS with encryption          │  │  │
              │  │  │   - Private endpoints only       │  │  │
              │  │  │   - IAM database authentication  │  │  │
              │  │  └──────────────────────────────────┘  │  │
              │  └─────────────────────────────────────────┘  │
              │                                                │
              │  ┌─────────────────────────────────────────┐  │
              │  │   Management Subnet (Bastion)           │  │
              │  │   CIDR: 10.0.100.0/24                   │  │
              │  │   - Bastion Hosts (Session Manager)     │  │
              │  │   - Jump Servers with MFA               │  │
              │  └─────────────────────────────────────────┘  │
              │                                                │
              └────────────────────────────────────────────────┘
                                    │
                        ┌───────────▼──────────────┐
                        │   Security Services      │
                        │   - SIEM/SOAR           │
                        │   - Threat Intelligence  │
                        │   - Security Monitoring  │
                        └──────────────────────────┘
```

---

## Network Segmentation

### Security Zones

#### 1. Public Zone (DMZ)
**Purpose:** Internet-facing services  
**CIDR:** 10.0.1.0/24  
**Components:**
- Web servers / Reverse proxies
- API gateways
- Load balancers (backend)

**Security Controls:**
- Security Group: Allow inbound 443 (HTTPS) only
- NACL: Deny all except required ports
- DDoS protection enabled
- WAF with OWASP rules

**Access:**
- Inbound: Internet (443)
- Outbound: Application tier (8080), Secrets Manager (443)

#### 2. Application Zone (Private)
**Purpose:** Business logic and services  
**CIDR:** 10.0.10.0/24  
**Components:**
- Microservices
- API services
- Message queues
- Cache layers (Redis/Memcached)

**Security Controls:**
- No direct internet access
- NAT Gateway for outbound updates
- Security Group: Allow only from web tier and service mesh
- mTLS required between all services
- Service accounts with pod identity

**Access:**
- Inbound: Web tier only
- Outbound: Database tier, External APIs (via NAT)

#### 3. Data Zone (Isolated)
**Purpose:** Data storage and databases  
**CIDR:** 10.0.20.0/24  
**Components:**
- RDS databases
- DynamoDB
- ElastiCache
- S3 VPC endpoints

**Security Controls:**
- Completely isolated from internet
- Private endpoints only
- Encryption at rest (KMS)
- Encryption in transit (TLS 1.3)
- Database firewall rules
- IAM authentication required

**Access:**
- Inbound: Application tier only
- Outbound: None (isolated)

#### 4. Management Zone (Bastion)
**Purpose:** Administrative access  
**CIDR:** 10.0.100.0/24  
**Components:**
- Bastion hosts
- Jump servers
- Monitoring agents

**Security Controls:**
- MFA required
- Session Manager (no SSH keys)
- All sessions logged and recorded
- Time-based access controls
- IP whitelist only

---

## Encryption Strategy

### Data at Rest

#### AWS Implementation:
```hcl
# RDS with KMS encryption
resource "aws_db_instance" "main" {
  identifier              = "shadowmesh-db"
  engine                  = "postgres"
  storage_encrypted       = true
  kms_key_id             = aws_kms_key.rds.arn
  
  # Enable encryption for automated backups
  backup_retention_period = 7
  
  # Enable encryption for replicas
  replicate_source_db    = null
}

# KMS key with rotation
resource "aws_kms_key" "rds" {
  description             = "RDS database encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = { AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root" }
        Action = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow RDS to use the key"
        Effect = "Allow"
        Principal = { Service = "rds.amazonaws.com" }
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:CreateGrant"
        ]
        Resource = "*"
      }
    ]
  })
}

# DynamoDB with KMS
resource "aws_dynamodb_table" "sessions" {
  name           = "user-sessions"
  billing_mode   = "PAY_PER_REQUEST"
  
  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.dynamodb.arn
  }
  
  point_in_time_recovery {
    enabled = true
  }
}

# S3 with SSE-KMS (from previous document)
resource "aws_s3_bucket_server_side_encryption_configuration" "main" {
  bucket = aws_s3_bucket.data.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3.arn
    }
    bucket_key_enabled = true
  }
}

# EBS volumes encrypted by default
resource "aws_ebs_encryption_by_default" "enabled" {
  enabled = true
}

resource "aws_ebs_default_kms_key" "main" {
  key_arn = aws_kms_key.ebs.arn
}
```

#### Key Management:
- **Customer-Managed Keys (CMK):** All KMS keys
- **Automatic Rotation:** Enabled (yearly)
- **Key Policies:** Least privilege access
- **Audit Logging:** All key usage logged to CloudTrail
- **Backup Keys:** Encrypted with separate master key

### Data in Transit

#### TLS 1.3 Everywhere:
```hcl
# ALB with TLS 1.3 only
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = aws_acm_certificate.main.arn
  
  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.web.arn
  }
}

# Force HTTPS redirect
resource "aws_lb_listener" "http_redirect" {
  load_balancer_arn = aws_lb.main.arn
  port              = "80"
  protocol          = "HTTP"
  
  default_action {
    type = "redirect"
    redirect {
      protocol    = "HTTPS"
      port        = "443"
      status_code = "HTTP_301"
    }
  }
}

# RDS with SSL required
resource "aws_db_instance" "main" {
  # ... other config
  
  # Force SSL connections
  parameter_group_name = aws_db_parameter_group.ssl_required.name
}

resource "aws_db_parameter_group" "ssl_required" {
  name   = "ssl-required"
  family = "postgres14"
  
  parameter {
    name  = "rds.force_ssl"
    value = "1"
  }
}
```

#### Service Mesh (mTLS):
```yaml
# Istio configuration for mutual TLS
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: production
spec:
  mtls:
    mode: STRICT

---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: mtls-for-all
spec:
  host: "*.production.svc.cluster.local"
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

#### VPN for Internal Networks:
- IPsec with IKEv2 (as documented in SITE_TO_SITE_VPN_CONFIG.md)
- AES-256-GCM encryption
- Perfect Forward Secrecy enabled
- Certificate-based authentication

---

## Identity and Access Management (IAM)

### Principle of Least Privilege

#### Service Accounts:
```hcl
# Application service role
resource "aws_iam_role" "app_service" {
  name = "app-service-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

# Inline policy with minimal permissions
resource "aws_iam_role_policy" "app_service" {
  name = "app-service-policy"
  role = aws_iam_role.app_service.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ReadSecrets"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          "arn:aws:secretsmanager:*:*:secret:app/*"
        ]
        Condition = {
          StringEquals = {
            "aws:RequestedRegion" = var.aws_region
          }
        }
      },
      {
        Sid    = "WriteMetrics"
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "ShadowMesh/Application"
          }
        }
      },
      {
        Sid    = "DatabaseAccess"
        Effect = "Allow"
        Action = [
          "rds-db:connect"
        ]
        Resource = [
          "arn:aws:rds-db:${var.aws_region}:${data.aws_caller_identity.current.account_id}:dbuser:*/app_user"
        ]
      }
    ]
  })
}
```

#### User Access:
```hcl
# Developer role with MFA required
resource "aws_iam_role" "developer" {
  name = "developer-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
      }
      Condition = {
        Bool = {
          "aws:MultiFactorAuthPresent" = "true"
        }
        NumericLessThan = {
          "aws:MultiFactorAuthAge" = "3600"  # 1 hour
        }
      }
    }]
  })
}

# Developer permissions (read-only production)
resource "aws_iam_role_policy" "developer" {
  name = "developer-policy"
  role = aws_iam_role.developer.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ReadOnlyProduction"
        Effect = "Allow"
        Action = [
          "ec2:Describe*",
          "rds:Describe*",
          "logs:GetLogEvents",
          "logs:FilterLogEvents",
          "cloudwatch:GetMetricData"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/Environment" = "production"
          }
        }
      },
      {
        Sid    = "DenyDestructiveActions"
        Effect = "Deny"
        Action = [
          "*:Delete*",
          "*:Terminate*",
          "*:Remove*"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/Environment" = "production"
          }
        }
      }
    ]
  })
}
```

### Zero-Trust Access (ZTA)

#### Session Manager (No SSH Keys):
```hcl
# EC2 instance role for Session Manager
resource "aws_iam_role" "instance" {
  name = "instance-ssm-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# SSM document for secure access
resource "aws_ssm_document" "session_manager_prefs" {
  name            = "SSM-SessionManagerRunShell"
  document_type   = "Session"
  document_format = "JSON"
  
  content = jsonencode({
    schemaVersion = "1.0"
    description   = "Document to hold regional settings for Session Manager"
    sessionType   = "Standard_Stream"
    inputs = {
      s3BucketName                = aws_s3_bucket.session_logs.id
      s3KeyPrefix                 = "session-logs/"
      s3EncryptionEnabled         = true
      cloudWatchLogGroupName      = aws_cloudwatch_log_group.sessions.name
      cloudWatchEncryptionEnabled = true
      kmsKeyId                    = aws_kms_key.sessions.id
      runAsEnabled                = false
      runAsDefaultUser            = ""
    }
  })
}
```

#### Conditional Access:
```hcl
# Policy with IP and time conditions
resource "aws_iam_policy" "conditional_access" {
  name = "conditional-access-policy"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowFromCorporateIP"
        Effect = "Allow"
        Action = "*"
        Resource = "*"
        Condition = {
          IpAddress = {
            "aws:SourceIp" = [
              "203.0.113.0/24",  # Corporate network
              "198.51.100.0/24"  # VPN network
            ]
          }
        }
      },
      {
        Sid    = "AllowDuringBusinessHours"
        Effect = "Allow"
        Action = "ec2:*"
        Resource = "*"
        Condition = {
          DateGreaterThan = {
            "aws:CurrentTime" = "2024-01-01T09:00:00Z"
          }
          DateLessThan = {
            "aws:CurrentTime" = "2024-01-01T17:00:00Z"
          }
        }
      }
    ]
  })
}
```

---

## Logging and Monitoring

### Comprehensive Audit Trail

#### CloudTrail (AWS):
```hcl
resource "aws_cloudtrail" "main" {
  name                          = "shadowmesh-trail"
  s3_bucket_name                = aws_s3_bucket.cloudtrail.id
  include_global_service_events = true
  is_multi_region_trail         = true
  enable_log_file_validation    = true
  
  event_selector {
    read_write_type           = "All"
    include_management_events = true
    
    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::*/"]
    }
    
    data_resource {
      type   = "AWS::Lambda::Function"
      values = ["arn:aws:lambda:*:*:function/*"]
    }
  }
  
  insight_selector {
    insight_type = "ApiCallRateInsight"
  }
  
  advanced_event_selector {
    name = "Log sensitive data access"
    
    field_selector {
      field  = "eventCategory"
      equals = ["Data"]
    }
    
    field_selector {
      field  = "resources.type"
      equals = ["AWS::S3::Object"]
    }
  }
}
```

#### VPC Flow Logs:
```hcl
resource "aws_flow_log" "main" {
  vpc_id          = aws_vpc.main.id
  traffic_type    = "ALL"
  iam_role_arn    = aws_iam_role.flow_logs.arn
  log_destination = aws_cloudwatch_log_group.flow_logs.arn
  
  tags = {
    Name = "vpc-flow-logs"
  }
}

resource "aws_cloudwatch_log_group" "flow_logs" {
  name              = "/aws/vpc/flow-logs"
  retention_in_days = 30
  kms_key_id        = aws_kms_key.logs.arn
}
```

#### Application Logging:
```yaml
# Fluent Bit configuration for container logs
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush         5
        Log_Level     info
        Parsers_File  parsers.conf

    [INPUT]
        Name              tail
        Path              /var/log/containers/*.log
        Parser            docker
        Tag               kube.*
        Refresh_Interval  5

    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL            https://kubernetes.default.svc:443
        Merge_Log           On

    [FILTER]
        Name    grep
        Match   *
        Exclude log (health|metrics)

    [OUTPUT]
        Name   cloudwatch_logs
        Match  *
        region us-east-1
        log_group_name /aws/eks/application
        auto_create_group true
```

### Security Monitoring

#### AWS GuardDuty:
```hcl
resource "aws_guardduty_detector" "main" {
  enable = true
  
  datasources {
    s3_logs {
      enable = true
    }
    kubernetes {
      audit_logs {
        enable = true
      }
    }
    malware_protection {
      scan_ec2_instance_with_findings {
        ebs_volumes {
          enable = true
        }
      }
    }
  }
}

resource "aws_guardduty_publishing_destination" "main" {
  detector_id     = aws_guardduty_detector.main.id
  destination_arn = aws_s3_bucket.findings.arn
  kms_key_arn     = aws_kms_key.guardduty.arn
  
  destination_type = "S3"
}
```

#### AWS Security Hub:
```hcl
resource "aws_securityhub_account" "main" {}

resource "aws_securityhub_standards_subscription" "cis" {
  standards_arn = "arn:aws:securityhub:${var.aws_region}::standards/cis-aws-foundations-benchmark/v/1.4.0"
}

resource "aws_securityhub_standards_subscription" "pci" {
  standards_arn = "arn:aws:securityhub:${var.aws_region}::standards/pci-dss/v/3.2.1"
}

resource "aws_securityhub_finding_aggregator" "main" {
  linking_mode = "ALL_REGIONS"
}
```

---

## Summary of Key Security Controls

### Network Layer
- [x] Network segmentation (4 zones)
- [x] Private subnets for sensitive workloads
- [x] Security groups with least privilege
- [x] NACLs for defense in depth
- [x] No direct internet access to data tier

### Encryption
- [x] TLS 1.3 for all connections
- [x] mTLS between microservices
- [x] KMS encryption for data at rest
- [x] Customer-managed encryption keys
- [x] Automatic key rotation

### Identity
- [x] IAM roles with least privilege
- [x] MFA required for humans
- [x] Service accounts for applications
- [x] Temporary credentials only
- [x] No long-lived access keys

### Monitoring
- [x] CloudTrail for API logging
- [x] VPC Flow Logs for network traffic
- [x] Application logs centralized
- [x] GuardDuty for threat detection
- [x] Security Hub for compliance

### Compliance
- [x] CIS AWS Foundations Benchmark
- [x] PCI DSS controls
- [x] HIPAA controls (if needed)
- [x] SOC 2 Type II ready
- [x] GDPR data protection

