## Shafran Backend

Go (Fiber) backend scaffold backed by PostgreSQL that implements the resources described in `model_backend_claude.json`. All endpoints live under the `/api` prefix and return the response envelopes defined in the specification (e.g. `{ "success": true, "data": ... }`).

### Stack

- Go 1.23+
- Fiber v2
- GORM + PostgreSQL
- JWT based authentication

### Configuration

Environment variables:

| Variable        | Default                                                           | Purpose                         |
| --------------- | ----------------------------------------------------------------- | ------------------------------- |
| `APP_PORT`      | `8080`                                                            | HTTP port for the API server    |
| `DATABASE_URL`  | `postgres://postgres:postgres@localhost:5432/shafran?sslmode=disable` | PostgreSQL connection string    |
| `JWT_SECRET`    | `super-secret-key`                                                | Signing key for access tokens   |
| `JWT_TTL_HOURS` | `24`                                                              | Token lifetime in hours         |

You can create a `.env` file in the project root; variables will be loaded automatically on startup.

### Local setup

```bash
# install deps (handled automatically by go modules)
go mod download

# run database migrations implicitly via application start
go run ./cmd/server
```

The server binds to `http://localhost:8080` by default.

### Key endpoints

- `POST /api/auth/register`, `POST /api/auth/login`, `POST /api/auth/verify`
- `GET|POST|PUT|DELETE /api/categories`, `/api/brands`, `/api/fragrance-notes`, `/api/seasons`, `/api/product-types`
- `GET|POST|PUT|DELETE /api/products`
- `GET|POST|PUT|DELETE /api/banner`, `/api/pickup-branches`, `/api/payment-providers`
- Authenticated (`Authorization: Bearer <token>`):
  - `GET|POST /api/orders`, `GET /api/orders/:id`
  - `GET|PUT /api/profile`
  - `GET|POST|PUT|DELETE /api/profile/addresses`
  - `GET /api/profile/bonus`

All CRUD endpoints implement pagination when a list is returned (`page` & `limit` query parameters).

