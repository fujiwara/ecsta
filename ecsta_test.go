package ecsta

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFilterTasksByService(t *testing.T) {
	task := func(arn, group string) types.Task {
		return types.Task{
			TaskArn: aws.String(arn),
			Group:   aws.String(group),
		}
	}
	taskNoGroup := types.Task{TaskArn: aws.String("arn:none")}

	all := []types.Task{
		task("arn:a1", "service:myapp-a"),
		task("arn:a2", "service:myapp-a"),
		task("arn:b1", "service:myapp-b"),
		task("arn:r1", "family:myapp"),
		task("arn:r2", "custom-group"),
		taskNoGroup,
	}

	tests := []struct {
		name    string
		service string
		want    []types.Task
	}{
		{
			name:    "keep target service and standalone tasks",
			service: "myapp-a",
			want: []types.Task{
				task("arn:a1", "service:myapp-a"),
				task("arn:a2", "service:myapp-a"),
				task("arn:r1", "family:myapp"),
				task("arn:r2", "custom-group"),
				taskNoGroup,
			},
		},
		{
			name:    "no matching service keeps only standalone tasks",
			service: "nonexistent",
			want: []types.Task{
				task("arn:r1", "family:myapp"),
				task("arn:r2", "custom-group"),
				taskNoGroup,
			},
		},
		{
			name:    "sibling service is excluded",
			service: "myapp-b",
			want: []types.Task{
				task("arn:b1", "service:myapp-b"),
				task("arn:r1", "family:myapp"),
				task("arn:r2", "custom-group"),
				taskNoGroup,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterTasksByService(all, tt.service)
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreUnexported(types.Task{})); diff != "" {
				t.Errorf("filterTasksByService mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
