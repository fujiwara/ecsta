package ecsta

import (
	"encoding/json"
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

type ssmRequestParameters struct {
	Target string
}

func (p *ssmRequestParameters) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func buildSsmRequestParameters(task types.Task, targetContainer string) (*ssmRequestParameters, error) {
	values := strings.Split(*task.TaskArn, "/")
	clusterName := values[1]
	taskID := values[2]
	var runtimeID string
	for _, c := range task.Containers {
		if *c.Name == targetContainer {
			runtimeID = *c.RuntimeId
		}
	}
	return &ssmRequestParameters{
		Target: fmt.Sprintf("ecs:%s_%s_%s", clusterName, taskID, runtimeID),
	}, nil
}
