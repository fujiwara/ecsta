package ecsta_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/fujiwara/ecsta"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var ignore = cmpopts.IgnoreUnexported(
	types.KeyValuePair{},
	types.Attachment{},
	types.Attribute{},
	types.ManagedAgent{},
	types.NetworkInterface{},
	types.Container{},
	types.EphemeralStorage{},
	types.ContainerOverride{},
	types.TaskOverride{},
	types.Task{},
)

func TestMarshalUnmarshalTask(t *testing.T) {
	b, err := os.ReadFile("testdata/task.json")
	if err != nil {
		t.Error(err)
	}
	var task types.Task
	if err := ecsta.UnmarshalJSONForStruct(b, &task); err != nil {
		t.Error(err)
	}
	if cpu := aws.ToString(task.Cpu); cpu != "256" {
		t.Errorf("unexpected cpu: %s", cpu)
	}
	if len(task.Containers) != 2 {
		t.Errorf("unexpected number of containers: %d", len(task.Containers))
	}
	if addr := aws.ToString(task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address); addr != "10.3.1.230" {
		t.Errorf("unexpected private ipv4 address: %s", addr)
	}
	if task.EnableExecuteCommand != true {
		t.Errorf("unexpected enable execute command: %v", task.EnableExecuteCommand)
	}
	if task.EphemeralStorage.SizeInGiB != 50 {
		t.Errorf("unexpected ephemeral storage size: %d", task.EphemeralStorage.SizeInGiB)
	}

	b2, err := ecsta.MarshalJSONForAPI(&task, nil)
	if err != nil {
		t.Error(err)
	}
	var task2 types.Task
	if err := ecsta.UnmarshalJSONForStruct(b2, &task2); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(task, task2, ignore); diff != "" {
		t.Error("not equal task", diff)
	}
}

var formatTestTasks = []types.Task{
	{
		TaskArn:              aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task/cluster-name/045a0639-1dc5-4d17-8101-2dd3fd339e91"),
		TaskDefinitionArn:    aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/taskdef-name:123"),
		ContainerInstanceArn: aws.String("arn:aws:ecs:ap-northeast-1:123456789012:container-instance/cluster-name/2ee1c131-7f61-43ab-884a-379e668d31fb"),
		Containers: []types.Container{
			{
				Name: aws.String("web"),
			},
			{
				Name: aws.String("db"),
			},
		},
		CreatedAt:     aws.Time(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)),
		Group:         aws.String("family:taskdef-name"),
		LastStatus:    aws.String("PENDING"),
		DesiredStatus: aws.String("RUNNING"),
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("task-name"),
			},
			{
				Key:   aws.String("Env"),
				Value: aws.String("prod"),
			},
		},
		LaunchType: types.LaunchTypeEc2,
	},
	{
		TaskArn:              aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task/cluster-name/8f431e68-a57d-41db-ae8d-5eb700a134dc"),
		TaskDefinitionArn:    aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/taskdef-name:999"),
		ContainerInstanceArn: aws.String("arn:aws:ecs:ap-northeast-1:123456789012:container-instance/cluster-name/70d14568-f853-4c03-92a5-86ef9b3c0077"),
		Containers: []types.Container{
			{
				Name: aws.String("web"),
			},
			{
				Name: aws.String("db"),
			},
		},
		CreatedAt:     aws.Time(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		Group:         aws.String("service:service-name"),
		LastStatus:    aws.String("PENDING"),
		DesiredStatus: aws.String("RUNNING"),
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("task-name"),
			},
			{
				Key:   aws.String("Env"),
				Value: aws.String("dev"),
			},
		},
		LaunchType: types.LaunchTypeFargate,
	},
}

var formatTestSuite = []struct {
	opt      ecsta.FormatterOption
	wantFile string
}{
	{
		opt: ecsta.FormatterOption{
			Format: "tsv",
		},
		wantFile: "testdata/tasks.tsv",
	},
	{
		opt: ecsta.FormatterOption{
			Format:   "tsv",
			WithTags: true,
		},
		wantFile: "testdata/tasks_withtags.tsv",
	},
	{
		opt: ecsta.FormatterOption{
			Format:    "table",
			HasHeader: true,
		},
		wantFile: "testdata/tasks.table",
	},
	{
		opt: ecsta.FormatterOption{
			Format:    "table",
			HasHeader: true,
			WithTags:  true,
		},
		wantFile: "testdata/tasks_withtags.table",
	},
	{
		opt: ecsta.FormatterOption{
			Format: "json",
		},
		wantFile: "testdata/tasks.json",
	},
	{
		opt: ecsta.FormatterOption{
			Format:       "json",
			Query:        `[(.tags[]|select(.key=="Env")|.value), .launchType] | @tsv`,
			AppendTaskID: true,
		},
		wantFile: "testdata/tasks.queried",
	},
}

func TestFormatTasks(t *testing.T) {
	for _, ts := range formatTestSuite {
		t.Run(fmt.Sprintf("%#v", ts.opt), func(t *testing.T) {
			buf := new(bytes.Buffer)
			if f, err := ecsta.NewTaskFormatter(buf, ts.opt); err != nil {
				t.Error(err)
			} else {
				for _, task := range formatTestTasks {
					f.AddTask(task)
				}
				f.Close()
			}
			b, _ := os.ReadFile(ts.wantFile)
			if got := buf.String(); got != string(b) {
				t.Errorf("got %q, want %q", got, string(b))
			}
		})
	}
}
