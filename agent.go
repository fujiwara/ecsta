package ecsta

import (
	"context"

	"github.com/fujiwara/grpcp"
)

type AgentOption struct {
}

func (app *Ecsta) RunAgent(ctx context.Context, opt *AgentOption) error {
	o := &grpcp.ServerOption{
		Listen: "127.0.0.1",
		Port:   8022,
	}
	go func() {
		err := grpcp.RunServer(context.TODO(), o)
		if err != nil {
			panic(err)
		}
	}()
	<-ctx.Done()
	return nil
}
