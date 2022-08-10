# ecsta

ecsta is a "ECS Task Assistant" tool.

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
NAME:
   ecsta - ECS task assistant

USAGE:
   ecsta [global options] command [command options] [arguments...]

COMMANDS:
   configure    Create a configuration file of ecsta
   describe     describe task
   exec         exec task
   list         List tasks
   portforward  forward port to task
   stop         stop task
   trace        trace task
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --cluster value, -c value  ECS cluster name [$ECS_CLUSTER]
   --help, -h                 show help (default: false)
   --output value, -o value   Output format (table, tsv, json)
   --region value, -r value   AWS region (default: "ap-northeast-1") [$AWS_REGION]
```

### Configuration

ecsta is a zero configuration command. But you can use a configuration file (`~/.config/ecsta/config.json`) to set default options.

Run an interactive setting as below.
```console
$ ecsta configure
```

```console
$ ecsta configure -show
2022/08/08 15:36:54 configuration file: /home/fujiwara/.config/ecsta/config.json
{
  "filter_command": "peco",
  "output": "tsv"
}
```

### List tasks

```
NAME:
   ecsta list - List tasks

USAGE:
   ecsta list [command options] [arguments...]

OPTIONS:
   --family value, -f value   Task definition family
   --service value, -s value  Service name
```

```console
$ ecsta list -cluster foo
|                ID                |   TASKDEFINITION   | INSTANCE | LASTSTATUS | DESIREDSTATUS |         CREATEDAT         |        GROUP        |  TYPE   |
+----------------------------------+--------------------+----------+------------+---------------+---------------------------+---------------------+---------+
| 38b0db90fd4c4b5aaff29288b2179b5a | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-05T09:59:27+09:00 | service:nginx-local | FARGATE |
| 4deeb701c49a4892b7de39a2d0df17e0 | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-06T00:12:50+09:00 | service:nginx-local | FARGATE |
```

### Describe task

```
NAME:
   ecsta describe - describe task

USAGE:
   ecsta describe [command options] [arguments...]

OPTIONS:
   --id value                 task ID
```

### Exec task

```
NAME:
   ecsta exec - exec task

USAGE:
   ecsta exec [command options] [arguments...]

OPTIONS:
   --command value            command to execute (default: "sh")
   --container value          container name
   --id value                 task ID
```

### Portfoward task

`-port` and `-localport` are required.

`-host` does not work currently. (ssm-agent version run by ECS Exec is too old)

```
NAME:
   ecsta portforward - forward port to task

USAGE:
   ecsta portforward [command options] [arguments...]

OPTIONS:
   --container value          container name
   --host value               remote host
   --id value                 task ID
   --local-port value         local port (default: 0)
   --port value               remote port (default: 0)
```

### Stop task

```
NAME:
   ecsta stop - stop task

USAGE:
   ecsta stop [command options] [arguments...]

OPTIONS:
   --force                    stop without confirmation (default: false)
   --id value                 task ID
```

### Trace task

Run [tracer](https://github.com/fujiwara/tracer). No need to install `tracer` command.

```
NAME:
   ecsta trace - trace task

USAGE:
   ecsta trace [command options] [arguments...]

OPTIONS:
   --duration value           duration to trace (default: 1m0s)
   --id value                 task ID
   --sns-topic-arn value      SNS topic ARN
```

### Logs

```
NAME:
   ecsta logs - show log messages of task

USAGE:
   ecsta logs [command options] [arguments...]

OPTIONS:
   --duration value           duration to start time (default: 1m0s)
   --follow, -f               follow logs (default: false)
   --id value                 task ID
```

## LICENSE

[MIT](LICENSE)
