package hexcore

import (
	"context"
	"encoding/json"
	"net/http"
)

// DisksService manages additional storage. Disk deletion is not documented.
type DisksService struct{ client *Client }

// CreateDiskRequest selects a disk size in GB and deployment region by slug.
// VDSID optionally requests attachment during creation. If supplied, it must be a
// positive decimal ID and is encoded as a JSON number without precision loss.
type CreateDiskRequest struct {
	Name   string  `json:"name"`
	Size   int64   `json:"size"`
	Region string  `json:"region"`
	VDSID  *string `json:"vds_id,omitempty"`
}

// MarshalJSON encodes the optional VDS ID in the API's numeric request format.
func (r CreateDiskRequest) MarshalJSON() ([]byte, error) {
	var id *json.Number
	if r.VDSID != nil {
		n, err := numericID(*r.VDSID)
		if err != nil {
			return nil, err
		}
		id = &n
	}
	return json.Marshal(struct {
		Name   string       `json:"name"`
		Size   int64        `json:"size"`
		Region string       `json:"region"`
		VDSID  *json.Number `json:"vds_id,omitempty"`
	}{Name: r.Name, Size: r.Size, Region: r.Region, VDSID: id})
}

// Get retrieves one disk using the /disk/{id} route in the documentation.
func (s *DisksService) Get(ctx context.Context, id string) (*Disk, error) {
	path, err := resourcePath("disk", id, "")
	if err != nil {
		return nil, err
	}
	return requestOne[Disk](ctx, s.client, http.MethodGet, path, nil, http.StatusOK, "disk")
}

// List retrieves all disks belonging to the current account.
func (s *DisksService) List(ctx context.Context) ([]Disk, error) {
	return requestList[Disk](ctx, s.client, http.MethodGet, "/disk_list", nil, http.StatusOK, "disk")
}

// Create requests a disk and returns its accepted resource immediately.
func (s *DisksService) Create(ctx context.Context, req CreateDiskRequest) (*Disk, error) {
	return requestOne[Disk](ctx, s.client, http.MethodPost, "/disk", req, http.StatusAccepted, "disk")
}

func (s *DisksService) action(ctx context.Context, id, kind string, size *int64) (*Action, error) {
	path, err := resourcePath("disk", id, "/action")
	if err != nil {
		return nil, err
	}
	req := struct {
		Type string `json:"type"`
		Size *int64 `json:"size,omitempty"`
	}{Type: kind, Size: size}
	return requestOne[Action](ctx, s.client, http.MethodPost, path, req, http.StatusOK, "action", http.StatusAccepted)
}

// Plug submits the documented plug action. The documentation provides no
// target-VDS parameter; target selection still requires live verification.
func (s *DisksService) Plug(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "plug", nil)
}

// Unplug requests detachment and returns the accepted action.
func (s *DisksService) Unplug(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "unplug", nil)
}

// Resize requests a new size in GB. The disk must already be unplugged and the
// size must increase. The client does not orchestrate these prerequisites.
func (s *DisksService) Resize(ctx context.Context, id string, size int64) (*Action, error) {
	return s.action(ctx, id, "resize", &size)
}
