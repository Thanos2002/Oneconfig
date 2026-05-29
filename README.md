# OneConfig

**Set up any dev environment with one command.**

OneConfig reads a single `oneconfig.yml` file at the root of your repository and provisions a fully working local development environment. No more reading READMEs, installing mismatched tool versions, or starting services manually.

```
git clone <repo>
cd <repo>
oneconfig up
```

That's it. You're ready to code.

---

## Quick Start

### 1. Install

```bash
# macOS / Linux
curl -fsSL https://oneconfig.dev/install.sh | sh

# Windows
winget install oneconfig

# Or build from source
go install github.com/oneconfig/oneconfig@latest
```

### 2. Initialize

```bash
# Generate a config from your existing project
oneconfig generate

# Or create a starter config
oneconfig init
```

### 3. Run

```bash
oneconfig up
```

---

## What It Does

When you run `oneconfig up`, it:

1. **Validates** your `oneconfig.yml`
2. **Installs runtimes** (Node.js, Python) via your version manager
3. **Installs dependencies** (npm, yarn, pip, poetry, etc.)
4. **Sets environment variables** from config into a local `.env`
5. **Starts services** in dependency order (databases, APIs, workers)
6. **Waits for health checks** before marking the environment ready
7. **Runs post-start commands** (migrations, seed data, etc.)

---

## Example Config

```yaml
project_name: my-app

runtimes:
  - name: node
    version: "20.x"

package_managers:
  - type: npm
    path: .

env_vars:
  NODE_ENV: development
  DATABASE_URL: postgres://localhost:5432/myapp

services:
  - name: postgres
    type: docker
    start_command: docker run -p 5432:5432 postgres:15
    port: 5432
    health_check:
      type: tcp
      target: ":5432"
      timeout: 30s

  - name: api
    start_command: npm run dev
    port: 3000
    depends_on: [postgres]
    health_check:
      type: http
      target: http://localhost:3000/health

setup_steps:
  - name: migrate
    command: npm run db:migrate
    depends_on: [postgres]

post_start_commands:
  - echo "🚀 Ready at http://localhost:3000"
```

---

## Commands

| Command | Description |
|---------|-------------|
| `oneconfig init` | Create or validate a `oneconfig.yml` |
| `oneconfig up` | Set up and start the environment |
| `oneconfig down` | Stop all running services |
| `oneconfig status` | Show service status (PID, port, health) |
| `oneconfig doctor` | Check for missing tools |
| `oneconfig generate` | Scan a project and generate a draft config |

---

## Supported Stacks (v1)

OneConfig's `generate` engine has universal auto-detection capabilities!

- **JavaScript / TypeScript** — Node.js, npm, yarn, pnpm, bun
- **Python** — pip, poetry, uv, pipenv
- **Systems & Backend** — Go, Rust, Java (Maven/Gradle), Ruby
- **Web Frameworks** — Next.js, Django, FastAPI, Flask, Streamlit, Gradio
- **Databases** — Auto-detects dependencies to spin up Postgres, MongoDB, Redis, MySQL via Docker.
- **Monorepos** — Deep-scans packages, apps, and services directories.
- **Other** — Makefile, Procfile, Dockerfile, docker-compose.yml.

---

## Why OneConfig?

| Before | After |
|--------|-------|
| Read a 50-line README | `oneconfig up` |
| Install Node.js 20, Python 3.11 | Automatic |
| Run `npm install` in 3 directories | Automatic |
| Copy `.env.example` to `.env` | Automatic |
| Start Postgres, Redis, the API | Automatic |
| Run migrations | Automatic |
| 30 minutes | 2 minutes |

---

## Contributing

We love contributions! OneConfig's universal detection engine is highly extensible. If your favorite framework, language, or tool isn't auto-detected yet, you can easily add it.

Check out our [Contributing Guide](CONTRIBUTING.md) to get started.

---

## Development

```bash
# Run tests
go test ./...

# Build
go build -o oneconfig .

# Run
./oneconfig generate
```

---

## License

[MIT](LICENSE)
