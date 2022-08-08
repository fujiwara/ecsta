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
Usage: ecsta <flags> <subcommand> <subcommand args>

Subcommands:
        configure        configure ecsta
        describe         describe task
        exec             exec task
        flags            describe all known top-level flags
        list             liste tasks
        portforward      port forwarding
        stop             stop task

Use "ecsta flags" for a list of top-level flags
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
$ ecsta list -h
list -cluster <cluster> [options]:
  Show task ARNs in the cluster.
  -family string
        task definition family
  -output string
        output format (table|json|tsv)
```

```console
$ ecsta -cluster foo list
|                ID                |   TASKDEFINITION   | INSTANCE | LASTSTATUS | DESIREDSTATUS |         CREATEDAT         |        GROUP        |  TYPE   |
+----------------------------------+--------------------+----------+------------+---------------+---------------------------+---------------------+---------+
| 38b0db90fd4c4b5aaff29288b2179b5a | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-05T09:59:27+09:00 | service:nginx-local | FARGATE |
| 4deeb701c49a4892b7de39a2d0df17e0 | ecspresso-test:499 |          | RUNNING    | RUNNING       | 2022-08-06T00:12:50+09:00 | service:nginx-local | FARGATE |
```

### Describe task

```
describe [options]:
  Describe a task in a cluster.
  -id string
        task ID
```

### Exec task

```
exec [options]:
  ECS Exec task in a cluster.
  -command string
        command (default "sh")
  -container string
        container name
  -id string
        task ID
```

### Portfoward task

`-port` and `-localport` are required.

`-host` does not work currently. (ssm-agent version run by ECS Exec is too old)

```
portforward [options]:
  Port forwarding to a task in a cluster.
  -container string
        container name
  -host string
        remote host
  -id string
        task ID
  -localport int
        local port
  -port int
        remote port
```

### Stop task

```
stop [options]:
  Stop task in a cluster.
  -force
        stop a task without confirmation
  -id string
        task ID
```

## LICENSE

[MIT](LICENSE)
