package main

import (
	"github.com/alecthomas/kong"
	"github.com/chiselstrike/iku-turso-cli/cmd"
)

type CLI struct {
	cmd.Globals

	Auth cmd.AuthCmd `cmd:"" help:"Authenticate with ChiselEdge"`
	Db   cmd.DbCmd   `cmd:"" help:"Manage ChiselEdge databases"`
}

func main() {
	cli := CLI{
		Globals: cmd.Globals{
			Version: cmd.VersionFlag("0.0.0"),
		},
	}
	ctx := kong.Parse(&cli,
		kong.Name("ikuctl"),
		kong.Description("ChiselEdge CLI"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": "0.0.1",
		})
	err := ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
