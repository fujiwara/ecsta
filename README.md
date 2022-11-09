# ecsta

ecsta is an "ECS Task Assistant" tool.

## Product status

alpha quality.

## Install

### Homebrew

```
$ brew install fujiwara/tap/ecsta
```

### [Binary releases](https://github.com/fujiwara/ecsta/releases)

## Usage

```
Usage: ecsta <command>

Flags:
  -h, --help              Show context-sensitive help.
  -c, --cluster=STRING    ECS cluster name ($ECS_CLUSTER)
  -r, --region=STRING     AWS region ($AWS_REGION)
  -o, --output="table"    output format (table, tsv, json)

Commands:
  configure
    Create a configuration file of ecsta

  describe
    Describe tasks

  exec
    Execute a command on a task

  list
    List tasks

  logs
    Show log messages of a task

  portforward --local-port=INT --remote-port=INT
    Forward a port of a task

  stop
    Stop a task

  trace
    Trace a task

  version
    Show version
```

### Configuration

ecsta is a zero configuration command. But you can use a configuration file (`~/.config/ecsta/config.json`) to set default options.

Run an interactive setting as below.
```console
$ ecsta configure
```

```console
$ ecsta configure --show
2022/08/08 15:36:54 configuration file: /home/fujiwara/.config/ecsta/config.json
{
  "filter_command": "peco",
  "output": "tsv"
}
```

### List tasks

```
Usage: ecsta list

List tasks

Flags:
  -f, --family=STRING     Task definition family
  -s, --service=STRING    Service name
```

```console
$ ecsta list --cluster foo
|                ID                |   TASKDEFINITION   | INSTANCE | LASTSTATUS | DESIREDSTATUS |         CREATEDAT         |        GROUP        |  TYPE   |
+----------------------------------+--------------------+----------+------------+---------------+---------------------------+---------------------+---------+
| 38b0db90fd4c4b5aaff29288b2179b5a | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-05T09:59:27+09:00 | service:nginx-local | FARGATE |
| 4deeb701c49a4892b7de39a2d0df17e0 | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-06T00:12:50+09:00 | service:nginx-local | FARGATE |
```

### Describe task

```
Usage: ecsta describe

Describe tasks

Flags:
      --id=STRING         task ID
```

### Exec task

```
Usage: ecsta exec

Execute a command on a task

Flags:
      --id=STRING           task ID
      --command="sh"        command to execute
      --container=STRING    container name
```

### Portforward task

`-port` and `-localport` are required.

`-host` does not work currently. (ssm-agent version run by ECS Exec is too old)

```
Usage: ecsta portforward --local-port=INT --remote-port=INT

Forward a port of a task

Flags:
      --id=STRING             task ID
      --container=STRING      container name
      --local-port=INT        local port
      --remote-port=INT       remote port
      --remote-host=STRING    remote host
```

### Stop task

```
Usage: ecsta stop

Stop a task

Flags:
      --id=STRING         task ID
      --force             stop without confirmation
```

### Trace task

Run [tracer](https://github.com/fujiwara/tracer). No need to install `tracer` command.

```
Usage: ecsta trace

Trace a task

Flags:
      --id=STRING               task ID
      --duration=1m             duration to trace
      --sns-topic-arn=STRING    SNS topic ARN
```

### Logs

```
Usage: ecsta logs

Show log messages of a task

Flags:
      --id=STRING           task ID
      --duration=1m         duration to start time
  -f, --follow              follow logs
      --container=STRING    container name
```

## LICENSE

[MIT](LICENSE)
