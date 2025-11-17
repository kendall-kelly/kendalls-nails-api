# File Storage

- **Storage Provider**: AWS S3 (Amazon Simple Storage Service)
  - Object storage for uploaded design images
  - Scalable, durable, and highly available
  - Cost-effective for image storage
- **Go SDK**: AWS SDK for Go v2 (github.com/aws/aws-sdk-go-v2)
  - S3 client for upload, download, and deletion operations
  - Credential management via environment variables or IAM roles
- **File Format Requirements**:
  - **Allowed Format**: PNG only
  - **Validation**:
    - File extension check: Must end with `.png` (case-insensitive)
    - MIME type verification: Must be `image/png`
    - File signature check: Validate PNG magic bytes (89 50 4E 47) for security
- **File Size & Dimensions**:
  - **Maximum file size**: 10MB per image
  - **Maximum dimensions**: 4096 x 4096 pixels
  - **Minimum dimensions**: 100 x 100 pixels (ensure design is visible)
  - Validation performed before upload to S3
- **S3 Bucket Configuration**:
  - **Bucket naming**: `kendalls-nails-{environment}` (e.g., `kendalls-nails-prod`, `kendalls-nails-dev`)
  - **Region**: us-east-1 (or closest to target users)
  - **Versioning**: Disabled (images are immutable once uploaded)
  - **Encryption**: Server-side encryption enabled (SSE-S3 or SSE-KMS)
  - **Public access**: Blocked by default (use signed URLs for access)
- **File Naming Convention**:
  - **Format**: `{order-id}/{uuid}.png`
  - **Example**: `12345/a3f2c9e1-4b5a-4d3c-8e2f-1a2b3c4d5e6f.png`
  - UUID prevents filename collisions
  - Order ID prefix organizes files and enables easy cleanup
- **Upload Process**:
  1. Receive multipart form upload from client
  2. Validate file format, size, and dimensions
  3. Generate UUID for filename
  4. Upload to S3 with appropriate metadata (content-type, order-id tag)
  5. Store S3 object key/URL in database `orders` table
  6. Return success response to client
- **Access Control**:
  - **Private Images** (order designs before sharing):
    - Not publicly accessible
    - Generate presigned URLs (temporary, expiring links) for authorized users
    - Expiration time: 1 hour for viewing
  - **Public Images** (designs shared to gallery):
    - Object ACL set to public-read, or
    - CloudFront distribution for better performance (optional future enhancement)
    - URLs stored in database for direct access
- **Image Serving**:
  - **For Orders**: Backend generates presigned URLs when customer/technician requests order details
  - **For Gallery**: Direct S3 URLs for public designs, or CloudFront URLs if CDN is configured
- **Deletion Policy**:
  - Images are never deleted automatically (order history preservation)
  - Manual deletion only if customer explicitly requests design removal from gallery
  - Soft delete in database, preserve S3 object with lifecycle policy (archive to Glacier after 1 year if needed)
- **Local Development**:
  - **Option 1**: Use LocalStack (S3 emulator) for development without AWS costs
  - **Option 2**: Use separate S3 dev bucket with minimal storage
  - **Option 3**: Use local filesystem in dev, S3 in production (abstraction layer recommended)
