package ecsta

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Songmu/flextime"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/tkuchiki/parsetime"
)

type LogsOption struct {
	ID        string        `help:"task ID"`
	StartTime string        `help:"a start time of logs" short:"s"`
	Duration  time.Duration `help:"duration to start time" short:"d" default:"1m"`
	Follow    bool          `help:"follow logs" short:"f"`
	Container string        `help:"container name"`
	Family    *string       `help:"task definiton family name"`
	Service   *string       `help:"ECS service name"`
}

func (opt *LogsOption) ResolveTimestamps() (time.Time, time.Time, error) {
	if opt.StartTime != "" {
		p, err := parsetime.NewParseTime()
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to create parsetime: %w", err)
		}
		t, err := p.Parse(opt.StartTime)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse start time: %w", err)
		}
		return t, t.Add(opt.Duration), nil
	}
	now := flextime.Now()
	return now.Add(-opt.Duration), now, nil
}

func (app *Ecsta) RunLogs(ctx context.Context, opt *LogsOption) error {
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
			}); err != nil {
				log.Println(err)
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
}

func (app *Ecsta) followLogs(ctx context.Context, opt *followOption) error {
	var nextToken *string
	in := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &opt.logGroup,
		LogStreamName: &opt.logStream,
		StartTime:     aws.Int64(timeToInt64msec(opt.startTime)),
		Limit:         aws.Int32(1000),
	}
	if !opt.follow {
		in.EndTime = aws.Int64(timeToInt64msec(opt.endTime))
	}
FOLLOW:
	for {
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
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		for _, e := range res.Events {
			ts := msecToTime(aws.ToInt64(e.Timestamp))
			if ts.After(opt.endTime) {
				break FOLLOW
			}
			fmt.Println(strings.Join([]string{
				ts.Format(time.RFC3339Nano),
				opt.containerName,
				aws.ToString(e.Message),
			}, "\t"))
		}
		if aws.ToString(nextToken) == aws.ToString(res.NextForwardToken) {
			if !opt.follow {
				break
			}
			time.Sleep(time.Second)
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
