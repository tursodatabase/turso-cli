package cmd

import (
	"github.com/pkg/browser"
)

type AuthCmd struct {
	Login LoginCmd `cmd:"" help:"Login to the platform."`
}

type LoginCmd struct {
}

func (cmd *LoginCmd) Run(globals *Globals) error {
	return browser.OpenURL("https://api.chiseledge.com")
}
