package hexcore_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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
			if liveEmptyCollection(err, "/actions_list", "No actions found.") {
				t.Log("Actions.List verified empty-collection 403; Actions.Get requires an action")
				return nil
			}
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
			if liveEmptyCollection(err, "/disk_list", "Can not find disk for your account.") {
				t.Log("Disks.List verified empty-collection 403; Disks.Get requires a disk")
				return nil
			}
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

// Only the integration harness recognizes these observed collection responses.
// The SDK retains the original 403 and never infers singular resource absence.
func liveEmptyCollection(err error, path, message string) bool {
	var apiErr *hexcore.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 403 && apiErr.Method == "GET" && apiErr.Path == path && apiErr.Message == message
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

func liveResourcePrefix(t *testing.T) string {
	t.Helper()
	// The live API truncates VDS names to 45 characters. Keep suffixes intact
	// so name-based reconciliation can distinguish resources from this run.
	return "go-hexcore-test-" + randomSuffix(t)[:16]
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
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

func liveVDSDeleted(ctx context.Context, c *hexcore.Client, id string) error {
	return livePoll(ctx, func() (bool, error) {
		items, err := c.VDS.List(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.ID == id {
				return false, nil
			}
		}
		return true, nil
	})
}

// Destruction requires a halted VDS. These fixtures are disposable, so cleanup
// uses hard shutdown and polls resource state independently of guest commands
// and action history. It never applies to preexisting account resources.
func liveDestroyVDS(ctx context.Context, c *hexcore.Client, id string) error {
	items, err := c.VDS.List(ctx)
	if err != nil {
		return err
	}
	var found *hexcore.VDS
	for i := range items {
		if items[i].ID == id {
			found = &items[i]
			break
		}
	}
	if found == nil {
		return nil
	}
	var shutdownErr error
	if found.Status == nil || !strings.EqualFold(*found.Status, "halted") {
		_, shutdownErr = c.VDS.HardShutdown(ctx, id)
		var apiErr *hexcore.APIError
		if errors.As(shutdownErr, &apiErr) && apiErr.StatusCode >= 400 {
			return fmt.Errorf("halt test VDS: %w", shutdownErr)
		}
		// A decoding or transport failure may follow an accepted shutdown.
		// Observe its state without replaying the write, then still clean up.
		if err := liveVDSState(ctx, c, id, "halted"); err != nil {
			return errors.Join(shutdownErr, err)
		}
	}
	if err := c.VDS.Destroy(ctx, id); err != nil {
		return errors.Join(shutdownErr, err)
	}
	return errors.Join(shutdownErr, liveVDSDeleted(ctx, c, id))
}

// Running does not imply that a freshly booted guest accepts commands yet.
// The harness may repeat an explicitly rejected command, never an accepted or
// ambiguously failed write. SDK methods themselves continue to make one attempt.
func liveVDSAction(ctx context.Context, id string, call func(context.Context, string) (*hexcore.Action, error)) (*hexcore.Action, error) {
	readyCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var action *hexcore.Action
	var lastRefusal error
	err := livePoll(readyCtx, func() (bool, error) {
		var err error
		action, err = call(readyCtx, id)
		var apiErr *hexcore.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 403 && apiErr.Method == "POST" && apiErr.Path == "/vds/"+id+"/action" && apiErr.Message == "Action not allowed by VDS." {
			lastRefusal = err
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return nil, errors.Join(lastRefusal, err)
	}
	return action, nil
}

func liveWaitAction(t *testing.T, ctx context.Context, c *hexcore.Client) func(*hexcore.Action, error) {
	return func(action *hexcore.Action, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("accepted action ID=%s type=%s", action.ID, action.Type)
		actionCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		if _, err := c.Actions.Wait(actionCtx, action.ID, hexcore.WaitOptions{}); err != nil {
			t.Fatal(err)
		}
		t.Logf("completed action ID=%s", action.ID)
	}
}

func liveDiskState(ctx context.Context, c *hexcore.Client, id string, check func(hexcore.Disk) bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
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

	prefix := liveResourcePrefix(t)
	standardName, customName, diskName := prefix+"-standard", prefix+"-custom", prefix+"-disk"
	ownedVDSNames := map[string]bool{standardName: true, customName: true}
	ownedVDSIDs := make(map[string]bool)
	ownedDiskIDs := make(map[string]bool)
	// Register cleanup before the first write. Name reconciliation also finds
	// resources accepted by the server when a response was lost or undecodable.
	t.Cleanup(func() {
		diskCtx, diskCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer diskCancel()
		disks, err := c.Disks.List(diskCtx)
		if err != nil && !liveEmptyCollection(err, "/disk_list", "Can not find disk for your account.") {
			t.Errorf("disk cleanup discovery failed; inspect resource name %s in the control panel: %v", diskName, err)
		}
		for _, disk := range disks {
			if disk.Name != diskName && !ownedDiskIDs[disk.ID] {
				continue
			}
			ownedDiskIDs[disk.ID] = true
			if disk.Status != nil && *disk.Status != "unplugged" {
				action, err := c.Disks.Unplug(diskCtx, disk.ID)
				if err == nil {
					_, err = c.Actions.Wait(diskCtx, action.ID, hexcore.WaitOptions{})
				}
				if err != nil {
					t.Errorf("cannot unplug test disk %s during cleanup: %v", disk.ID, err)
				}
			}
		}
		for id := range ownedDiskIDs {
			t.Logf("MANUAL CLEANUP: remove test disk ID=%s name=%s in the HexCore control panel", id, diskName)
		}
		diskCancel()
		// Give server cleanup its own deadline even when a disk action stalls.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cleanupCancel()
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
			if err := liveDestroyVDS(cleanupCtx, c, id); err != nil {
				t.Errorf("test VDS %s cleanup returned an error; verify removal: %v", id, err)
			}
		}
	})

	password := "Aa1!" + randomSuffix(t)
	// Verify custom and standard servers sequentially. This avoids reserving
	// the monthly price of two servers at once on a minimally funded account.
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
	if _, err := c.VDS.Get(ctx, customVDS.ID); err != nil {
		t.Fatal(err)
	}
	if err := liveDestroyVDS(ctx, c, customVDS.ID); err != nil {
		t.Fatal(err)
	}
	delete(ownedVDSIDs, customVDS.ID)
	_, err = c.VDS.Get(ctx, customVDS.ID)
	var apiErr *hexcore.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an API error for the known deleted test VDS, got %v", err)
	}
	t.Logf("known deleted VDS returned HTTP %d; 403 remains ambiguous for arbitrary resources", apiErr.StatusCode)

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

	waitAction := liveWaitAction(t, ctx, c)
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

	liveVDSActions(t, ctx, c, standard.ID)
}

// VDS actions can be checked independently when disk provisioning is blocked.
func liveVDSActions(t *testing.T, ctx context.Context, c *hexcore.Client, id string) {
	t.Helper()
	for _, step := range []struct {
		name, state string
		call        func(context.Context, string) (*hexcore.Action, error)
	}{
		{"hard shutdown", "halted", c.VDS.HardShutdown},
		{"power on after hard shutdown", "Running", c.VDS.PowerOn},
		{"hard reboot", "Running", c.VDS.HardReboot},
		{"shutdown", "halted", c.VDS.Shutdown},
		{"power on", "Running", c.VDS.PowerOn},
	} {
		if !t.Run(step.name, func(t *testing.T) {
			liveWaitAction(t, ctx, c)(liveVDSAction(ctx, id, step.call))
			if err := liveVDSState(ctx, c, id, step.state); err != nil {
				t.Fatal(err)
			}
		}) {
			return
		}
	}
	newPassword := "Aa1!" + randomSuffix(t)
	if !t.Run("reset password", func(t *testing.T) {
		liveWaitAction(t, ctx, c)(liveVDSAction(ctx, id, func(ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.ResetPassword(ctx, id, newPassword)
		}))
		if err := liveVDSState(ctx, c, id, "Running"); err != nil {
			t.Fatal(err)
		}
	}) {
		return
	}

	t.Run("reboot", func(t *testing.T) {
		liveWaitAction(t, ctx, c)(liveVDSAction(ctx, id, c.VDS.Reboot))
	})
}
