# Configuration Reference

This document describes all available fields in `oneconfig.yml`.

## `project_name` (string, required)

A human-readable name for your project.

```yaml
project_name: my-app
```

## `runtimes` (list)

Language runtimes required by the project.

- `name`: (string) `node` or `python`
- `version`: (string) The exact or major version (e.g., `20.x`, `3.11`).

```yaml
runtimes:
  - name: node
    version: "20.x"
```

## `package_managers` (list)

Package managers to run during setup.

- `type`: (string) `npm`, `yarn`, `pnpm`, `pip`, `poetry`, or `uv`
- `path`: (string, optional) Directory relative to project root. Defaults to `.`.
- `install_command`: (string, optional) Override the default install command.

```yaml
package_managers:
  - type: npm
    path: ./frontend
```

## `env_vars` (map)

Environment variables to set. Variables are merged into a local `.env` file and will not overwrite existing user values.
Use a `- file: <path>` item to merge variables from a template file.

```yaml
env_vars:
  DATABASE_URL: postgres://localhost:5432/db
  - file: .env.example
```

## `services` (list)

Local services that need to be running for development.

- `name`: (string, required) Unique name for the service.
- `type`: (string) `process`, `docker`, or `script`. Defaults to `process`.
- `start_command`: (string, required) Shell command to start the service.
- `port`: (int) Port the service listens on (used for `oneconfig status`).
- `health_check`: (object, optional) Check to verify the service is running.
  - `type`: `http`, `tcp`, or `cmd`
  - `target`: URL, port string, or command
  - `timeout`: e.g. `30s`
  - `interval`: e.g. `2s`
- `depends_on`: (list of strings) Names of services that must be healthy first.
- `working_dir`: (string) Directory to run the command in.
- `env`: (map) Additional environment variables for this service only.

```yaml
services:
  - name: api
    start_command: npm run dev
    port: 3000
    health_check:
      type: http
      target: http://localhost:3000/health
```

## `setup_steps` (list)

One-off commands to run during setup, after services are started.

- `name`: (string, required) Label for the step.
- `command`: (string, required) Shell command to execute.
- `depends_on`: (list of strings) Names of services or other steps that must complete first.
- `working_dir`: (string) Directory to run the command in.

```yaml
setup_steps:
  - name: migrate
    command: npm run migrate
    depends_on: [api]
```

## `health_checks` (list)

Final health checks to confirm the environment is fully ready.

- `name`: (string) Label for the check.
- `url`: (string) HTTP GET URL, expects 2xx.
- `port`: (int) TCP port to dial.
- `command`: (string) Shell command, expects exit 0.
- `timeout`: (string) e.g., `60s`
- `interval`: (string) e.g., `5s`

```yaml
health_checks:
  - name: frontend
    url: http://localhost:3000
```

## `post_start_commands` (list)

Shell commands to print or execute after the environment is fully ready.

```yaml
post_start_commands:
  - echo "🚀 Project is ready!"
```
