# ecsta Project-Specific Instructions

## Project Overview
ecsta is a CLI tool for operating Amazon ECS (Elastic Container Service) tasks. It supports a wide range of operations including listing tasks, viewing details, executing commands, displaying logs, port forwarding, and file transfers.

## Development Guidelines

### Logging
- Use the standard `log/slog` package for logging output
- Text format logs are formatted using `github.com/fujiwara/sloghandler`
- Log format can be switched between `text` (default) and `json` using the `--log-format` flag
- Colored logs are automatically enabled in TTY environments

### Log Levels
- `Debug`: Detailed debug information (architecture detection, retries, etc.)
- `Info`: Normal operational information (connections, transfers, config file paths, etc.)
- `Warn`: Warnings (unknown architecture, task stop notices, etc.)
- `Error`: Errors (connection failures, boot failures, etc.)

### Structured Logging
Use structured logging format (key-value pairs) for all log output:
```go
slog.Info("message", "key1", value1, "key2", value2)
slog.Error("failed to do something", "error", err)
```

### Testing and Building
Before committing changes, execute the following:
- `go build -o ecsta ./cmd/ecsta/` - Ensure build succeeds
- `make test` or `go test ./...` - Ensure tests pass
- `make download-assets` - Download required tncl assets for file transfer functionality

**CI/CD Information:**
- Tests run on Go 1.23 and 1.24 in GitHub Actions
- Releases are built with Go 1.24
- Tests run with `TZ=Asia/Tokyo` for time.Local consistency
- The `make download-assets` step downloads tncl binaries required for the cp command

### Git Operations
- Create a new branch from main when adding new features
- Write commit messages in English with clear descriptions
- feat: new feature
- fix: bug fix
- refactor: refactoring
- docs: documentation update

## Architecture

### Core Struct
```go
type Ecsta struct {
    Config  *Config                // Configuration
    region  string                 // AWS region
    cluster string                 // ECS cluster name
    awscfg  aws.Config            // AWS configuration
    ecs     *ecs.Client           // ECS client
    ssm     *ssm.Client           // SSM client
    logs    *cloudwatchlogs.Client // CloudWatch Logs client
    w       io.Writer             // Output destination
}
```

### Available Commands
- `configure`: Create a configuration file
- `describe`: Show task details
- `exec`: Execute a command in a task (using Session Manager)
- `list`: List tasks
- `logs`: Show task logs (CloudWatch Logs)
- `portforward`: Forward a port from a task
- `stop`: Stop a task
- `trace`: Trace a task (X-Ray integration)
- `cp`: Copy files between tasks and local
- `version`: Show version

## Coding Conventions

### Naming Conventions
- Interface names use `-er` suffix (e.g., `taskFormatter`)
- Option structs use `Option` suffix (e.g., `ExecOption`)
- Internal struct fields start with lowercase

### Error Handling
```go
// Always return error messages with context
if err != nil {
    return fmt.Errorf("failed to list tasks in cluster %s: %w", app.cluster, err)
}
```

### Design Patterns
- **Factory Pattern**: Formatter creation (`newTaskFormatter`)
- **Strategy Pattern**: Different output format implementations
- **Context Usage**: All major functions accept context as the first parameter

### Testing
- Use table-driven tests
- Use `google/go-cmp` for struct comparisons
- Place test data in `testdata/` directory

### Configuration Management
- XDG_CONFIG_HOME compliant configuration file placement (~/.config/ecsta/config.json)
- JSON format configuration file
- CLI options can override configuration

## Dependencies

### AWS SDK
- `aws-sdk-go-v2/service/ecs`: ECS task management
- `aws-sdk-go-v2/service/ssm`: Session Manager integration
- `aws-sdk-go-v2/service/cloudwatchlogs`: Log retrieval

### Major Libraries
- `github.com/alecthomas/kong`: CLI framework
- `github.com/fujiwara/sloghandler`: Log formatter
- `github.com/itchyny/gojq`: JSON query processing (jq-like)
- `github.com/olekukonko/tablewriter`: Table format output
- `github.com/Songmu/prompter`: Interactive input handling
- `github.com/creack/pty`: Pseudo terminal handling
- `github.com/mattn/go-isatty`: TTY detection
- `github.com/fujiwara/tracer`: X-Ray tracing

## Special Implementations

### Session Manager Plugin Integration
- Launch Session Manager Plugin as an external process
- Support PTY usage in non-interactive mode

### File Transfer (cp command)
- Launch agent (tncl) inside the task
- Secure transfer using port forwarding
- Progress bar display (using `progressbar`)
- Requires tncl binaries (downloaded via `make download-assets`)
- Supports ARM64 and x86_64 architectures

### Log Streams
- Concurrent fetching of multiple log streams
- Real-time follow feature (`--follow`)
- Time range specification (`--start`, `--end`)