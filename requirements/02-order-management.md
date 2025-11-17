# Order Management

## Order Submission
- Customers submit orders containing:
  - Design image (PNG format only, one per order)
  - Text description
  - Quantity (number of sets)
  - Note: Only hand nails supported (no feet)
- Orders cannot be cancelled once submitted
- No returns allowed (this may be supported at a later time)

## Order Status Workflow
Orders progress through the following statuses:
1. **Submitted** - Initial state when customer submits order
2. **Under Review** - Nail technician reviewing design
3. **Accepted** - Design approved and priced by technician
4. **Rejected** - Design rejected with feedback
5. **In Production** - Technician creating the nails
6. **Shipped** - Order shipped to customer
7. **Delivered** - Order received by customer

## Order Assignment
- Orders automatically distributed to available nail technicians
- Distribution algorithm TBD (round-robin, load balancing, etc.)

## Design Review Process
- Nail technician reviews submitted designs
- For **Acceptance**:
  - Technician sets final price (base price + complexity multiplier)
  - Design becomes final (no changes allowed after acceptance)
- For **Rejection**:
  - Technician must provide reason and feedback
  - Customer can update design and resubmit
  - Unlimited resubmission attempts allowed
  - Updated designs return to "Under Review" status

## Order History
- Customers can view all details of past orders
- Customers can reorder using same design
- Reorders treated as new orders (full review process applies)
