package main

import (
	"flag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"terraform-provider-fptcloud/fptcloud"
)

func main() {
	var debugMode bool

	flag.BoolVar(
		&debugMode,
		"debug",
		false,
		"set to true to run the provider with support for debuggers",
	)
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: fptcloud.Provider}

	if debugMode {
		opts.Debug = true
		opts.ProviderAddr = "github.com/terraform-providers/fptcloud"
	}

	plugin.Serve(opts)
}
