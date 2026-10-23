# Standardized Ecosystem Error Taxonomy

All NENAV APIs return errors in a standardized structure:

```json
{
  "error_code": "ERR_RESOURCE_NOT_FOUND",
  "message": "The requested booking ID does not exist.",
  "details": []
}
```

---

## 1. Authentication & Authorization Errors

| Error Code | HTTP Status | Description |
|---|---|---|
| `ERR_AUTH_MISSING_TOKEN` | 401 | `Authorization` header is missing or malformed. |
| `ERR_AUTH_INVALID_TOKEN` | 401 | JWT signature verification failed or token is malformed. |
| `ERR_AUTH_EXPIRED_TOKEN` | 401 | JWT token timestamp has expired. |
| `ERR_FORBIDDEN_INSUFFICIENT_ROLE` | 403 | User role is not authorized to access this resource. |
| `ERR_FORBIDDEN_OBJECT_ACCESS` | 403 | Attempted BOLA/IDOR access to an unauthorized object. |

---

## 2. Marketplace & Booking Errors

| Error Code | HTTP Status | Description |
|---|---|---|
| `ERR_SERVICE_UNAVAILABLE` | 400 | Service is inactive or outside operating hours. |
| `ERR_OUTSIDE_SERVICE_RADIUS` | 400 | Geolocation is beyond provider's maximum radius. |
| `ERR_BOOKING_SLOT_CONFLICT` | 409 | Professional is already booked for the requested timeframe. |
| `ERR_INVALID_STATE_TRANSITION` | 422 | Disallowed status jump (e.g. `REQUESTED` directly to `COMPLETED`). |

---

## 3. Financial & Payment Errors

| Error Code | HTTP Status | Description |
|---|---|---|
| `ERR_PAYMENT_INTENT_FAILED` | 402 | Gateway declined payment authorization. |
| `ERR_ESCROW_ALREADY_RELEASED`| 409 | Escrow funds have already been disbursed to the professional. |
