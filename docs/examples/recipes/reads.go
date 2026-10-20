package recipes

import (
	"context"
	"fmt"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region deadline
func ReadAccount(parent context.Context, client *hexcore.Client) (*hexcore.Account, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	return client.Account.Get(ctx)
}

// #endregion deadline

// #region balance
func PrintBalance(parent context.Context, client *hexcore.Client) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	account, err := client.Account.Get(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Balance: %s %s\n", account.Balance, account.Currency)
	return nil
}

// #endregion balance

// #region vds
func FindServer(parent context.Context, client *hexcore.Client, id string) (*hexcore.VDS, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	return client.VDS.Get(ctx, id)
}

// #endregion vds

// #region disks
func PrintDisks(parent context.Context, client *hexcore.Client) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	disks, err := client.Disks.List(ctx)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		fmt.Printf("%s: %s (%d GB)\n", disk.ID, disk.Name, disk.Size)
	}
	return nil
}

// #endregion disks

// #region optional
func PrintAddress(parent context.Context, client *hexcore.Client, id string) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	server, err := client.VDS.Get(ctx, id)
	if err != nil {
		return err
	}
	if server.IPv4 != nil && *server.IPv4 != "" {
		fmt.Println(*server.IPv4)
	} else {
		fmt.Println("No IPv4 address in this response")
	}
	return nil
}

// #endregion optional
