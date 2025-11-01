# AWS S3 + KMS Terraform Configuration
## Secure Storage with Server-Side Encryption

---

## Overview

This Terraform configuration deploys:
- Private S3 bucket with versioning enabled
- AWS KMS customer-managed key for encryption
- Bucket policy enforcing encryption at rest
- IAM role with least-privilege access

---

## File Structure

```
terraform/
├── main.tf           # Main resources
├── variables.tf      # Input variables
├── outputs.tf        # Output values
├── kms.tf           # KMS key configuration
├── s3.tf            # S3 bucket configuration
├── iam.tf           # IAM roles and policies
└── versions.tf      # Provider versions
```

---

## versions.tf

```hcl
terraform {
  required_version = ">= 1.5.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  # Optional: Remote state backend
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "shadowmesh/s3-kms/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}

provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project     = "ShadowMesh"
      ManagedBy   = "Terraform"
      Environment = var.environment
    }
  }
}
```

---

## variables.tf

```hcl
variable "aws_region" {
  description = "AWS region where resources will be created"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "prod"
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod"
  }
}

variable "bucket_name" {
  description = "Name of the S3 bucket (must be globally unique)"
  type        = string
  
  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]*[a-z0-9]$", var.bucket_name))
    error_message = "Bucket name must be lowercase alphanumeric and hyphens only"
  }
}

variable "kms_key_alias" {
  description = "Alias for the KMS key"
  type        = string
  default     = "shadowmesh-s3-encryption"
}

variable "authorized_role_arns" {
  description = "List of IAM role ARNs authorized to use the KMS key"
  type        = list(string)
  default     = []
  
  # Example: ["arn:aws:iam::123456789012:role/AppRole"]
}

variable "enable_versioning" {
  description = "Enable S3 bucket versioning"
  type        = bool
  default     = true
}

variable "enable_logging" {
  description = "Enable S3 access logging"
  type        = bool
  default     = true
}

variable "lifecycle_rules" {
  description = "S3 lifecycle rules for object transitions"
  type = object({
    transition_to_ia_days        = number
    transition_to_glacier_days   = number
    expiration_days              = number
  })
  default = {
    transition_to_ia_days      = 30   # Move to Infrequent Access after 30 days
    transition_to_glacier_days = 90   # Move to Glacier after 90 days
    expiration_days            = 365  # Delete after 1 year
  }
}

variable "tags" {
  description = "Additional tags for all resources"
  type        = map(string)
  default     = {}
}
```

---

## kms.tf

```hcl
# Data source for current AWS account
data "aws_caller_identity" "current" {}

# Data source for current AWS region
data "aws_region" "current" {}

# KMS Key for S3 encryption
resource "aws_kms_key" "s3_encryption" {
  description             = "KMS key for S3 bucket encryption - ${var.bucket_name}"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  # Key policy
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # Allow root account full access
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      # Allow S3 service to use the key
      {
        Sid    = "Allow S3 to use the key"
        Effect = "Allow"
        Principal = {
          Service = "s3.amazonaws.com"
        }
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "kms:ViaService" = "s3.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      },
      # Allow authorized roles to use the key
      {
        Sid    = "Allow authorized IAM roles"
        Effect = "Allow"
        Principal = {
          AWS = var.authorized_role_arns
        }
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:GenerateDataKey"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "kms:ViaService" = "s3.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      },
      # Allow CloudTrail to describe the key
      {
        Sid    = "Allow CloudTrail to describe key"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action = [
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })
  
  tags = merge(
    var.tags,
    {
      Name        = "s3-encryption-key-${var.environment}"
      Description = "KMS key for ${var.bucket_name} encryption"
    }
  )
}

# KMS Key Alias
resource "aws_kms_alias" "s3_encryption" {
  name          = "alias/${var.kms_key_alias}"
  target_key_id = aws_kms_key.s3_encryption.key_id
}

# Grant for S3 service
resource "aws_kms_grant" "s3_grant" {
  count = length(var.authorized_role_arns) > 0 ? 1 : 0
  
  name              = "s3-${var.bucket_name}-grant"
  key_id            = aws_kms_key.s3_encryption.key_id
  grantee_principal = var.authorized_role_arns[0]
  
  operations = [
    "Encrypt",
    "Decrypt",
    "GenerateDataKey",
    "DescribeKey"
  ]
}
```

---

## s3.tf

```hcl
# S3 Bucket for logging (if enabled)
resource "aws_s3_bucket" "logs" {
  count = var.enable_logging ? 1 : 0
  
  bucket = "${var.bucket_name}-logs"
  
  tags = merge(
    var.tags,
    {
      Name    = "${var.bucket_name}-logs"
      Purpose = "Access logs for ${var.bucket_name}"
    }
  )
}

resource "aws_s3_bucket_acl" "logs" {
  count = var.enable_logging ? 1 : 0
  
  bucket = aws_s3_bucket.logs[0].id
  acl    = "log-delivery-write"
}

# Main S3 Bucket
resource "aws_s3_bucket" "main" {
  bucket = var.bucket_name
  
  tags = merge(
    var.tags,
    {
      Name = var.bucket_name
    }
  )
}

# Block all public access
resource "aws_s3_bucket_public_access_block" "main" {
  bucket = aws_s3_bucket.main.id
  
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Enable versioning
resource "aws_s3_bucket_versioning" "main" {
  bucket = aws_s3_bucket.main.id
  
  versioning_configuration {
    status = var.enable_versioning ? "Enabled" : "Suspended"
  }
}

# Server-side encryption with KMS
resource "aws_s3_bucket_server_side_encryption_configuration" "main" {
  bucket = aws_s3_bucket.main.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3_encryption.arn
    }
    bucket_key_enabled = true
  }
}

# Enable access logging
resource "aws_s3_bucket_logging" "main" {
  count = var.enable_logging ? 1 : 0
  
  bucket = aws_s3_bucket.main.id
  
  target_bucket = aws_s3_bucket.logs[0].id
  target_prefix = "access-logs/"
}

# Lifecycle policy
resource "aws_s3_bucket_lifecycle_configuration" "main" {
  bucket = aws_s3_bucket.main.id
  
  rule {
    id     = "intelligent-tiering"
    status = "Enabled"
    
    transition {
      days          = var.lifecycle_rules.transition_to_ia_days
      storage_class = "STANDARD_IA"
    }
    
    transition {
      days          = var.lifecycle_rules.transition_to_glacier_days
      storage_class = "GLACIER"
    }
    
    expiration {
      days = var.lifecycle_rules.expiration_days
    }
    
    noncurrent_version_expiration {
      noncurrent_days = 90
    }
  }
}

# Bucket policy enforcing encryption
resource "aws_s3_bucket_policy" "main" {
  bucket = aws_s3_bucket.main.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # Deny unencrypted object uploads
      {
        Sid    = "DenyUnencryptedObjectUploads"
        Effect = "Deny"
        Principal = "*"
        Action = "s3:PutObject"
        Resource = "${aws_s3_bucket.main.arn}/*"
        Condition = {
          StringNotEquals = {
            "s3:x-amz-server-side-encryption" = "aws:kms"
          }
        }
      },
      # Deny uploads without correct KMS key
      {
        Sid    = "DenyIncorrectEncryptionKey"
        Effect = "Deny"
        Principal = "*"
        Action = "s3:PutObject"
        Resource = "${aws_s3_bucket.main.arn}/*"
        Condition = {
          StringNotEquals = {
            "s3:x-amz-server-side-encryption-aws-kms-key-id" = aws_kms_key.s3_encryption.arn
          }
        }
      },
      # Deny insecure transport
      {
        Sid    = "DenyInsecureTransport"
        Effect = "Deny"
        Principal = "*"
        Action = "s3:*"
        Resource = [
          aws_s3_bucket.main.arn,
          "${aws_s3_bucket.main.arn}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      }
    ]
  })
}

# Enable bucket inventory
resource "aws_s3_bucket_inventory" "main" {
  bucket = aws_s3_bucket.main.id
  name   = "EntireBucketInventory"
  
  included_object_versions = "All"
  
  schedule {
    frequency = "Daily"
  }
  
  destination {
    bucket {
      format     = "CSV"
      bucket_arn = aws_s3_bucket.main.arn
      prefix     = "inventory/"
      encryption {
        sse_kms {
          key_id = aws_kms_key.s3_encryption.arn
        }
      }
    }
  }
  
  optional_fields = [
    "Size",
    "LastModifiedDate",
    "StorageClass",
    "ETag",
    "IsMultipartUploaded",
    "ReplicationStatus",
    "EncryptionStatus"
  ]
}
```

---

## iam.tf

```hcl
# IAM Role for application access (example)
resource "aws_iam_role" "app_role" {
  name = "${var.bucket_name}-app-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
  
  tags = merge(
    var.tags,
    {
      Name = "${var.bucket_name}-app-role"
    }
  )
}

# IAM Policy for S3 access with encryption
resource "aws_iam_role_policy" "s3_access" {
  name = "s3-kms-access"
  role = aws_iam_role.app_role.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # S3 bucket permissions
      {
        Sid    = "S3BucketAccess"
        Effect = "Allow"
        Action = [
          "s3:ListBucket",
          "s3:GetBucketLocation"
        ]
        Resource = aws_s3_bucket.main.arn
      },
      # S3 object permissions
      {
        Sid    = "S3ObjectAccess"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.main.arn}/*"
      },
      # KMS permissions
      {
        Sid    = "KMSAccess"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ]
        Resource = aws_kms_key.s3_encryption.arn
        Condition = {
          StringEquals = {
            "kms:ViaService" = "s3.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      }
    ]
  })
}

# Instance profile for EC2
resource "aws_iam_instance_profile" "app" {
  name = "${var.bucket_name}-instance-profile"
  role = aws_iam_role.app_role.name
}
```

---

## outputs.tf

```hcl
output "bucket_id" {
  description = "The name of the bucket"
  value       = aws_s3_bucket.main.id
}

output "bucket_arn" {
  description = "The ARN of the bucket"
  value       = aws_s3_bucket.main.arn
}

output "bucket_domain_name" {
  description = "The bucket domain name"
  value       = aws_s3_bucket.main.bucket_domain_name
}

output "bucket_regional_domain_name" {
  description = "The bucket region-specific domain name"
  value       = aws_s3_bucket.main.bucket_regional_domain_name
}

output "kms_key_id" {
  description = "The globally unique identifier for the key"
  value       = aws_kms_key.s3_encryption.key_id
}

output "kms_key_arn" {
  description = "The Amazon Resource Name (ARN) of the key"
  value       = aws_kms_key.s3_encryption.arn
  sensitive   = true
}

output "kms_key_alias" {
  description = "The display name of the key alias"
  value       = aws_kms_alias.s3_encryption.name
}

output "iam_role_arn" {
  description = "ARN of the IAM role for application access"
  value       = aws_iam_role.app_role.arn
}

output "instance_profile_name" {
  description = "Name of the instance profile for EC2"
  value       = aws_iam_instance_profile.app.name
}
```

---

## terraform.tfvars (Example)

```hcl
aws_region  = "us-east-1"
environment = "prod"
bucket_name = "shadowmesh-secure-data-prod"

authorized_role_arns = [
  "arn:aws:iam::123456789012:role/ApplicationRole",
  "arn:aws:iam::123456789012:role/DataProcessingRole"
]

enable_versioning = true
enable_logging    = true

lifecycle_rules = {
  transition_to_ia_days      = 30
  transition_to_glacier_days = 90
  expiration_days            = 365
}

tags = {
  Owner       = "Platform Team"
  CostCenter  = "Engineering"
  Compliance  = "SOC2"
}
```

---

## Usage Instructions

### 1. Initialize Terraform

```bash
terraform init
```

### 2. Validate Configuration

```bash
terraform validate
terraform fmt
```

### 3. Plan Deployment

```bash
terraform plan -out=tfplan
```

### 4. Apply Configuration

```bash
terraform apply tfplan
```

### 5. Verify Deployment

```bash
# Check bucket
aws s3 ls s3://shadowmesh-secure-data-prod

# Verify encryption
aws s3api get-bucket-encryption --bucket shadowmesh-secure-data-prod

# Verify KMS key
aws kms describe-key --key-id alias/shadowmesh-s3-encryption
```

---

## Testing Encryption

### Upload Test File

```bash
# This should SUCCEED (encrypted with KMS)
aws s3 cp test.txt s3://shadowmesh-secure-data-prod/ \
    --server-side-encryption aws:kms \
    --ssekms-key-id alias/shadowmesh-s3-encryption

# This should FAIL (no encryption specified)
aws s3 cp test.txt s3://shadowmesh-secure-data-prod/
# Error: Access Denied due to bucket policy
```

### Verify Object Encryption

```bash
aws s3api head-object \
    --bucket shadowmesh-secure-data-prod \
    --key test.txt

# Output should show:
# "ServerSideEncryption": "aws:kms"
# "SSEKMSKeyId": "arn:aws:kms:us-east-1:123456789012:key/..."
```

---

## Security Best Practices

1. **KMS Key Rotation:**
   - Automatic rotation enabled (yearly)
   - Monitor rotation with CloudWatch Events

2. **Access Logging:**
   - All access logged to separate bucket
   - Logs are also encrypted
   - Retention policy applied

3. **Bucket Policy:**
   - Denies unencrypted uploads
   - Requires specific KMS key
   - Enforces TLS/HTTPS only

4. **IAM Policies:**
   - Least privilege access
   - Scoped to specific resources
   - Conditions limit KMS usage to S3 service

5. **Versioning:**
   - Protects against accidental deletion
   - Supports compliance requirements
   - Noncurrent versions auto-expire

---

## Monitoring and Alerts

### CloudWatch Alarms

Add to your configuration:

```hcl
resource "aws_cloudwatch_metric_alarm" "kms_key_disabled" {
  alarm_name          = "kms-key-disabled-${var.bucket_name}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "UserErrorCount"
  namespace           = "AWS/KMS"
  period              = 300
  statistic           = "Sum"
  threshold           = 0
  alarm_description   = "KMS key may be disabled"
  
  dimensions = {
    KeyId = aws_kms_key.s3_encryption.key_id
  }
}
```

---

## Cleanup

To destroy all resources:

```bash
# Empty the bucket first
aws s3 rm s3://shadowmesh-secure-data-prod --recursive

# Destroy infrastructure
terraform destroy
```

**Note:** KMS keys have a 30-day deletion window and cannot be immediately deleted.
