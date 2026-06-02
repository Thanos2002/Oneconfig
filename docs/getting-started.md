# Getting Started with OneConfig

Welcome to OneConfig! This guide will help you set up OneConfig for your project in just a few minutes.

## 1. Install OneConfig

Choose the installation method for your operating system:

**From Source (requires Go 1.26+):**
```bash
go install github.com/Thanos2002/Oneconfig@latest
```

Verify your installation:
```bash
oneconfig doctor
```

## 2. Initialize a Project

Navigate to an existing project directory and run:

```bash
cd my-project
oneconfig generate
```

This command will scan your project for `package.json`, `requirements.txt`, `.env` templates, and `docker-compose.yml` to generate a draft `oneconfig.yml`.

Alternatively, create a new config from scratch:

```bash
oneconfig init
```

## 3. Customize Your Config

Open `oneconfig.yml` in your editor. A typical configuration looks like this:

```yaml
project_name: my-project

runtimes:
  - name: node
    version: "20.x"

package_managers:
  - type: npm
    path: .

env_vars:
  NODE_ENV: development
  # Add other environment variables here

services:
  - name: api
    start_command: npm run dev
    port: 3000
    health_check:
      type: http
      target: http://localhost:3000/health
```

Customize the services, environment variables, and setup steps as needed.

## 4. Run `oneconfig up`

Now, whenever you or a teammate clone the repository, you can bring up the entire environment with a single command:

```bash
oneconfig up
```

OneConfig will:
1. Ensure the correct runtimes are installed.
2. Install package dependencies.
3. Set up your `.env` file.
4. Start your services in the correct order.
5. Wait for all health checks to pass.

To see what's running:
```bash
oneconfig status
```

To shut everything down:
```bash
oneconfig down
```
