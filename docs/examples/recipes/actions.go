package recipes

import (
	"context"
	"fmt"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region restart
func Restart(parent context.Context, client *hexcore.Client, serverID string) (*hexcore.Action, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	action, err := client.VDS.HardReboot(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return client.Actions.Wait(ctx, action.ID, hexcore.WaitOptions{})
}

// #endregion restart

// #region history
func PrintActions(parent context.Context, client *hexcore.Client) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	actions, err := client.Actions.List(ctx)
	if err != nil {
		return err
	}
	for _, action := range actions {
		fmt.Printf("%s: %s on %s %s: %s\n",
			action.ID, action.Type, action.ResourceType, action.ResourceID, action.Status)
	}
	return nil
}

// #endregion history

// #region wait
func WaitForAction(parent context.Context, client *hexcore.Client, actionID string) (*hexcore.Action, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	return client.Actions.Wait(ctx, actionID, hexcore.WaitOptions{
		PollInterval: 3 * time.Second,
	})
}

// #endregion wait
