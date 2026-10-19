package hexcore_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

func liveClient(t *testing.T) *hexcore.Client {
	t.Helper()
	if os.Getenv("HEXCORE_LIVE") != "1" {
		t.Skip("set HEXCORE_LIVE=1 to enable live API checks")
	}
	if os.Getenv("HEXCORE_TOKEN") == "" {
		t.Fatal("HEXCORE_TOKEN is required for live checks")
	}
	c, err := hexcore.NewClient(hexcore.Config{Token: os.Getenv("HEXCORE_TOKEN"), BaseURL: os.Getenv("HEXCORE_BASE_URL")})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestLiveReads does not change account resources. It also checks singular reads
// when the account contains a corresponding resource or action.
func TestLiveReads(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, tt := range []struct {
		name string
		call func() error
	}{
		{"Account.Get", func() error { _, err := c.Account.Get(ctx); return err }},
		{"Regions.List", func() error { _, err := c.Regions.List(ctx); return err }},
		{"Sizes.List", func() error { _, err := c.Sizes.List(ctx); return err }},
		{"Images.List", func() error { _, err := c.Images.List(ctx); return err }},
		{"Applications.List", func() error { _, err := c.Applications.List(ctx); return err }},
		{"Actions.ListAndGet", func() error {
			items, err := c.Actions.List(ctx)
			if err == nil && len(items) > 0 {
				_, err = c.Actions.Get(ctx, items[0].ID)
			}
			return err
		}},
		{"VDS.ListAndGet", func() error {
			items, err := c.VDS.List(ctx)
			if err == nil && len(items) > 0 {
				_, err = c.VDS.Get(ctx, items[0].ID)
			}
			return err
		}},
		{"Disks.ListAndGet", func() error {
			items, err := c.Disks.List(ctx)
			if err == nil && len(items) > 0 {
				_, err = c.Disks.Get(ctx, items[0].ID)
			}
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func liveRequired(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is required for live writes", name)
	}
	return v
}

func livePositive(t *testing.T, name string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(liveRequired(t, name), 10, 64)
	if err != nil || n <= 0 {
		t.Fatalf("%s must be a positive integer", name)
	}
	return n
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		t.Fatal("cannot generate test resource name")
	}
	return hex.EncodeToString(data[:])
}

func livePoll(ctx context.Context, check func() (bool, error)) error {
	for {
		ready, err := check()
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func liveVDSState(ctx context.Context, c *hexcore.Client, id, state string) error {
	return livePoll(ctx, func() (bool, error) {
		items, err := c.VDS.List(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.ID == id {
				return item.Status != nil && strings.EqualFold(*item.Status, state), nil
			}
		}
		return false, nil
	})
}

func liveDiskState(ctx context.Context, c *hexcore.Client, id string, check func(hexcore.Disk) bool) error {
	return livePoll(ctx, func() (bool, error) {
		items, err := c.Disks.List(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.ID == id {
				return check(item), nil
			}
		}
		return false, nil
	})
}

// TestLiveWrites is deliberately sequential. Only resources with this run's
// unique names/returned IDs are eligible for write operations and cleanup.
// Disks must be removed manually in the control panel after the run.
func TestLiveWrites(t *testing.T) {
	if os.Getenv("HEXCORE_LIVE_WRITES") != "1" {
		t.Skip("set HEXCORE_LIVE_WRITES=1 as well as HEXCORE_LIVE=1 to create disposable resources")
	}
	c := liveClient(t)
	region := liveRequired(t, "HEXCORE_REGION")
	size := liveRequired(t, "HEXCORE_SIZE")
	image := liveRequired(t, "HEXCORE_IMAGE")
	custom := hexcore.CustomSize{VCPU: livePositive(t, "HEXCORE_CUSTOM_VCPU"), Memory: livePositive(t, "HEXCORE_CUSTOM_MEMORY"), Disk: livePositive(t, "HEXCORE_CUSTOM_DISK")}
	diskSize := livePositive(t, "HEXCORE_DISK_SIZE")
	if diskSize == int64(1<<63-1) || custom.Memory > int64(1<<63-1)/1024 {
		t.Fatal("test quantities are too large")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Validate the selected catalog entries before creating billable resources.
	regions, err := c.Regions.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	validRegion := false
	for _, r := range regions {
		validRegion = validRegion || (r.Slug == region && r.Available)
	}
	sizes, err := c.Sizes.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var selectedSize *hexcore.Size
	for i := range sizes {
		if sizes[i].Slug == size {
			selectedSize = &sizes[i]
		}
	}
	images, err := c.Images.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var selectedImage *hexcore.Image
	for i := range images {
		if images[i].Slug == image {
			selectedImage = &images[i]
		}
	}
	if !validRegion || selectedSize == nil || selectedImage == nil {
		t.Fatal("select an available region and existing size/image slugs from the catalogs")
	}
	if selectedSize.Disk < selectedImage.MinDiskSize || custom.Disk < selectedImage.MinDiskSize {
		t.Fatal("test VDS disk sizes are below the selected image's minimum")
	}

	prefix := "go-hexcore-test-" + randomSuffix(t)
	standardName, customName, diskName := prefix+"-standard", prefix+"-custom", prefix+"-disk"
	ownedVDSNames := map[string]bool{standardName: true, customName: true}
	ownedVDSIDs := make(map[string]bool)
	ownedDiskIDs := make(map[string]bool)
	// Register cleanup before the first write. Name reconciliation also finds
	// resources accepted by the server when a response was lost or undecodable.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cleanupCancel()
		disks, err := c.Disks.List(cleanupCtx)
		if err != nil {
			t.Errorf("disk cleanup discovery failed; inspect resource name %s in the control panel: %v", diskName, err)
		}
		for _, disk := range disks {
			if disk.Name != diskName {
				continue
			}
			ownedDiskIDs[disk.ID] = true
			if disk.Status != nil && *disk.Status != "unplugged" {
				action, err := c.Disks.Unplug(cleanupCtx, disk.ID)
				if err == nil {
					_, err = c.Actions.Wait(cleanupCtx, action.ID, hexcore.WaitOptions{})
				}
				if err != nil {
					t.Errorf("cannot unplug test disk %s during cleanup: %v", disk.ID, err)
				}
			}
		}
		for id := range ownedDiskIDs {
			t.Logf("MANUAL CLEANUP: remove test disk ID=%s name=%s in the HexCore control panel", id, diskName)
		}
		servers, err := c.VDS.List(cleanupCtx)
		if err != nil {
			t.Errorf("VDS cleanup discovery failed; inspect names %s and %s: %v", standardName, customName, err)
		}
		for _, server := range servers {
			if ownedVDSNames[server.Name] {
				ownedVDSIDs[server.ID] = true
			}
		}
		for id := range ownedVDSIDs {
			if err := c.VDS.Destroy(cleanupCtx, id); err != nil {
				t.Errorf("cannot destroy test VDS %s; remove it manually: %v", id, err)
			}
		}
	})

	password := "Aa1!" + randomSuffix(t)
	standard, err := c.VDS.Create(ctx, hexcore.CreateVDSRequest{Name: standardName, Size: size, Image: image, Region: region, Password: password})
	if err != nil {
		t.Fatal(err)
	}
	ownedVDSIDs[standard.ID] = true
	t.Logf("created test VDS ID=%s name=%s", standard.ID, standardName)
	if err := liveVDSState(ctx, c, standard.ID, "Running"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.VDS.Get(ctx, standard.ID); err != nil {
		t.Fatal(err)
	}
	customVDS, err := c.VDS.CreateCustom(ctx, hexcore.CreateCustomVDSRequest{Name: customName, Size: custom, Image: image, Region: region, Password: password})
	if err != nil {
		t.Fatal(err)
	}
	ownedVDSIDs[customVDS.ID] = true
	t.Logf("created custom test VDS ID=%s name=%s", customVDS.ID, customName)
	if customVDS.VCPU != custom.VCPU || customVDS.Disk != custom.Disk || customVDS.Memory != custom.Memory*1024 {
		t.Fatal("custom quantities differ from the documentation's expected mapping; verify memory request units")
	}
	if err := liveVDSState(ctx, c, customVDS.ID, "Running"); err != nil {
		t.Fatal(err)
	}

	waitAction := func(action *hexcore.Action, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Actions.Wait(ctx, action.ID, hexcore.WaitOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, step := range []struct {
		name, state string
		call        func(context.Context, string) (*hexcore.Action, error)
	}{
		{"reboot", "Running", c.VDS.Reboot},
		{"hard reboot", "Running", c.VDS.HardReboot},
		{"shutdown", "halted", c.VDS.Shutdown},
		{"power on", "Running", c.VDS.PowerOn},
		{"hard shutdown", "halted", c.VDS.HardShutdown},
		{"power on after hard shutdown", "Running", c.VDS.PowerOn},
	} {
		t.Log("checking " + step.name)
		waitAction(step.call(ctx, standard.ID))
		if err := liveVDSState(ctx, c, standard.ID, step.state); err != nil {
			t.Fatal(err)
		}
	}
	waitAction(c.VDS.ResetPassword(ctx, standard.ID, "Aa1!"+randomSuffix(t)))
	if err := liveVDSState(ctx, c, standard.ID, "Running"); err != nil {
		t.Fatal(err)
	}
	disk, err := c.Disks.Create(ctx, hexcore.CreateDiskRequest{Name: diskName, Size: diskSize, Region: region, VDSID: &standard.ID})
	if err != nil {
		t.Fatal(err)
	}
	ownedDiskIDs[disk.ID] = true
	t.Logf("created test disk ID=%s name=%s; manual removal will be required", disk.ID, diskName)
	attached := func(d hexcore.Disk) bool {
		return d.VDSID != nil && *d.VDSID == standard.ID && d.Status != nil && (*d.Status == "plugged" || *d.Status == "not_attached")
	}
	if err := liveDiskState(ctx, c, disk.ID, attached); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Disks.Get(ctx, disk.ID); err != nil {
		t.Fatal(err)
	}
	unplugged := func(d hexcore.Disk) bool { return d.Status != nil && *d.Status == "unplugged" }
	waitAction(c.Disks.Unplug(ctx, disk.ID))
	if err := liveDiskState(ctx, c, disk.ID, unplugged); err != nil {
		t.Fatal(err)
	}
	waitAction(c.Disks.Resize(ctx, disk.ID, diskSize+1))
	if err := liveDiskState(ctx, c, disk.ID, func(d hexcore.Disk) bool { return d.Size == diskSize+1 && unplugged(d) }); err != nil {
		t.Fatal(err)
	}
	waitAction(c.Disks.Plug(ctx, disk.ID))
	if err := liveDiskState(ctx, c, disk.ID, attached); err != nil {
		t.Fatal(err)
	}
	waitAction(c.Disks.Unplug(ctx, disk.ID))
	if err := liveDiskState(ctx, c, disk.ID, unplugged); err != nil {
		t.Fatal(err)
	}

	// Check deletion and the actual missing-resource response without assuming
	// that an arbitrary 403 means absence. This ID was created by this test.
	if err := c.VDS.Destroy(ctx, customVDS.ID); err != nil {
		t.Fatal(err)
	}
	delete(ownedVDSIDs, customVDS.ID)
	if err := livePoll(ctx, func() (bool, error) {
		items, err := c.VDS.List(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.ID == customVDS.ID {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err = c.VDS.Get(ctx, customVDS.ID)
	var apiErr *hexcore.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an API error for the known deleted test VDS, got %v", err)
	}
	t.Logf("known deleted VDS returned HTTP %d; 403 remains ambiguous for arbitrary resources", apiErr.StatusCode)
}
