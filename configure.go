package ecsta

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
)

type ConfigureCmd struct {
	app *Ecsta

	show bool
}

func NewConfigureCmd(app *Ecsta) *ConfigureCmd {
	return &ConfigureCmd{
		app: app,
	}
}

func (*ConfigureCmd) Name() string     { return "configure" }
func (*ConfigureCmd) Synopsis() string { return "configure ecsta" }
func (*ConfigureCmd) Usage() string {
	return `configure [options]:
  Configure ecsta.
`
}

func (p *ConfigureCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.show, "show", false, "show a current configuration")
}

func (p *ConfigureCmd) execute(ctx context.Context) error {
	if p.show {
		log.Println("configuration file:", configFilePath())
		fmt.Fprintln(p.app.w, p.app.config.String())
		return nil
	}
	if err := reConfigure(p.app.config); err != nil {
		return err
	}
	return nil
}

func (p *ConfigureCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
