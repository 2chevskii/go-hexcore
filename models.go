package hexcore

import (
	"bytes"
	"encoding/json"
)

// Account describes the authenticated account. Monetary values retain their
// decimal representation; CreatedAt retains the server's timezone-free date.
type Account struct {
	ID                        string         `json:"id"`
	Email                     string         `json:"email"`
	Status                    int64          `json:"status"`
	Type                      int64          `json:"type"`
	Balance                   string         `json:"balance"`
	Currency                  string         `json:"currency"`
	ReferralClickCount        int64          `json:"referalClickCount"`
	ReferralRegistrationCount int64          `json:"referalRegCount"`
	Discount                  string         `json:"discount"`
	CreatedAt                 string         `json:"create_date"`
	ConfigData                *CompanyConfig `json:"configData"`
}

// CompanyConfig contains billing details for a corporate account.
type CompanyConfig struct {
	Address  string `json:"address"`
	INN      string `json:"inn"`
	OGRN     string `json:"ogrn"`
	BankRS   string `json:"bankRs"`
	BankBIK  string `json:"bankBik"`
	BankName string `json:"bankName"`
	BankKS   string `json:"bankKs"`
}

// UnmarshalJSON accepts the API's quoted numeric quantities.
func (a *Account) UnmarshalJSON(data []byte) error {
	type plain Account
	var result plain
	err := decodeFields(data, &result,
		[]string{"status", "type", "referalClickCount", "referalRegCount"},
		[]string{"id", "balance", "discount"}, nil)
	if err = requireID(result.ID, err); err != nil {
		return err
	}
	*a = Account(result)
	return nil
}

// ActionStatus is the server's asynchronous action status.
type ActionStatus string

// Documented action statuses. Unknown values can still be decoded.
const (
	ActionInProgress ActionStatus = "in-progress"
	ActionCompleted  ActionStatus = "completed"
	ActionFailed     ActionStatus = "error"
)

// Action describes an asynchronous operation. CompletedAt and Message are nil
// when absent or null. Timestamp strings preserve the server's representation.
type Action struct {
	ID           string       `json:"id"`
	ResourceID   string       `json:"resource_id"`
	ResourceType string       `json:"resource_type"`
	Type         string       `json:"type"`
	Status       ActionStatus `json:"status"`
	StartedAt    string       `json:"started_at"`
	CompletedAt  *string      `json:"completed_at"`
	Message      *string      `json:"message"`
}

// UnmarshalJSON normalizes action timestamps from history and action responses.
func (a *Action) UnmarshalJSON(data []byte) error {
	type plain Action
	var result plain
	err := decodeFields(data, &result, nil, []string{"id", "resource_id"},
		map[string]string{"date_started": "started_at", "date_completed": "completed_at"})
	if err = requireID(result.ID, err); err != nil {
		return err
	}
	*a = Action(result)
	return nil
}

// VDS describes a virtual dedicated server. Memory is in MB and Disk is in GB.
// Optional fields are nil when a particular endpoint omits them; Get does not
// synthesize status or region by making additional requests.
type VDS struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	VCPU         int64   `json:"vcpu"`
	Memory       int64   `json:"memory"`
	Disk         int64   `json:"disk"`
	IPv4         *string `json:"ipv4"`
	Status       *string `json:"status"`
	Price        *string `json:"price"`
	Region       *string `json:"region"`
	CreatedAt    *string `json:"created"`
	NextBillDate string  `json:"next_bill_date"`
}

// UnmarshalJSON accepts quoted or numeric resource IDs and quantities.
func (v *VDS) UnmarshalJSON(data []byte) error {
	type plain VDS
	var result plain
	err := decodeFields(data, &result, []string{"vcpu", "memory", "disk"}, []string{"id", "price"}, nil)
	if err = requireID(result.ID, err); err != nil {
		return err
	}
	*v = VDS(result)
	return nil
}

// Disk describes additional storage. Size is in GB. Status and VDSID flatten
// the nested attachment object returned by reads; creation may omit Status.
type Disk struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Region       *string `json:"region"`
	Size         int64   `json:"size"`
	Price        *string `json:"price"`
	CreatedAt    string  `json:"created_date"`
	NextBillDate string  `json:"next_bill_date"`
	Status       *string `json:"status"`
	VDSID        *string `json:"vds_id"`
}

// UnmarshalJSON normalizes attachment IDs and creation date field names.
func (d *Disk) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	status := bytes.TrimSpace(fields["status"])
	if len(status) != 0 && status[0] == '{' {
		var attachment struct {
			Status *string `json:"status"`
			VDSID  *string `json:"vds_id"`
		}
		if err := decodeFields(status, &attachment, nil, []string{"vds_id"}, map[string]string{"vdsId": "vds_id"}); err != nil {
			return err
		}
		fields["status"], _ = json.Marshal(attachment.Status)
		if _, exists := fields["vds_id"]; !exists && attachment.VDSID != nil {
			fields["vds_id"], _ = json.Marshal(attachment.VDSID)
		}
		var err error
		data, err = json.Marshal(fields)
		if err != nil {
			return err
		}
	}
	type plain Disk
	var result plain
	err := decodeFields(data, &result, []string{"size"}, []string{"id", "price", "vds_id"},
		map[string]string{"created": "created_date", "vdsId": "vds_id", "nex_bill_date": "next_bill_date"})
	if err = requireID(result.ID, err); err != nil {
		return err
	}
	*d = Disk(result)
	return nil
}

// Region describes a deployment location and whether it accepts new VDS.
type Region struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Available bool   `json:"available"`
}

// Size describes a standard VDS plan. Memory is in MB and Disk is in GB.
// Prices retain their decimal representation in rubles.
type Size struct {
	Slug       string `json:"slug"`
	VCPU       int64  `json:"cpu"`
	Memory     int64  `json:"ram"`
	Disk       int64  `json:"disk"`
	MonthPrice string `json:"monthPrice"`
	HourPrice  string `json:"hourPrice"`
}

// UnmarshalJSON accepts the field names shown in both schemas and examples.
func (s *Size) UnmarshalJSON(data []byte) error {
	type plain Size
	var result plain
	if err := decodeFields(data, &result, []string{"cpu", "ram", "disk"}, []string{"monthPrice", "hourPrice"},
		map[string]string{"vcpu": "cpu", "month_price": "monthPrice", "hour_price": "hourPrice"}); err != nil {
		return err
	}
	*s = Size(result)
	return nil
}

// Image describes an OS image. MinDiskSize is in GB.
type Image struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Slug        string `json:"slug"`
	MinDiskSize int64  `json:"min_disk_size"`
}

// UnmarshalJSON accepts quoted or numeric IDs and minimum disk sizes.
func (i *Image) UnmarshalJSON(data []byte) error {
	type plain Image
	var result plain
	err := decodeFields(data, &result, []string{"min_disk_size"}, []string{"id", "version"}, nil)
	if err = requireID(result.ID, err); err != nil {
		return err
	}
	*i = Image(result)
	return nil
}

// Application describes an OS image with preinstalled software.
type Application Image

// UnmarshalJSON uses the same wire representation as an Image.
func (a *Application) UnmarshalJSON(data []byte) error {
	var result Image
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*a = Application(result)
	return nil
}
