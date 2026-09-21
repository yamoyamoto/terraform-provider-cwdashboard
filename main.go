package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/yamoyamoto/terraform-provider-cwdashboard/internal/provider"
)

var (
	version string = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/yamoyamoto/cwdashboard",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
