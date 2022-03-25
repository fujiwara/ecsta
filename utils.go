package ecsta

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
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
