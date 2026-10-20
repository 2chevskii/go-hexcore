package recipes

import (
	"context"
	"fmt"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region create-disk
func CreateDisk(parent context.Context, client *hexcore.Client, name, region string, vdsID *string) (*hexcore.Disk, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	return client.Disks.Create(ctx, hexcore.CreateDiskRequest{
		Name: name, Region: region, Size: 10, VDSID: vdsID,
	})
}

// #endregion create-disk

// #region grow-disk
func GrowDisk(parent context.Context, client *hexcore.Client, id string, newSizeGB int64) (*hexcore.Action, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	disk, err := client.Disks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if disk.Status == nil || *disk.Status != "unplugged" {
		return nil, fmt.Errorf("disk %s is not confirmed unplugged", id)
	}
	if newSizeGB <= disk.Size {
		return nil, fmt.Errorf("new size must exceed %d GB", disk.Size)
	}
	action, err := client.Disks.Resize(ctx, id, newSizeGB)
	if err != nil {
		return nil, err
	}
	return client.Actions.Wait(ctx, action.ID, hexcore.WaitOptions{})
}

// #endregion grow-disk
