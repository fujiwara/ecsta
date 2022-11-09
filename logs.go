package ecsta

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type LogsOption struct {
	ID       string        `help:"task ID"`
	Duration time.Duration `help:"duration to start time" default:"1m"`
	Follow   bool          `help:"follow logs" short:"f"`
}

func (app *Ecsta) RunLogs(ctx context.Context, opt *LogsOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, opt.ID)
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
	start := time.Now().Add(-opt.Duration)
	for _, c := range res.TaskDefinition.ContainerDefinitions {
		name := aws.ToString(c.Name)
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
		go func() {
			defer wg.Done()
			if err := app.followLogs(ctx, &followOption{
				logGroup:      logGroup,
				logStream:     logStream,
				start:         start,
				follow:        opt.Follow,
				containerName: name,
			}); err != nil {
				log.Println(err)
			}
		}()
	}
	wg.Wait()
	return nil
}

type followOption struct {
	logGroup      string
	logStream     string
	containerName string
	start         time.Time
	follow        bool
}

func (app *Ecsta) followLogs(ctx context.Context, opt *followOption) error {
	var nextToken *string
	in := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &opt.logGroup,
		LogStreamName: &opt.logStream,
		Limit:         aws.Int32(100),
		StartTime:     aws.Int64(timeToInt64msec(opt.start)),
	}
	for {
		if nextToken != nil {
			in.NextToken = nextToken
			in.StartFromHead = nil
			in.StartTime = nil
		}
		res, err := app.logs.GetLogEvents(ctx, in)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		for _, e := range res.Events {
			ts := msecToTime(aws.ToInt64(e.Timestamp))
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
