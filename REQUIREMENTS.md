# Custom Nails API - Functional Requirements Document

## Document History

| Version | Author | Review Date | Notes |
|---------|--------|-------------|-------|
| 1.0 | [Your Name] | 2025-11-13 | Initial document creation |

## Overview
Backend API for an online application enabling customers to order custom-designed nails. The platform connects customers with nail technicians who review, price, create, and ship custom nail orders.

## User Roles

### 1. Customer
- Self-registration enabled
- Can submit orders, communicate with technicians, view order history
- Can optionally share designs publicly and comment on other designs

### 2. Nail Technician
- Invitation/approval-based registration
- Reviews and prices design submissions
- Creates and ships custom nails
- Communicates with customers about orders

### 3. Anonymous User
- Browse-only access to public design gallery
- Must create Customer account to comment or place orders

## Core Features

### Order Management

#### Order Submission
- Customers submit orders containing:
  - Design image (PNG format only, one per order)
  - Text description
  - Quantity (number of sets)
  - Note: Only hand nails supported (no feet)
- Orders cannot be cancelled once submitted
- No returns allowed (this may be supported at a later time)

#### Order Status Workflow
Orders progress through the following statuses:
1. **Submitted** - Initial state when customer submits order
2. **Under Review** - Nail technician reviewing design
3. **Accepted** - Design approved and priced by technician
4. **Rejected** - Design rejected with feedback
5. **In Production** - Technician creating the nails
6. **Shipped** - Order shipped to customer
7. **Delivered** - Order received by customer

#### Order Assignment
- Orders automatically distributed to available nail technicians
- Distribution algorithm TBD (round-robin, load balancing, etc.)

#### Design Review Process
- Nail technician reviews submitted designs
- For **Acceptance**:
  - Technician sets final price (base price + complexity multiplier)
  - Design becomes final (no changes allowed after acceptance)
- For **Rejection**:
  - Technician must provide reason and feedback
  - Customer can update design and resubmit
  - Unlimited resubmission attempts allowed
  - Updated designs return to "Under Review" status

#### Order History
- Customers can view all details of past orders
- Customers can reorder using same design
- Reorders treated as new orders (full review process applies)

### Payment

#### Payment Model
- Payment occurs after design approval
- Pricing structure: Base price + complexity multiplier
- Nail technician determines complexity multiplier during approval process
- No refunds offered (this may be supported at a later time)

### Design Gallery & Sharing

#### Public Gallery
- Customers can optionally make their designs public
- Anonymous users can browse public designs
- Customers can:
  - Toggle design visibility (public/private) after posting
  - Remove their designs from gallery
  - See who used their design as inspiration

#### Design Reuse
- Customers can use existing public designs as starting point for new orders
- Creates a modifiable copy of the original design
- Original creator not credited or notified automatically
- No restrictions on reuse
- Original creator can see list of who used their design

#### Comments
- Only authenticated Customers can comment on public designs
- Anonymous users cannot comment
- No comment moderation by design owners
- No reporting mechanism for inappropriate content

### Communication

#### Direct Messaging
- Customers and nail technicians can message each other about specific orders
- Messages tied to order context

#### Notifications
- Not included in initial version (future feature)
- Potential future notifications:
  - Order status changes
  - New comments on shared designs
  - New messages

## Technical Requirements

### Authentication & Authorization
- Two authenticated user roles: Customer and Nail Technician
- Role-based access control (RBAC) for all endpoints
- Customer self-registration flow
- Nail technician invitation/approval flow

### File Upload
- Support PNG image uploads only
- One image per order
- Image validation and storage requirements TBD (max file size, dimensions, etc.)

### Data Models (High-Level)

#### User
- Customer profile
- Nail Technician profile
- Authentication credentials

#### Order
- Design image reference
- Text description
- Quantity (number of sets)
- Status
- Price (set during approval)
- Customer reference
- Assigned nail technician reference
- Timestamps (created, updated, status changes)

#### Design (Public Gallery)
- Reference to original order
- Visibility setting (public/private)
- Owner (customer)
- Usage tracking (who used as inspiration)

#### Comment
- Reference to design
- Author (customer)
- Comment text
- Timestamp

#### Message
- Reference to order
- Sender (customer or technician)
- Recipient (customer or technician)
- Message text
- Timestamp

## API Endpoints (High-Level)

### Authentication
- `POST /auth/register` - Customer registration
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout
- `POST /auth/technicians/invite` - Invite nail technician

### Orders
- `POST /orders` - Submit new order
- `GET /orders` - List orders (filtered by role)
- `GET /orders/:id` - Get order details
- `PUT /orders/:id/review` - Review order (accept/reject)
- `PUT /orders/:id/status` - Update order status
- `POST /orders/:id/reorder` - Create new order from existing design

### Designs (Public Gallery)
- `GET /designs` - Browse public designs
- `GET /designs/:id` - Get design details
- `PUT /designs/:id/visibility` - Toggle design visibility
- `DELETE /designs/:id` - Remove design from gallery
- `GET /designs/:id/inspiration` - See who used design as inspiration

### Comments
- `POST /designs/:id/comments` - Add comment to design
- `GET /designs/:id/comments` - Get comments for design

### Messages
- `POST /orders/:id/messages` - Send message about order
- `GET /orders/:id/messages` - Get messages for order

### Users
- `GET /users/me` - Get current user profile
- `PUT /users/me` - Update user profile

## Business Rules Summary

1. Only PNG images accepted for designs
2. One image per order
3. Only hand nails supported (no feet)
4. No order cancellations once submitted
5. No changes to design after acceptance
6. No returns or refunds
7. Payment after design approval
8. Unlimited design resubmission attempts
9. Orders automatically assigned to technicians
10. Customers must be authenticated to comment
11. Design reuse creates independent copy

## Future Enhancements
- Notifications system
- Comment moderation and reporting
- Multiple images per order
- Save/favorite designs
- Foot nail support
- Order cancellation (with conditions)
- Refund system
- Customer choice of preferred technician
- Real-time order tracking
- Rating/review system

## Open Questions / TBD
- Max file size for PNG uploads
- Image dimension requirements/restrictions
- Specific order assignment algorithm
- Payment gateway integration details
- Shipping provider integration
- Delivery confirmation mechanism
- Data retention policies
- Privacy policy for shared designs
