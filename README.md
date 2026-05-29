# <div align="center">🚀 OneConfig</div>

**<div align="center">Set up any dev environment with one command.</div>**

OneConfig provisions a fully working local development environment with zero manual setup. If you don't have a configuration file, it automatically deep-scans your repository to build one from scratch. Once generated, it reads it and installs tool versions, configure databases, and start services. No more reading long READMEs like what you do right now!

> **Universal Detection Engine**: Run `oneconfig generate` in a project, and OneConfig will automatically infer your frameworks, runtimes, package managers, and databases—no manual writing required!

```bash
git clone https://github.com/Thanos2002/Oneconfig.git
cd Oneconfig
oneconfig generate
oneconfig up
```

---

## ⚡ Quick Start

### 1. Install
Install the CLI directly from source using Go:
```bash
go install github.com/Thanos2002/Oneconfig@latest
```
*(Make sure your `GOPATH/bin` is added to your system's PATH).*

### 2. Auto-Generate Configuration
In any of your local projects, let OneConfig figure out your stack:
```bash
cd your-project/
oneconfig generate
```
*(This will scan for package managers, frameworks, and databases, creating a draft `oneconfig.yml` for you).*

### 3. Run
Spin up the entire environment (runtimes, dependencies, databases, and apps):
```bash
oneconfig up
```

---

## 🧠 Universal Detection Engine

OneConfig's `generate` engine has deep, universal auto-detection capabilities. It traverses your directories (even monorepos!) and automatically detects:

- **JavaScript / TypeScript**: Node.js, npm, yarn, pnpm, bun
- **Python**: pip, poetry, uv, pipenv
- **Systems & Backend**: Go, Rust (Cargo), Java (Maven/Gradle), Ruby (Bundler)
- **Web Frameworks**: Next.js, Django, FastAPI, Flask, Streamlit, Gradio, Rails, Sinatra
- **Databases**: Auto-detects dependencies to spin up Postgres, MongoDB, Redis, and MySQL via Docker.
- **Monorepos**: Deep-scans workspaces, packages, and apps directories.
- **Task Runners**: Makefile, Procfile, Dockerfile, docker-compose.yml.

---

## 🛠️ What `oneconfig up` Does

When you run `oneconfig up`, the orchestrator takes over:
1. **Validates** your `oneconfig.yml`
2. **Installs runtimes** (Node.js, Python, Ruby, etc.)
3. **Installs dependencies** (npm, yarn, pip, poetry, bun, cargo, etc.)
4. **Sets environment variables** from config into a local `.env`
5. **Starts services** in dependency order (databases, APIs, workers)
6. **Waits for health checks** before marking the environment ready
7. **Runs post-start commands** (migrations, seed data, etc.)

---

## 💻 Example Config

Here is an example of what `oneconfig generate` might build for a full-stack app:

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

## ⌨️ CLI Commands

| Command | Description |
|---------|-------------|
| `oneconfig generate` | Scan a project and auto-generate a robust `oneconfig.yml` |
| `oneconfig init` | Create a blank starter config manually |
| `oneconfig up` | Set up and start the entire environment |
| `oneconfig down` | Stop all running services |
| `oneconfig status` | Show service status (PID, port, health) |
| `oneconfig doctor` | Check your system for missing underlying tools |

---

## 🤝 Contributing

We love contributions! OneConfig's universal detection engine is highly extensible. If your favorite framework, language, or tool isn't auto-detected yet, you can easily add it by creating a single `detector_*.go` file.

Check out our [Contributing Guide](CONTRIBUTING.md) to get started!

---

## 📄 License

[MIT](LICENSE)
