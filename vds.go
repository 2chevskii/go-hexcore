package hexcore

import (
	"context"
	"encoding/json"
	"net/http"
)

// VDSService manages virtual dedicated servers and their lifecycle actions.
type VDSService struct{ client *Client }

// CreateVDSRequest selects a standard size, image, and region by slug.
// Password is omitted when empty.
type CreateVDSRequest struct {
	Name     string `json:"name"`
	Size     string `json:"size"`
	Image    string `json:"image"`
	Region   string `json:"region"`
	Password string `json:"password,omitempty"`
}

// CustomSize contains quantities for custom VDS creation. Memory and Disk are
// in GB. The API reports VDS.Memory in MB (1024 per requested GB). Quantities
// are sent unchanged.
type CustomSize struct {
	VCPU   int64 `json:"vcpu"`
	Memory int64 `json:"memory"`
	Disk   int64 `json:"disk"`
}

// CreateCustomVDSRequest selects custom quantities instead of a size slug.
// The required size=0 and custom_size=1 flags are supplied automatically.
type CreateCustomVDSRequest struct {
	Name     string     `json:"name"`
	Size     CustomSize `json:"custom_size_data"`
	Image    string     `json:"image"`
	Region   string     `json:"region"`
	Password string     `json:"password,omitempty"`
}

// MarshalJSON adds the flags required by the custom VDS API.
func (r CreateCustomVDSRequest) MarshalJSON() ([]byte, error) {
	type plain CreateCustomVDSRequest
	return json.Marshal(struct {
		plain
		SizeFlag   int `json:"size"`
		CustomFlag int `json:"custom_size"`
	}{plain: plain(r), SizeFlag: 0, CustomFlag: 1})
}

// Get retrieves one VDS. Some fields present in List are omitted by this route.
func (s *VDSService) Get(ctx context.Context, id string) (*VDS, error) {
	path, err := resourcePath("vds", id, "")
	if err != nil {
		return nil, err
	}
	return requestOne[VDS](ctx, s.client, http.MethodGet, path, nil, http.StatusOK, "vds")
}

// List retrieves all VDS belonging to the current account.
func (s *VDSService) List(ctx context.Context) ([]VDS, error) {
	return requestList[VDS](ctx, s.client, http.MethodGet, "/vds_list", nil, http.StatusOK, "vds")
}

// Create requests a standard VDS and returns its accepted resource immediately.
// The API does not return an action ID or guarantee resource readiness here.
func (s *VDSService) Create(ctx context.Context, req CreateVDSRequest) (*VDS, error) {
	return requestOne[VDS](ctx, s.client, http.MethodPost, "/vds", req, http.StatusAccepted, "vds")
}

// CreateCustom requests a custom VDS and returns its accepted resource immediately.
func (s *VDSService) CreateCustom(ctx context.Context, req CreateCustomVDSRequest) (*VDS, error) {
	return requestOne[VDS](ctx, s.client, http.MethodPost, "/vds", req, http.StatusAccepted, "vds")
}

type actionRequest struct {
	Type     string  `json:"type"`
	Password *string `json:"password,omitempty"`
}

func (s *VDSService) action(ctx context.Context, id, kind string, password *string) (*Action, error) {
	path, err := resourcePath("vds", id, "/action")
	if err != nil {
		return nil, err
	}
	return requestOne[Action](ctx, s.client, http.MethodPost, path, actionRequest{Type: kind, Password: password}, http.StatusOK, "action", http.StatusAccepted)
}

// Reboot requests a graceful reboot and returns the accepted action.
func (s *VDSService) Reboot(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "reboot", nil)
}

// HardReboot requests a hardware-style reset and returns the accepted action.
func (s *VDSService) HardReboot(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "hard_reboot", nil)
}

// Shutdown requests a graceful shutdown and returns the accepted action.
func (s *VDSService) Shutdown(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "shutdown", nil)
}

// HardShutdown requests a power cut and returns the accepted action.
func (s *VDSService) HardShutdown(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "hard_shutdown", nil)
}

// PowerOn requests power-on and returns the accepted action.
func (s *VDSService) PowerOn(ctx context.Context, id string) (*Action, error) {
	return s.action(ctx, id, "power_on", nil)
}

// ResetPassword requests a root password change, which also reboots the VDS.
func (s *VDSService) ResetPassword(ctx context.Context, id, password string) (*Action, error) {
	return s.action(ctx, id, "reset_password", &password)
}

// Destroy requests destruction of an already halted VDS. Both 204 and 202
// acknowledgements are accepted. The caller owns shutdown and deletion polling.
func (s *VDSService) Destroy(ctx context.Context, id string) error {
	path, err := resourcePath("vds", id, "/action")
	if err != nil {
		return err
	}
	return s.client.do(ctx, http.MethodPost, path, actionRequest{Type: "destroy"}, http.StatusNoContent, nil, http.StatusAccepted)
}
