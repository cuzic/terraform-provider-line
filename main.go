// Command terraform-provider-line is the Terraform Plugin Framework server
// entrypoint for the "line" provider.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/cuzic/terraform-provider-line/internal/provider"
)

// version is overridden at build time via:
//
//	go build -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "start provider in debug mode, for use with delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/cuzic/line",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
