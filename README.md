# krypt

Encrypted `.env` credential manager for Go. Inspired by Rails credentials.

Encrypt your secrets with AES-256-GCM, commit `.env.enc` files to version control, and decrypt them at runtime. Works as a CLI tool and as a Go package.

## Install

```bash
go install github.com/aschiavon91/krypt/cmd/krypt@latest
```

## Quick Start

```bash
# Generate an encryption key
export KRYPT_KEY=$(krypt keygen)
echo $KRYPT_KEY  # save this somewhere safe

# Create your secrets file
cat > .env <<EOF
DATABASE_URL=postgres://localhost:5432/myapp
API_KEY=sk_live_abc123
REDIS_URL=redis://localhost:6379
EOF

# Encrypt it
krypt encrypt
# -> creates .env.enc (add this to git)
# -> keep .env in .gitignore

# Decrypt to stdout
krypt decrypt

# Run a command with secrets injected
krypt run -- go run ./cmd/server

# Edit secrets in your $EDITOR
krypt edit

# Set/get individual keys
krypt set "STRIPE_KEY=sk_live_new"
krypt get STRIPE_KEY
```

## Multiple Environments

krypt supports named environments with separate keys:

```bash
# File naming convention:
#   .env.enc       -> KRYPT_KEY
#   .env.dev.enc   -> KRYPT_KEY_DEV
#   .env.staging.enc -> KRYPT_KEY_STAGING

# Generate a key for dev
export KRYPT_KEY_DEV=$(krypt keygen)

# Encrypt dev secrets
krypt encrypt dev

# Decrypt dev secrets
krypt decrypt dev

# Run with dev secrets
krypt run dev -- go run ./cmd/server

# Edit dev secrets
krypt edit dev

# Set/get in dev
krypt set "DEBUG=true" dev
krypt get DEBUG dev
```

## CLI Reference

### `krypt keygen`

Generate a new 32-byte encryption key (64 hex characters).

```bash
krypt keygen
# a1b2c3d4e5f6...
```

### `krypt encrypt [env]`

Encrypt a plaintext `.env` file into `.env.enc`.

```bash
krypt encrypt           # .env -> .env.enc
krypt encrypt staging   # .env.staging -> .env.staging.enc
```

### `krypt decrypt [env]`

Decrypt and print to stdout. Useful for piping.

```bash
krypt decrypt
krypt decrypt dev | grep DATABASE
```

### `krypt edit [env]`

Open decrypted secrets in `$VISUAL` or `$EDITOR` (falls back to `vi`). Re-encrypts on save. If the editor exits non-zero, changes are discarded.

The temp file is created in `/dev/shm` (Linux) when available for in-memory storage, otherwise `/tmp`, with `0600` permissions.

```bash
krypt edit
krypt edit staging
```

### `krypt run [env] -- <command> [args...]`

Decrypt secrets and inject them into the environment of a subprocess. Secrets override existing env vars with the same name. The exit code of the child process is passed through.

```bash
krypt run -- go run ./cmd/server
krypt run dev -- npm start
krypt run staging -- docker compose up
```

### `krypt set KEY=VALUE [env]`

Set or update a single key in an encrypted file.

```bash
krypt set "DATABASE_URL=postgres://prod-host/myapp"
krypt set "DEBUG=true" dev
```

### `krypt get KEY [env]`

Get a single value from an encrypted file. Exits non-zero if the key is not found.

```bash
krypt get DATABASE_URL
krypt get DEBUG dev
```

### Common Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--key` | `-k` | Provide encryption key directly |
| `--file` | `-f` | Override encrypted file path |
| `--source` | | Override source `.env` file path (encrypt only) |

### Key Resolution Order

1. `--key` flag
2. `KRYPT_KEY_<ENV>` environment variable (if env specified)
3. `KRYPT_KEY` environment variable
4. Error with helpful message

## Using as a Go Package

The `pkg/krypt` package has zero external dependencies (pure stdlib) and can be imported into any Go project.

```bash
go get github.com/aschiavon91/krypt
```

```go
import "github.com/aschiavon91/krypt/pkg/krypt"
```

### Load secrets as a map

```go
key, _ := hex.DecodeString(os.Getenv("KRYPT_KEY"))

secrets, err := krypt.Load(".env.enc", key)
if err != nil {
    log.Fatal(err)
}

db.Connect(secrets["DATABASE_URL"])
```

### Auto-inject into process environment

```go
key, _ := hex.DecodeString(os.Getenv("KRYPT_KEY"))

if err := krypt.Autoload(".env.enc", key); err != nil {
    log.Fatal(err)
}

// os.Getenv("DATABASE_URL") now works
```

### Encrypt secrets programmatically

```go
keyHex, _ := krypt.GenerateKey()
key, _ := hex.DecodeString(keyHex)

content := []byte("DATABASE_URL=postgres://localhost/myapp\nAPI_KEY=secret\n")
if err := krypt.Encrypt(content, ".env.enc", key); err != nil {
    log.Fatal(err)
}
```

### Read/write individual keys

```go
key, _ := hex.DecodeString(os.Getenv("KRYPT_KEY"))

// Set
krypt.Set(".env.enc", key, "NEW_SECRET", "value123")

// Get
val, err := krypt.Get(".env.enc", key, "DATABASE_URL")
if errors.Is(err, krypt.ErrKeyNotFound) {
    // key doesn't exist
}
```

### API Reference

```go
func GenerateKey() (string, error)
func Encrypt(plaintext []byte, path string, key []byte) error
func Decrypt(path string, key []byte) ([]byte, error)
func Load(path string, key []byte) (map[string]string, error)
func Autoload(path string, key []byte) error
func Set(path string, key []byte, envKey, envValue string) error
func Get(path string, key []byte, envKey string) (string, error)

var ErrKeyNotFound = errors.New("key not found")
```

All functions accept `[]byte` keys (already decoded from hex). The caller handles hex decoding. No global state, no `init()`, no singletons.

## File Format

Encrypted files are raw AES-256-GCM sealed bytes: `nonce (12 bytes) || ciphertext || GCM auth tag (16 bytes)`. A fresh random nonce is generated on every encrypt, so re-encrypting the same content produces different output.

Decrypted content is standard `.env` format:

```bash
# Database config
DATABASE_URL=postgres://localhost:5432/myapp
DATABASE_POOL=10

# API keys
API_KEY=sk_live_abc123
```

Parser rules:
- `KEY=VALUE` lines are parsed as env vars
- `#` lines and blank lines are preserved on round-trip
- Values can be quoted with `"` or `'`
- No variable interpolation
- No trailing comments

## Security

- AES-256-GCM encryption (Go stdlib `crypto/aes` + `crypto/cipher`)
- Keys are 32 bytes generated from `crypto/rand`
- Fresh random nonce on every encrypt operation
- Temp files during `edit` use `/dev/shm` (Linux) with `0600` permissions
- Temp files are cleaned up even on panic (deferred removal)

## License

MIT
