package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	client, err := hexcore.NewClient(hexcore.Config{Token: os.Getenv("HEXCORE_TOKEN")})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	servers, err := client.VDS.List(ctx)
	if err != nil {
		return err
	}
	for _, server := range servers {
		fmt.Printf("%s: %s\n", server.ID, server.Name)
	}
	return nil
}
