package main

import (
	"github.com/alecthomas/kong"
	"github.com/chiselstrike/iku-turso-cli/internal/cmd"
	"github.com/willabides/kongplete"
	"os"
)

type CLI struct {
	cmd.Globals

	Auth        cmd.AuthCmd                  `cmd:"" help:"Authenticate with ChiselEdge"`
	Db          cmd.DbCmd                    `cmd:"" help:"Manage ChiselEdge databases"`
	Completions kongplete.InstallCompletions `cmd:"" help:"Install shell completions"`
}

func main() {
	cli := CLI{
		Globals: cmd.Globals{
			Version: cmd.VersionFlag("0.0.0"),
		},
	}
	parser := kong.Must(&cli,
		kong.Name("ikuctl"),
		kong.Description("ChiselEdge CLI"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": "0.0.1",
		},
	)
	kongplete.Complete(parser)
	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	err = ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
