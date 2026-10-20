package recipes

import (
	"context"
	"errors"
	"fmt"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region http-error
func ReadDisks(parent context.Context, client *hexcore.Client) ([]hexcore.Disk, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	disks, err := client.Disks.List(ctx)
	if err != nil {
		var apiErr *hexcore.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("HTTP %d: %s (Retry-After: %s)\n",
				apiErr.StatusCode, apiErr.Message, apiErr.Headers.Get("Retry-After"))
		}
		return nil, err
	}
	return disks, nil
}

// #endregion http-error

// #region action-error
func CheckAction(parent context.Context, client *hexcore.Client, actionID string) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	_, err := client.Actions.Wait(ctx, actionID, hexcore.WaitOptions{})
	var failed *hexcore.ActionError
	if errors.As(err, &failed) {
		fmt.Printf("Action %s failed\n", failed.Action.ID)
		if failed.Action.Message != nil {
			fmt.Println(*failed.Action.Message)
		}
	}
	return err
}

// #endregion action-error
