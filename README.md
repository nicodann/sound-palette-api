# Sound Palette API

The Go backend for [Sound Palette](https://github.com/nicodann/sound-palette), a demo app that turns a description of a scene or feeling into five words and five colours. This API generates palettes with Claude, and handles accounts and saved palettes.

> **This is a demo.** It's a personal learning project, not a production service. AI generation is rate-limited and capped per day, and the database may be reset at any time.

## Requirements

- Go 1.26+
- PostgreSQL 13+ (uses the built-in `gen_random_uuid()`)
- An [Anthropic API key](https://console.anthropic.com/) — optional locally if you use mock mode

## Running it locally

```sh
git clone https://github.com/nicodann/sound-palette-api.git
cd sound-palette-api
```

**1. Create the database and load the schema:**

```sh
createdb sound_palette
psql sound_palette -f schema.sql
```

**2. Configure it:**

```sh
cp .env.example .env
```

Fill in `.env` (see [Environment variables](#environment-variables)). At minimum, set `DATABASE_URL` and a `JWT_SECRET`:

```sh
openssl rand -base64 32   # generates a suitable JWT_SECRET
```

Leave `MOCK_AI=true` while developing: the API then returns a fixed palette instead of calling Claude, so it costs nothing and needs no API key.

**3. Run it:**

```sh
go run .
```

The server listens on <http://localhost:8080>. It exits at startup with a clear message if a required variable is missing.

Then run the [frontend](https://github.com/nicodann/sound-palette) with `VITE_API_URL=http://localhost:8080`.

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | yes | Postgres connection string. Locally `...?sslmode=disable`; hosted databases usually need `sslmode=require` |
| `JWT_SECRET` | yes | Secret for signing login tokens, at least 32 characters. Use a different one in every environment |
| `ALLOWED_ORIGIN` | yes | The frontend's exact origin, for CORS, e.g. `http://localhost:5173` or `https://yourdomain.com` (no trailing slash) |
| `ANTHROPIC_API_KEY` | unless `MOCK_AI=true` | Key used to call Claude |
| `MOCK_AI` | no | `true` returns a fixed palette instead of calling Claude |
| `COOKIE_SECURE` | no | `true` makes the login cookie HTTPS-only. Set it in production; leave it `false` locally (Safari won't keep `Secure` cookies on `http://localhost`) |
| `TRUST_PROXY` | no | `true` reads the client IP from `X-Forwarded-For`. See [Rate limiting](#rate-limiting) |
| `PORT` | no | Port to listen on. Defaults to `8080`; most hosts set it for you |

`.env` is git-ignored. In production, set these in your host's dashboard rather than in a file.

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/ai-query` | — | Generate a palette. Body: `{ "input": "..." }` (max 500 characters) |
| `POST` | `/auth/register` | — | Create an account and log in. Body: `{ "email", "password" }` (password 8+ characters) |
| `POST` | `/auth/login` | — | Log in. Sets the `token` cookie |
| `GET` | `/auth/me` | cookie | The logged-in user |
| `POST` | `/auth/logout` | — | Clears the `token` cookie |
| `GET` | `/palettes` | cookie | The user's saved palettes, newest first |
| `POST` | `/palettes` | cookie | Save a palette. Body: `{ "name", "prompt", "palette" }` |
| `DELETE` | `/palettes/{id}` | cookie | Delete one of the user's palettes |

A palette is an array of five `{ "adjective": "dreamy", "colour": "#A8D8EA" }` objects. Request bodies are limited to 16 KB, and names (50), prompts (500), adjectives (30) and colours (`#RRGGBB`) are validated.

## Rate limiting

| Routes | Per IP | Total |
|---|---|---|
| `/ai-query` | 5 a minute, ~30 a day | ~150 a day |
| `/auth/login`, `/auth/register` | 10 a minute, ~100 a day | — |

Limits are token buckets held **in memory**: they reset when the server restarts and apply per server instance. Over-limit requests get `429 Too Many Requests`.

### `TRUST_PROXY`

Behind a hosting provider's reverse proxy, every request reaches the server from the proxy's address, so per-IP limits would treat all visitors as one. With `TRUST_PROXY=true`, the client IP is taken from the **last** `X-Forwarded-For` entry, which is the one added by the proxy (clients can only fake entries before it).

- **Locally, leave it unset or `false`.** Without a proxy, clients can send any `X-Forwarded-For` they like and bypass the per-IP limits.
- **This assumes exactly one proxy.** With another layer in front (such as Cloudflare), the last entry is that layer's address; use its client-IP header instead.

## Deploying

1. Create a managed Postgres database and run `schema.sql` against it.
2. Deploy the API to a host that runs Go (most platforms detect it automatically).
3. Set the environment variables, with production values: `MOCK_AI=false`, `COOKIE_SECURE=true`, `TRUST_PROXY=true`, a new `JWT_SECRET`, and `ALLOWED_ORIGIN` set to the deployed frontend.
4. Set a monthly spend limit for the API key in the Anthropic Console.

Host the frontend and API on the **same site** (e.g. `yourdomain.com` and `api.yourdomain.com`). The login cookie uses `SameSite=Lax`, so it isn't sent between different sites.

## Project layout

| File | Contents |
|---|---|
| `main.go` | configuration checks, middleware, routes, server setup |
| `auth.go` | register, login, logout, `/auth/me`, cookie and JWT helpers |
| `palettes.go` | saving, listing, deleting and validating palettes |
| `ai.go` | palette generation with Claude |
| `ratelimit.go` | per-IP and global rate limiting |
| `schema.sql` | database tables |
