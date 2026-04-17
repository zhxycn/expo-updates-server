# expo-updates-server

A self-hosted, multi-tenant [Expo OTA Updates](https://docs.expo.dev/technical-specs/expo-updates-1/) server written in Go. Implements the Expo Updates protocol v0/v1 (manifest, multipart response, `rollBackToEmbedded` / `noUpdateAvailable` directives, code signing) and adds JWT-based user/project management plus API-key authenticated publishing.

## Quick start

### Run locally

Build the binary once (requires Go 1.26+), then run it standalone:

```bash
go build -o server ./cmd/main.go
./server
```

Or just run from source during development:

```bash
go run ./cmd/main.go
```

### Run with Docker

Use the prebuilt image:

```bash
docker run -d --name expo-updates-server \
  -p 8080:8080 \
  -v $PWD/data:/data \
  --env-file .env \
  zhxycn/expo-updates-server
```

Or build it yourself from this repo:

```bash
docker build -t expo-updates-server .
docker run -d --name expo-updates-server \
  -p 8080:8080 \
  -v $PWD/data:/data \
  --env-file .env \
  expo-updates-server
```

> When running in Docker, leave `STORAGE_DIR` and `DATABASE_PATH` at their defaults (or set them to paths under `/data`); the image mounts `/data` as a volume, so anything written elsewhere will be lost when the container is recreated.

## Configuration

All settings are loaded from environment variables (and a `.env` file if present):

| Variable                         | Default                  | Description                                         |
| -------------------------------- | ------------------------ | --------------------------------------------------- |
| `HOST` / `PORT`                  | `0.0.0.0` / `8080`       | Listen address.                                     |
| `HOSTNAME`                       | `http://localhost:8080`  | Public base URL used in manifest asset URLs.        |
| `STORAGE_TYPE`                   | `local`                  | `local` or `s3` (S3-compatible, e.g. Cloudflare R2). |
| `STORAGE_DIR`                    | `./data/updates`         | Local storage root (when `STORAGE_TYPE=local`).     |
| `DATABASE_PATH`                  | `./data/ota.db`          | SQLite database file.                               |
| `S3_ENDPOINT` / `S3_BUCKET` / `S3_REGION` / `S3_ACCESS_KEY` / `S3_SECRET_KEY` | – | S3 credentials. |
| `PRIVATE_KEY`                    | –                        | RSA private key (PEM contents) **or** path to a `.pem` file, for code signing. See [Code signing](#code-signing). |
| `JWT_SECRET`                     | –                        | HMAC secret used to sign user JWTs.                 |

## Concepts

- **User** – authenticates with JWT (`/api/auth/*`).
- **Project** – an OTA channel identified by an `id`. Membership has roles `owner` and `member`. Owners may manage members and API keys.
- **API Key** – per-project bearer token used by CI/CLI to publish updates. The plain-text secret is shown **only once** at creation.
- **Update** – a manifest plus a set of assets, grouped by `(project, runtimeVersion)` and identified by a Unix-timestamp `updateID`. The latest update for a `(project, runtimeVersion)` pair is served to clients.

## Authentication

Three different auth mechanisms are used depending on the route group:

| Route group              | Auth                                                                                              |
| ------------------------ | ------------------------------------------------------------------------------------------------- |
| `/api/auth/*`            | None (public).                                                                                    |
| `/api/projects/*`        | `Authorization: Bearer <jwt>` issued by login/register, valid for 72 h.                           |
| `/api/updates/:project/manifest` and `/assets` | None – uses the Expo client protocol headers / query params (and optional `expo-expect-signature` for code signing). |
| `/api/updates/:project/publish` | `Authorization: Bearer <api_key_secret>`, scoped to the matching project.                    |

All error responses use the shape `{ "error": "<message>" }` unless otherwise noted.

---

## API reference

### Auth

#### `POST /api/auth/register`

Create a user account and receive a JWT.

Request body:

```json
{ "username": "alice", "email": "alice@example.com", "password": "secret" }
```

Responses:

- `201 Created` – `{ "user": User, "token": "<jwt>" }`
- `400 Bad Request` – missing parameters.
- `409 Conflict` – username or email already exists.

#### `POST /api/auth/login`

Exchange credentials for a JWT.

Request body (`login` may be either the username or the email):

```json
{ "login": "alice", "password": "secret" }
```

Responses:

- `200 OK` – `{ "user": User, "token": "<jwt>" }`
- `400 Bad Request` – missing parameters.
- `401 Unauthorized` – invalid credentials.

---

### Projects

All routes below require `Authorization: Bearer <jwt>`.

#### `POST /api/projects`

Create a new project. The caller is automatically added as `owner`.

Request body:

```json
{ "name": "My App", "slug": "my-app" }
```

Responses:

- `201 Created` – the created `Project` object.
- `400 Bad Request` – missing parameters.

#### `GET /api/projects`

List projects the current user is a member of.

Response: `200 OK` – `Project[]`.

#### `POST /api/projects/:id/users`

Add a user to a project. **Owner only.**

Request body:

```json
{ "userId": "<user_id>", "role": "member" }
```

`role` is one of `owner` | `member`.

Responses:

- `201 Created` – the created `ProjectUser` membership.
- `403 Forbidden` – caller is not an owner of the project.
- `409 Conflict` – the user is already a member.

#### `DELETE /api/projects/:id/users/:userId`

Remove a user from a project. **Owner only.** Owners cannot remove themselves.

Responses:

- `204 No Content`
- `400 Bad Request` – tried to remove yourself.
- `403 Forbidden` – not an owner.

#### `POST /api/projects/:id/keys`

Create a new API key for the project. **Owner only.**

Request body (the `name` is optional; one is auto-generated when omitted):

```json
{ "name": "ci-key" }
```

Response: `201 Created`

```json
{
  "key": { "id": "...", "projectId": "...", "name": "ci-key", "keyPrefix": "abcd1234", "createdBy": "...", "createdAt": "..." },
  "secret": "<plain-text token shown only once — use this for publishing>"
}
```

#### `GET /api/projects/:id/keys`

List API keys (without the secret) for the project. **Owner only.**

Response: `200 OK` – `Key[]`.

#### `PATCH /api/projects/:id/keys/:keyId`

Rename an API key. **Owner only.**

Request body: `{ "name": "new-name" }` → `204 No Content`.

#### `DELETE /api/projects/:id/keys/:keyId`

Revoke an API key. **Owner only.** → `204 No Content`.

---

### Updates (Expo client protocol)

These endpoints implement [Expo Updates v0/v1](https://docs.expo.dev/technical-specs/expo-updates-1/). The path parameter `:project` is the project `id`.

#### `GET /api/updates/:project/manifest`

Returns the latest manifest (or a directive) for the requested runtime/platform.

Request headers:

| Header                       | Required | Notes                                                          |
| ---------------------------- | -------- | -------------------------------------------------------------- |
| `expo-platform`              | yes      | `ios` or `android`.                                            |
| `expo-runtime-version`       | yes      | Runtime version baked into the native client.                  |
| `expo-protocol-version`      | no       | `0` (legacy) or `1`. Required (`>=1`) for rollback/no-update directives. |
| `expo-current-update-id`     | no       | UUID of the update currently installed. With protocol `v1` enables the `noUpdateAvailable` directive when already up to date. |
| `expo-embedded-update-id`    | no       | UUID of the update embedded in the binary; used by the rollback flow. |
| `expo-expect-signature`      | no       | Presence enables [code signing](#code-signing); the response will include `expo-signature`. |

Response: `200 OK` `multipart/mixed; boundary=…` with parts:

- `manifest` – `application/json` body matching the [Expo manifest](https://docs.expo.dev/technical-specs/expo-updates-1/#manifest-body) spec:

  ```jsonc
  {
    "id": "<uuid>",
    "createdAt": "2026-01-01T10:00:00Z",
    "runtimeVersion": "1.0.0",
    "launchAsset": { "key": "...", "contentType": "application/javascript", "url": "https://.../assets?asset=...&platform=...&runtimeVersion=..." },
    "assets": [{ "hash": "...", "key": "...", "contentType": "image/png", "fileExtension": ".png", "url": "..." }],
    "metadata": {},
    "extra": { "expoClient": { /* contents of app config */ } }
  }
  ```

- `extensions` – `application/json` body containing `assetRequestHeaders` (one entry per asset key, currently always empty maps).
- `directive` – returned **instead of** `manifest` when the latest update is a rollback or when the client is already up to date. Body example:

  ```json
  { "type": "rollBackToEmbedded", "parameters": { "commitTime": "2026-04-17T10:00:00Z" } }
  ```

  or

  ```json
  { "type": "noUpdateAvailable" }
  ```

Response headers: `expo-protocol-version`, `expo-sfv-version: 0`, `cache-control: private, max-age=0`. When code signing is enabled the manifest/directive part also includes a part header `expo-signature: sig=:<base64>:, keyid="main"`.

Errors:

- `400 Bad Request` – missing/invalid `expo-platform` or `expo-runtime-version`; or `expo-expect-signature` was sent but the server has no `PRIVATE_KEY` configured.
- `404 Not Found` – no update for this `(project, runtimeVersion)`, the update is missing `expoConfig.json`, or rollback requested with `expo-protocol-version: 0`.

#### `GET /api/updates/:project/assets`

Stream a single asset belonging to the latest update.

Query parameters:

| Param            | Required | Notes                                              |
| ---------------- | -------- | -------------------------------------------------- |
| `asset`          | yes      | Asset path as referenced in the manifest URL.      |
| `platform`       | yes      | `ios` or `android`.                                |
| `runtimeVersion` | yes      | Same value used in the manifest request.           |

Responses:

- `200 OK` – binary stream. `Content-Type` is `application/javascript` for files under `bundles/`, otherwise inferred from the file extension (defaults to `application/octet-stream`). `Cache-Control: public, max-age=31536000, immutable`.
- `400 Bad Request` – missing parameter.
- `404 Not Found` – asset does not exist.

#### `POST /api/updates/:project/publish`

Publish a new update for the project. Requires a project API key:

```text
Authorization: Bearer <project_api_key_secret>
```

Request: `multipart/form-data` with:

| Field             | Type | Description                                                                                                                                              |
| ----------------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `runtimeVersion`  | text | Runtime version of the update. Must match the value baked into the native client.                                                                        |
| `metadata.json`   | file | Standard Expo [`metadata.json`](https://docs.expo.dev/eas-update/how-it-works/) produced by `npx expo export`. Lists per-platform `bundle` and `assets`. |
| `expoConfig.json` | file | Public Expo app config (`npx expo config --type public --json`). Surfaced to clients as `manifest.extra.expoClient` and required for serving manifests.  |
| (bundle + assets) | file | One file field per path listed in `metadata.json` — e.g. `_expo/static/js/<platform>/entry-<hash>.hbc` and `assets/<hash>` files. The form field **name** must equal that path with forward slashes. |

The server reads `metadata.json`, hashes every referenced file (SHA-256 base64url for `hash`, MD5 hex for `key`), generates per-platform `index.<platform>.json` files, and stores everything under `<project>/<runtimeVersion>/<updateID>/` where `updateID` is the current Unix timestamp. Backslashes in `metadata.json` paths (which `expo export` produces on Windows) are normalized to forward slashes automatically.

Responses:

- `200 OK` – `{ "updateID": "<unix_ts>", "message": "Update published successfully" }`
- `400 Bad Request` – missing `runtimeVersion` or malformed multipart form.
- `401 Unauthorized` – missing/invalid API key, or the key does not belong to `:project`.
- `500 Internal Server Error` – missing `metadata.json`, or a bundle/asset referenced by `metadata.json` is absent from the upload.

#### End-to-end publish example

1. Build the update artifacts in your Expo project:

   ```bash
   npx expo export --platform ios --platform android --output-dir dist
   npx expo config --type public --json > dist/expoConfig.json
   ```

   This produces (Expo SDK 50+):

   ```text
   dist/
     metadata.json
     expoConfig.json
     _expo/static/js/ios/entry-<hash>.hbc
     _expo/static/js/android/entry-<hash>.hbc
     assets/<md5>...
   ```

2. Upload everything with `curl` (bash). Each file's `-F` field name must be its path inside `dist/`:

   ```bash
   cd dist
   FORMS=(
     -F "runtimeVersion=1.0.0"
     -F "metadata.json=@metadata.json"
     -F "expoConfig.json=@expoConfig.json"
   )
   while IFS= read -r f; do
     FORMS+=(-F "$f=@$f")
   done < <(find _expo assets -type f)

   curl -X POST "$HOSTNAME/api/updates/$PROJECT_ID/publish" \
     -H "Authorization: Bearer $API_KEY" \
     "${FORMS[@]}"
   ```

   Windows PowerShell equivalent (uses `curl.exe`, works on 5.1+):

   ```powershell
   cd dist
   $cargs = @(
     '-X','POST',"$env:HOSTNAME/api/updates/$env:PROJECT_ID/publish",
     '-H',"Authorization: Bearer $env:API_KEY",
     '-F','runtimeVersion=1.0.0',
     '-F','metadata.json=@metadata.json',
     '-F','expoConfig.json=@expoConfig.json'
   )
   Get-ChildItem -Recurse -File _expo, assets | ForEach-Object {
     $rel = (Resolve-Path -Relative $_.FullName).TrimStart('.\').Replace('\','/')
     $cargs += '-F'; $cargs += "$rel=@$rel"
   }
   & curl.exe @cargs
   ```

   > Tip: when generating `expoConfig.json` on Windows, redirect with `cmd /c "npx expo config --type public --json > dist\expoConfig.json"` rather than PowerShell `Out-File`, which adds a UTF-8 BOM that prevents the server from parsing `manifest.extra.expoClient`.

---

## Code signing

Set the `PRIVATE_KEY` environment variable to enable manifest signing. The value can be either:

- The **PEM contents** themselves (recommended — works in both PKCS#1 `-----BEGIN RSA PRIVATE KEY-----` and PKCS#8 `-----BEGIN PRIVATE KEY-----` formats); ideal for Docker / 12-factor deployments where you don't want to mount a key file.
- A **filesystem path** to a `.pem` file (used when the value does not contain `BEGIN`).

Example `.env`:

```env
PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQ...
...
-----END PRIVATE KEY-----"
```

When a client sends `expo-expect-signature`, the server signs the JSON body of the `manifest` (or `directive`) part using `RSASSA-PKCS1-v1_5` + `SHA-256` and adds an `expo-signature: sig=:<base64>:, keyid="main"` part header (the signature value uses the [Expo SFV](https://docs.expo.dev/technical-specs/expo-sfv-0) byte-string syntax). Configure the matching public certificate in the Expo client with `keyid: "main"` and `alg: "rsa-v1_5-sha256"` (see the [Expo code-signing guide](https://docs.expo.dev/eas-update/code-signing/)). If `expo-expect-signature` is sent but the server has no key configured, the request fails with `400`.

## Storage layout

Each update is written under `<STORAGE_DIR or S3 bucket>/<project>/<runtimeVersion>/<updateID>/`. The exact subpaths depend on what `metadata.json` references; for an Expo SDK 50+ export it looks like:

```
<project>/<runtimeVersion>/<updateID>/
  metadata.json                                  # uploaded
  expoConfig.json                                # uploaded
  index.ios.json                                 # generated by server
  index.android.json                             # generated by server
  _expo/static/js/ios/entry-<hash>.hbc           # uploaded (ios bundle)
  _expo/static/js/android/entry-<hash>.hbc       # uploaded (android bundle)
  assets/<md5>...                                # uploaded
  rollback                                       # optional marker (see below)
```

For pre-SDK-50 exports the bundles live under `bundles/index.<platform>.js` instead; the server doesn't care — it stores whatever paths `metadata.json` declares.

### Triggering a rollback

There is no dedicated API for rollbacks. To roll a `(project, runtimeVersion)` back to the embedded update, drop an empty file named `rollback` into the desired update directory:

- **Local storage:** `touch <STORAGE_DIR>/<project>/<runtimeVersion>/<updateID>/rollback`
- **S3:** `PUT` an empty object at `<project>/<runtimeVersion>/<updateID>/rollback`

When that update becomes the latest, the manifest endpoint returns a `rollBackToEmbedded` directive with `parameters.commitTime` set to the update's timestamp (see [internal/service/update.go](internal/service/update.go) and [internal/storage](internal/storage)).

## Project layout

- [cmd/main.go](cmd/main.go) – entrypoint, wires config / storage / DB / signer / handlers.
- [internal/config](internal/config) – env-based configuration.
- [internal/handler](internal/handler) – HTTP handlers (auth, projects, manifest, assets, publish).
- [internal/middleware](internal/middleware) – JWT middleware.
- [internal/service](internal/service) – update service (manifest assembly, publish, rollback, hashing).
- [internal/storage](internal/storage) – `local` and `s3` storage backends.
- [internal/signing](internal/signing) – RSA code-signing of manifests and directives.
- [internal/database](internal/database) – Bun + SQLite repositories for users, projects, memberships and keys.
- [internal/model](internal/model) – persistent and protocol data models.
- [internal/crypto](internal/crypto) – Argon2id password hashing.

## License

See [LICENSE](LICENSE).
