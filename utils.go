package ecsta

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func arnToResource(s string) string {
	an, err := arn.Parse(s)
	if err != nil {
		return s
	}
	return an.Resource
}

func arnToName(s string) string {
	ns := strings.Split(s, "/")
	return ns[len(ns)-1]
}

func ssmRequestTarget(task types.Task, targetContainer string) (string, error) {
	values := strings.Split(*task.TaskArn, "/")
	clusterName := values[1]
	taskID := values[2]
	var runtimeID string
	for _, c := range task.Containers {
		if *c.Name == targetContainer {
			runtimeID = *c.RuntimeId
		}
	}
	return fmt.Sprintf("ecs:%s_%s_%s", clusterName, taskID, runtimeID), nil
}
