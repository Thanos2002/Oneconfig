<div align="center">

# 🚀 OneConfig

### Set up any dev environment with one command.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Built_with-Go-00ADD8?logo=go&logoColor=white)](https://go.dev/)


> oneconfig generate   →   scans your project, builds a config

> oneconfig up         →   installs everything, starts all services


**That's it. No Dockerfiles to write. No READMEs to follow. No "works on my machine."**


---

## 💡 The Problem

Every project has the same onboarding pain:

> *"Clone the repo, install Node 20, make sure you have Postgres running, oh and you need Redis too, then copy `.env.example`, run the migrations, seed the database, then…"*

OneConfig eliminates all of that. It deep-scans your repository, infers your entire stack—languages, runtimes, package managers, databases—and generates a single `oneconfig.yml` that describes your full environment. Then it provisions everything automatically, in the right order, with health checks.

**New developer? One command. Done.**

---

## ✨ Why OneConfig?

| | Traditional Setup | With OneConfig |
|---|---|---|
| ⏱️ **Time to first run** | 30–60 min of manual steps | **< 2 minutes** |
| 📖 **Documentation** | Long, fragile setup guides | Self-documenting config |
| 🔄 **Reproducibility** | "Works on my machine" | Deterministic, every time |
| 🧩 **Monorepo support** | DIY scripting | Built-in deep scanning |
| 🏥 **Health checks** | Hope for the best | Automated, per-service |

---

## 🚀 Quick Start

### Prerequisites

- [Go 1.21+](https://go.dev/dl/) installed on your machine

### 1. Install

```bash
go install github.com/Thanos2002/Oneconfig@latest
```

> **Note:** Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is in your system `PATH` so the `oneconfig` command is available globally.

### 2. Generate

Navigate to any project and let OneConfig figure out your stack:

```bash
cd your-project/
oneconfig generate
```

This will scan your codebase and create a tailored `oneconfig.yml`—detecting your runtimes, package managers, frameworks, and databases automatically.

### 3. Launch

```bash
oneconfig up
```

OneConfig will install runtimes, resolve dependencies, start databases, run migrations, boot your app, and verify health checks—all in the correct order.

```
✔ Validated oneconfig.yml
✔ Installed Node.js 20.x
✔ Ran npm install
✔ Started postgres (healthy on :5432)
✔ Ran migrations
✔ Started api (healthy on :3000)

🚀 Environment ready — http://localhost:3000
```

---

## 🧠 Universal Detection Engine

When you run `oneconfig generate`, the engine recursively traverses your project (including monorepos) and automatically identifies:

| Category | Detected Technologies |
|---|---|
| **JavaScript / TypeScript** | Node.js, npm, yarn, pnpm, bun |
| **Python** | pip, Poetry, uv, Pipenv |
| **Backend / Systems** | Go, Rust (Cargo), Java (Maven / Gradle), Ruby (Bundler) |
| **Web Frameworks** | Next.js, Django, FastAPI, Flask, Streamlit, Gradio, Rails, Sinatra |
| **Databases** | PostgreSQL, MongoDB, Redis, MySQL — spun up via Docker |
| **Infrastructure** | Makefile, Procfile, Dockerfile, docker-compose.yml |
| **Monorepos** | Workspaces, `packages/`, `apps/` directories |

> **Extensible by design:** Adding support for a new technology is as simple as creating a single `detector_*.go` file.

---

## ⚙️ The `oneconfig up` Lifecycle

Under the hood, the orchestrator executes a deterministic pipeline:

```
  oneconfig.yml
       │
       ▼
┌──────────────┐
│   Validate   │  ← Schema & dependency checks
└──────┬───────┘
       ▼
┌──────────────┐
│   Runtimes   │  ← Install Node.js, Python, Ruby, etc.
└──────┬───────┘
       ▼
┌──────────────┐
│ Dependencies │  ← npm install, poetry install, cargo build…
└──────┬───────┘
       ▼
┌──────────────┐
│  Env Vars    │  ← Inject variables into .env
└──────┬───────┘
       ▼
┌──────────────┐
│  Services    │  ← Boot in dependency order (DB → API → Workers)
└──────┬───────┘
       ▼
┌──────────────┐
│Health Checks │  ← TCP / HTTP probes per service
└──────┬───────┘
       ▼
┌──────────────┐
│  Post-Start  │  ← Migrations, seeds, notifications
└──────────────┘
```

---

## 📄 Example Configuration

Here's what `oneconfig generate` might produce for a typical full-stack Node.js + Postgres application:

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
  - echo "🚀 Application ready at http://localhost:3000"
```

---

## ⌨️ CLI Reference

| Command | Description |
|:---|:---|
| `oneconfig generate` | Scan the current project and auto-generate `oneconfig.yml` |
| `oneconfig init` | Create a blank starter config for manual editing |
| `oneconfig up` | Provision the full environment and start all services |
| `oneconfig down` | Gracefully stop and tear down all running services |
| `oneconfig status` | Show live service status, PIDs, ports, and health |
| `oneconfig doctor` | Audit your system for missing tools and dependencies |

---

## 🤝 Contributing

We welcome contributions! OneConfig's detection engine is designed to be **highly extensible**—adding support for a new framework or tool typically requires just one new file.

See the [Contributing Guide](CONTRIBUTING.md) for full details on how to get started.

---

<div align="center">

**Built by [Thanos2002](https://github.com/Thanos2002)**

[MIT License](LICENSE)

</div>
