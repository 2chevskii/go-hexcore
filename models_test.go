package hexcore_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	hexcore "github.com/2chevskii/go-hexcore"
)

func TestResponseRepresentations(t *testing.T) {
	tests := []struct {
		name string
		body string
		want any
	}{
		{"account numeric fields", `{"id":9007199254740993,"status":1,"type":"2","balance":123.4500,"referalClickCount":"12","referalRegCount":3,"discount":1.25,"configData":null}`,
			&hexcore.Account{ID: "9007199254740993", Status: 1, Type: 2, Balance: "123.4500", ReferralClickCount: 12, ReferralRegistrationCount: 3, Discount: "1.25"}},
		{"action history aliases", `{"id":9,"resource_id":9007199254740993,"date_started":"start","date_completed":null,"message":null,"status":"in-progress"}`,
			&hexcore.Action{ID: "9", ResourceID: "9007199254740993", StartedAt: "start", Status: hexcore.ActionInProgress}},
		{"action canonical timestamps", `{"id":"9","resource_id":"2","started_at":"start","completed_at":"end","date_started":"ignored","message":"failure","status":"error"}`,
			&hexcore.Action{ID: "9", ResourceID: "2", StartedAt: "start", CompletedAt: ptr("end"), Message: ptr("failure"), Status: hexcore.ActionFailed}},
		{"sparse VDS", `{"id":7,"vcpu":2,"memory":"3072","disk":15,"price":1200.00,"ipv4":null,"unknown":"ignored"}`,
			&hexcore.VDS{ID: "7", VCPU: 2, Memory: 3072, Disk: 15, Price: ptr("1200.00")}},
		{"disk creation", `{"id":1,"size":5,"created":"date","vds_id":42}`,
			&hexcore.Disk{ID: "1", Size: 5, CreatedAt: "date", VDSID: ptr("42")}},
		{"disk camel attachment", `{"id":"1","size":"5","created_date":"date","status":{"status":"plugged","vdsId":42}}`,
			&hexcore.Disk{ID: "1", Size: 5, CreatedAt: "date", Status: ptr("plugged"), VDSID: ptr("42")}},
		{"disk snake attachment", `{"id":"1","status":{"status":"unplugged","vds_id":null}}`,
			&hexcore.Disk{ID: "1", Status: ptr("unplugged")}},
		{"disk schema aliases", `{"id":1,"status":"plugged","vdsId":"42","nex_bill_date":"date"}`,
			&hexcore.Disk{ID: "1", Status: ptr("plugged"), VDSID: ptr("42"), NextBillDate: "date"}},
		{"disk null attachment", `{"id":1,"status":null,"vds_id":null}`, &hexcore.Disk{ID: "1"}},
		{"disk top-level attachment alias", `{"id":1,"vdsId":42,"status":{"status":"plugged"}}`, &hexcore.Disk{ID: "1", VDSID: ptr("42"), Status: ptr("plugged")}},
		{"size schema aliases", `{"slug":"plan","vcpu":"2","ram":1024,"disk":"15","month_price":250.00,"hour_price":"0.34"}`,
			&hexcore.Size{Slug: "plan", VCPU: 2, Memory: 1024, Disk: 15, MonthPrice: "250.00", HourPrice: "0.34"}},
		{"image numeric version", `{"id":1,"name":"Debian","version":12,"slug":"debian_12","min_disk_size":"10"}`,
			&hexcore.Image{ID: "1", Name: "Debian", Version: "12", Slug: "debian_12", MinDiskSize: 10}},
		{"application quantities", `{"id":2,"version":"1","min_disk_size":10}`, &hexcore.Application{ID: "2", Version: "1", MinDiskSize: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reflect.New(reflect.TypeOf(tt.want).Elem()).Interface()
			if err := json.Unmarshal([]byte(tt.body), got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decoded = %#v; want %#v", got, tt.want)
			}
			// Normalized exported resources remain serializable and decodable.
			data, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			roundtrip := reflect.New(reflect.TypeOf(tt.want).Elem()).Interface()
			if err := json.Unmarshal(data, roundtrip); err != nil || !reflect.DeepEqual(got, roundtrip) {
				t.Errorf("roundtrip = %#v, %v", roundtrip, err)
			}
		})
	}
}

func TestInvalidResponseScalars(t *testing.T) {
	for _, body := range []string{
		`null`, `[]`, `true`, `{}`, `{"id":null}`,
		`{"id":true}`, `{"id":{}}`, `{"id":"1","vcpu":""}`,
		`{"id":"1","vcpu":"1.5"}`, `{"id":"1","vcpu":"9223372036854775808"}`,
		`{"id":"1","vcpu":9223372036854775808}`, `{"id":"1","vcpu":true}`,
		`{"id":"1","price":false}`, `{"id":"1","price":[]}`,
	} {
		t.Run(body, func(t *testing.T) {
			vds := hexcore.VDS{ID: "original"}
			if err := json.Unmarshal([]byte(body), &vds); err == nil {
				t.Fatal("accepted invalid resource")
			}
			if vds.ID != "original" {
				t.Fatal("failed decoding modified the previous resource")
			}
		})
	}
}

func TestRequestEncoding(t *testing.T) {
	for _, tt := range []struct {
		name string
		req  any
		want string
	}{
		{"standard optional password", hexcore.CreateVDSRequest{Name: "host", Size: "plan", Image: "image", Region: "region"},
			`{"name":"host","size":"plan","image":"image","region":"region"}`},
		{"custom password", hexcore.CreateCustomVDSRequest{Name: "host", Size: hexcore.CustomSize{VCPU: 2, Memory: 3, Disk: 13}, Image: "image", Region: "region", Password: "example"},
			`{"name":"host","size":0,"custom_size":1,"custom_size_data":{"vcpu":2,"memory":3,"disk":13},"image":"image","region":"region","password":"example"}`},
		{"unattached disk", hexcore.CreateDiskRequest{Name: "disk", Size: 5, Region: "region"}, `{"name":"disk","size":5,"region":"region"}`},
		{"exact disk attachment ID", hexcore.CreateDiskRequest{Name: "disk", Size: 5, Region: "region", VDSID: ptr("9007199254740993")},
			`{"name":"disk","size":5,"region":"region","vds_id":9007199254740993}`},
		{"leading zeros", hexcore.CreateDiskRequest{VDSID: ptr("00012")}, `{"name":"","size":0,"region":"","vds_id":12}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			equalJSON(t, string(data), tt.want)
		})
	}
	for _, id := range []string{"", "0", "0000", "-1", "1.5", "1e3", " 1", "not-a-number"} {
		if _, err := json.Marshal(hexcore.CreateDiskRequest{VDSID: &id}); err == nil {
			t.Errorf("accepted invalid numeric attachment ID %q", id)
		} else if strings.Contains(err.Error(), id) && id == "not-a-number" {
			t.Error("encoding error included request data")
		}
	}
}
