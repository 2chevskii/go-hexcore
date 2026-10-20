package recipes

import (
	"context"
	"fmt"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region create
func CreateServer(parent context.Context, client *hexcore.Client, name, size, image, region string) (*hexcore.VDS, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	return client.VDS.Create(ctx, hexcore.CreateVDSRequest{
		Name: name, Size: size, Image: image, Region: region,
	})
}

// #endregion create

// #region custom
func CreateCustomServer(parent context.Context, client *hexcore.Client, name, image, region string) (*hexcore.VDS, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	return client.VDS.CreateCustom(ctx, hexcore.CreateCustomVDSRequest{
		Name: name, Image: image, Region: region,
		Size: hexcore.CustomSize{VCPU: 2, Memory: 2, Disk: 20},
	})
}

// #endregion custom

// #region ready
func WaitForRunning(parent context.Context, client *hexcore.Client, id string) (*hexcore.VDS, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		servers, err := client.VDS.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, server := range servers {
			if server.ID == id && server.Status != nil && *server.Status == "running" &&
				server.IPv4 != nil && *server.IPv4 != "" {
				return &server, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// #endregion ready

// #region destroy
func DeleteHaltedServer(parent context.Context, client *hexcore.Client, id string) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	servers, err := client.VDS.List(ctx)
	if err != nil {
		return err
	}
	halted := false
	for _, server := range servers {
		if server.ID == id && server.Status != nil && *server.Status == "halted" {
			halted = true
		}
	}
	if !halted {
		return fmt.Errorf("server %s is not confirmed halted", id)
	}
	if err := client.VDS.Destroy(ctx, id); err != nil {
		return err
	}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		servers, err := client.VDS.List(ctx)
		if err != nil {
			return err // An error, including HTTP 403, does not confirm deletion.
		}
		found := false
		for _, server := range servers {
			found = found || server.ID == id
		}
		if !found {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// #endregion destroy
