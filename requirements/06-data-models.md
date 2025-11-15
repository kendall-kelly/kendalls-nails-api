# Data Models (High-Level)

## User
- Customer profile
- Nail Technician profile
- Authentication credentials

## Order
- Design image reference
- Text description
- Quantity (number of sets)
- Status
- Price (set during approval)
- Customer reference
- Assigned nail technician reference
- Timestamps (created, updated, status changes)

## Design (Public Gallery)
- Reference to original order
- Visibility setting (public/private)
- Owner (customer)
- Usage tracking (who used as inspiration)

## Comment
- Reference to design
- Author (customer)
- Comment text
- Timestamp

## Message
- Reference to order
- Sender (customer or technician)
- Recipient (customer or technician)
- Message text
- Timestamp
