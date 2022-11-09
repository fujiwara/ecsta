package ecsta

import (
	"context"
	"fmt"
	"log"
)

type ConfigureOption struct {
	Show bool `help:"show current configuration" short:"s"`
}

func (app *Ecsta) RunConfigure(ctx context.Context, cmd *ConfigureOption) error {
	if cmd.Show {
		log.Println("configuration file:", configFilePath())
		fmt.Fprintln(app.w, app.Config.String())
		return nil
	}
	if err := reConfigure(app.Config); err != nil {
		return err
	}
	return nil
}
