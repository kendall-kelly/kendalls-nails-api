# API Endpoints (High-Level)

## Authentication
- `POST /auth/register` - Customer registration
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout
- `POST /auth/technicians/invite` - Invite nail technician

## Orders
- `POST /orders` - Submit new order
- `GET /orders` - List orders (filtered by role)
- `GET /orders/:id` - Get order details
- `PUT /orders/:id/review` - Review order (accept/reject)
- `PUT /orders/:id/status` - Update order status
- `POST /orders/:id/reorder` - Create new order from existing design

## Designs (Public Gallery)
- `GET /designs` - Browse public designs
- `GET /designs/:id` - Get design details
- `PUT /designs/:id/visibility` - Toggle design visibility
- `DELETE /designs/:id` - Remove design from gallery
- `GET /designs/:id/inspiration` - See who used design as inspiration

## Comments
- `POST /designs/:id/comments` - Add comment to design
- `GET /designs/:id/comments` - Get comments for design

## Messages
- `POST /orders/:id/messages` - Send message about order
- `GET /orders/:id/messages` - Get messages for order

## Users
- `GET /users/me` - Get current user profile
- `PUT /users/me` - Update user profile
