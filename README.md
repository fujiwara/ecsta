# ecsta

ecsta is an "ECS Task Assistant" tool.

## Product status

Production ready.

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
  -h, --help                        Show context-sensitive help.
  -c, --cluster=STRING              ECS cluster name ($ECS_CLUSTER)
  -r, --region=STRING               AWS region ($AWS_REGION)
  -o, --output="table"              output format (table, tsv, json) ($ECSTA_OUTPUT)
  -q, --task-format-query=STRING    A jq query to format task in selector
                                    ($ECSTA_TASK_FORMAT_QUERY)

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
  -f, --family=FAMILY               Task definition family
  -s, --service=SERVICE             Service name
      --output-tags                 Output tags of tasks
      --tags=KEY=VALUE,...          Show only tasks that have specified tags
```

```console
$ ecsta list --cluster foo
|                ID                |   TASKDEFINITION   | INSTANCE | LASTSTATUS | DESIREDSTATUS |         CREATEDAT         |        GROUP        |  TYPE   |
+----------------------------------+--------------------+----------+------------+---------------+---------------------------+---------------------+---------+
| 38b0db90fd4c4b5aaff29288b2179b5a | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-05T09:59:27+09:00 | service:nginx-local | FARGATE |
| 4deeb701c49a4892b7de39a2d0df17e0 | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-06T00:12:50+09:00 | service:nginx-local | FARGATE |
```

```console
$ ecsta list --cluster foo --output-tags --tags Env=prod
|                ID                |   TASKDEFINITION   | INSTANCE | LASTSTATUS | DESIREDSTATUS |         CREATEDAT         |        GROUP        |  TYPE   |           TAGS            |
+----------------------------------+--------------------+----------+------------+---------------+---------------------------+---------------------+---------+---------------------------+
| 38b0db90fd4c4b5aaff29288b2179b5a | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-05T09:59:27+09:00 | service:nginx-local | FARGATE | Env=prod,Name=nginx-local |
| 4deeb701c49a4892b7de39a2d0df17e0 | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-06T00:12:50+09:00 | service:nginx-local | FARGATE | Env=prod,Name=nginx-local |
```

### Describe task

```
Usage: ecsta describe

Describe tasks

Flags:
      --id=STRING          task ID
      --family=FAMILY      task definition family name
      --service=SERVICE    ECS service name
```

### Exec task

```
Usage: ecsta exec

Execute a command on a task

Flags:
      --id=STRING           task ID
      --command="sh"        command to execute
      --container=STRING    container name
      --family=FAMILY       task definition family name
      --service=SERVICE     ECS service name
```

### Portforward task

`--local-port` and `--remote-port`, or `-L` is required.

```
Usage: ecsta portforward

Forward a port of a task

Flags:
      --id=STRING                   task ID
      --container=STRING            container name
      --local-port=INT              local port
      --remote-port=INT             remote port
      --remote-host=STRING          remote host
  -L, --L=STRING                    short expression of local-port:remote-host:remote-port
      --family=FAMILY               task definition family name
      --service=SERVICE             ECS service name
```

An example of port forwarding. Forward a port 8080 of a task to 80 of example.com.

```console
$ ecsta portforward --local-port 8080 --remote-port 80 --remote-host example.com

$ ecsta portforward -L 8080:example.com:80
```

ecsta connects to the task and starts a port forwarding. You can access the port 8080 of the local machine.

```console
$ curl -H"Host: example.com" http://localhost:8080
```

### Stop task

```
Usage: ecsta stop

Stop a task

Flags:
      --id=STRING         task ID
      --force             stop without confirmation
      --family=FAMILY     task definition family name
      --service=SERVICE   ECS service name
```

### Trace task

Run [tracer](https://github.com/fujiwara/tracer). No need to install `tracer` command.

```
Usage: ecsta trace [flags]

Trace a task

Flags:
      --id=STRING                   task ID
  -d, --duration=1m                 duration to trace
      --sns-topic-arn=STRING        SNS topic ARN
      --family=FAMILY               task definition family name
      --service=SERVICE             ECS service name
  -j, --json                        output JSON lines
```

### Logs

```
Usage: ecsta logs

Show log messages of a task

Flags:
      --id=STRING                   task ID
  -s, --start-time=STRING           a start time of logs
  -d, --duration=1m                 log timestamps duration
  -f, --follow                      follow logs
      --container=STRING            container name
      --family=FAMILY               task definition family name
      --service=SERVICE             ECS service name
  -j, --json                        output as JSON lines
```

`--start-time` accepts flexible time formats (ISO8601, RFC3339, and etc). See also (tkuchiki/parsetime)[https://github.com/tkuchiki/parsetime].

When `--start-time` and `--follow` is specified both, `--start-time` may not work correctly.

### `--task-format-query(-q)` option

This option provides a formatter by [jq](https://stedolan.github.io/jq/) query. The query processes tasks JSON (that output equals to ecsta describe) in task selector outputs.

For example,

```console
$ ecsta -q '[(.tags[]|select(.key=="Env")|.value), .launchType] | @tsv' exec
```

A task selector output will be as below.

```console
045a0639-1dc5-4d17-8101-2dd3fd339e91    prod    EC2
8f431e68-a57d-41db-ae8d-5eb700a134dc    dev     FARGATE
```

The query `[(.tags[]|select(.key=="Env")|.value), .launchType] | @tsv` means,
"Show tags value of "Env" key, and LaunchType for tasks as TSV format.".

## LICENSE

[MIT](LICENSE)
