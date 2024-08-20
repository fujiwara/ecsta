package ecsta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Songmu/flextime"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/tkuchiki/parsetime"
)

type LogsOption struct {
	ID        string        `help:"task ID"`
	StartTime string        `help:"a start time of logs" short:"s"`
	Duration  time.Duration `help:"log timestamps duration" short:"d" default:"1m"`
	Follow    bool          `help:"follow logs" short:"f"`
	Container string        `help:"container name"`
	Family    *string       `help:"task definition family name"`
	Service   *string       `help:"ECS service name"`
	JSON      bool          `help:"output as JSON lines" short:"j"`
}

func (opt *LogsOption) ResolveTimestamps() (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	if opt.StartTime != "" {
		p, err := parsetime.NewParseTime()
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to create parsetime: %w", err)
		}
		t, err := p.Parse(opt.StartTime)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse start time: %w", err)
		}
		startTime = t
		endTime = t.Add(opt.Duration)
	} else {
		now := flextime.Now()
		startTime = now.Add(-opt.Duration)
		endTime = now
	}
	if opt.Follow {
		endTime = time.Time{}
	}
	return startTime, endTime, nil
}

type logRecord struct {
	Time      string `json:"time"`
	Container string `json:"container"`
	Msg       string `json:"msg"`
}

type logEncoder interface {
	Encode(v *logRecord) error
}

type logTextEncoder struct {
	w io.Writer
}

func (e *logTextEncoder) Encode(v *logRecord) error {
	_, err := fmt.Fprintln(e.w, strings.Join([]string{v.Time, v.Container, v.Msg}, "\t"))
	return err
}

type logJSONEncoder struct {
	enc *json.Encoder
}

func (e *logJSONEncoder) Encode(v *logRecord) error {
	return e.enc.Encode(v)
}

func newLogEncoder(w io.Writer, jsonFormat bool) logEncoder {
	if jsonFormat {
		return &logJSONEncoder{enc: json.NewEncoder(w)}
	}
	return &logTextEncoder{w: w}
}

func (app *Ecsta) RunLogs(ctx context.Context, opt *LogsOption) error {
	if app.Config.Output == "json" {
		opt.JSON = true
	}
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, &optionFindTask{id: opt.ID, family: opt.Family, service: opt.Service})
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	res, err := app.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task.TaskDefinitionArn,
	})
	if err != nil {
		return fmt.Errorf("failed to describe task definition: %w", err)
	}
	var wg sync.WaitGroup
	startTime, endTime, err := opt.ResolveTimestamps()
	if err != nil {
		return err
	}
	follows := 0
	containerNames := make([]string, 0, len(res.TaskDefinition.ContainerDefinitions))
	for _, c := range res.TaskDefinition.ContainerDefinitions {
		name := aws.ToString(c.Name)
		containerNames = append(containerNames, name)
		if opt.Container != "" && opt.Container != name {
			continue
		}
		if c.LogConfiguration == nil {
			continue
		}
		if c.LogConfiguration.LogDriver != types.LogDriverAwslogs {
			continue
		}
		logOpts := c.LogConfiguration.Options
		logGroup := logOpts["awslogs-group"]
		logStream := fmt.Sprintf("%s/%s/%s", logOpts["awslogs-stream-prefix"], *c.Name, arnToName(*task.TaskArn))
		wg.Add(1)
		follows++
		go func() {
			defer wg.Done()
			if err := app.followLogs(ctx, &followOption{
				logGroup:      logGroup,
				logStream:     logStream,
				startTime:     startTime,
				endTime:       endTime,
				follow:        opt.Follow,
				containerName: name,
				json:          opt.JSON,
			}); err != nil {
				slog.Error("failed to follow logs", "error", err)
			}
		}()
	}
	wg.Wait()
	if follows == 0 {
		return fmt.Errorf("no logs found. available containers: %s", strings.Join(containerNames, ", "))
	}
	return nil
}

type followOption struct {
	logGroup      string
	logStream     string
	containerName string
	startTime     time.Time
	endTime       time.Time
	follow        bool
	json          bool
}

func (o followOption) Follow() bool {
	return o.follow || o.endTime.IsZero()
}

func (app *Ecsta) followLogs(ctx context.Context, opt *followOption) error {
	var nextToken *string
	in := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &opt.logGroup,
		LogStreamName: &opt.logStream,
		StartTime:     aws.Int64(timeToInt64msec(opt.startTime)),
		Limit:         aws.Int32(1000),
	}
	if !opt.Follow() {
		in.EndTime = aws.Int64(timeToInt64msec(opt.endTime))
	}
	enc := newLogEncoder(os.Stdout, opt.json)

FOLLOW:
	for {
		if err := sleepWithContext(ctx, time.Second); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if nextToken != nil {
			in = &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &opt.logGroup,
				LogStreamName: &opt.logStream,
				Limit:         aws.Int32(1000),
				NextToken:     nextToken,
			}
		}
		res, err := app.logs.GetLogEvents(ctx, in)
		if err != nil {
			var ne *logsTypes.ResourceNotFoundException
			// log group or log stream not found, retry after 5 seconds
			if errors.As(err, &ne) {
				sleepWithContext(ctx, 5*time.Second)
			} else {
				slog.Warn("failed to get log events", "error", err)
			}
			continue
		}
		for _, e := range res.Events {
			ts := msecToTime(aws.ToInt64(e.Timestamp))
			if !opt.Follow() && ts.After(opt.endTime) {
				break FOLLOW
			}
			if err := enc.Encode(&logRecord{
				Time:      ts.Format(time.RFC3339Nano),
				Msg:       aws.ToString(e.Message),
				Container: opt.containerName,
			}); err != nil {
				return fmt.Errorf("failed to encode log record: %w", err)
			}
		}
		if aws.ToString(nextToken) == aws.ToString(res.NextForwardToken) {
			if !opt.Follow() {
				break FOLLOW
			}
			continue
		}
		nextToken = res.NextForwardToken
	}
	return nil
}

var epoch = time.Unix(0, 0)

func timeToInt64msec(t time.Time) int64 {
	return int64(t.Sub(epoch) / time.Millisecond)
}

func msecToTime(i int64) time.Time {
	return epoch.Add(time.Duration(i) * time.Millisecond)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
