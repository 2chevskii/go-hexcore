package hexcore_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

func ExampleNewClient() {
	client, err := hexcore.NewClient(hexcore.Config{Token: os.Getenv("HEXCORE_TOKEN")})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	regions, err := client.Regions.List(ctx)
	if err != nil {
		return
	}
	for _, region := range regions {
		fmt.Println(region.Slug, region.Available)
	}
}

func ExampleActionsService_Wait() {
	client, err := hexcore.NewClient(hexcore.Config{Token: os.Getenv("HEXCORE_TOKEN")})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	action, err := client.VDS.Reboot(ctx, "123")
	if err != nil {
		return
	}
	_, err = client.Actions.Wait(ctx, action.ID, hexcore.WaitOptions{})
	var failure *hexcore.ActionError
	if errors.As(err, &failure) {
		fmt.Println("Action failed:", failure.Action.ID)
	}
}
