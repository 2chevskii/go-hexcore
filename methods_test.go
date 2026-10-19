package hexcore_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	hexcore "github.com/2chevskii/go-hexcore"
)

func ptr[T any](v T) *T { return &v }

func testClient(t *testing.T, handler http.HandlerFunc) *hexcore.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := hexcore.NewClient(hexcore.Config{Token: "test-token", BaseURL: server.URL + "/api/", UserAgent: "test-provider"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func equalJSON(t *testing.T, got, want string) {
	t.Helper()
	decode := func(text string) any {
		dec := json.NewDecoder(strings.NewReader(text))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("invalid test JSON: %v", err)
		}
		return v
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Errorf("JSON = %s; want %s", got, want)
	}
}

// Responses are checked-in examples from the official Swagger document.
// Expected values are hand-authored so tests exercise actual decoding behavior.
func TestDocumentedOperations(t *testing.T) {
	data, err := os.ReadFile("testdata/responses.json")
	if err != nil {
		t.Fatal(err)
	}
	var responses map[string]json.RawMessage
	if err := json.Unmarshal(data, &responses); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	history := hexcore.Action{ID: "9", ResourceID: "5681", ResourceType: "vds", Type: "reboot", Status: hexcore.ActionCompleted,
		StartedAt: "2019-02-19T13:08:53+03:00", CompletedAt: ptr("2019-02-19T13:09:19+03:00")}
	vdsCreated := hexcore.VDS{ID: "24536", Name: "example.com", VCPU: 1, Memory: 1024, Disk: 10,
		CreatedAt: ptr("2018-06-01 12:32:10"), NextBillDate: "2018-07-01"}
	vdsCustom := vdsCreated
	vdsCustom.Memory, vdsCustom.Disk = 3072, 13
	diskCreated := hexcore.Disk{ID: "24536", Name: "new-disk-name", Size: 5, CreatedAt: "2024-07-12 13:45:12", NextBillDate: "2024-08-12", VDSID: ptr("1782")}
	tests := []struct {
		operation string
		method    string
		path      string
		body      string
		status    int
		invoke    func(*hexcore.Client) (any, error)
		want      any
	}{
		{"get_account_information", "GET", "/account", "", 200, func(c *hexcore.Client) (any, error) { return c.Account.Get(ctx) },
			&hexcore.Account{ID: "2", Email: "support@hexcore.ru", Status: 1, Type: 2, Balance: "250.00", Discount: "0", CreatedAt: "2015-07-06 22:49:09",
				ConfigData: &hexcore.CompanyConfig{Address: "192012, САНКТ-ПЕТЕРБУРГ ГОРОД, ОБУХОВСКОЙ ОБОРОНЫ ПРОСПЕКТ, д. 2", INN: "78111111", OGRN: "1000000000", BankRS: "4070281234234234", BankBIK: "044030858", BankName: "ПЕТЕРБУРГСКИЙ ФИЛИАЛ АО ЮНИКРЕДИТ БАНКА", BankKS: "3010181231231231231"}}},
		{"get_Action_ById", "GET", "/actions/9", "", 200, func(c *hexcore.Client) (any, error) { return c.Actions.Get(ctx, "9") }, &history},
		{"get_all_Actions", "GET", "/actions_list", "", 200, func(c *hexcore.Client) (any, error) { return c.Actions.List(ctx) },
			[]hexcore.Action{history, {ID: "10", ResourceID: "5681", ResourceType: "vds", Type: "shutodwn", StartedAt: "2019-02-19T13:08:53+03:00", Status: hexcore.ActionInProgress}}},
		{"get_Disk_ById", "GET", "/disk/36", "", 200, func(c *hexcore.Client) (any, error) { return c.Disks.Get(ctx, "36") },
			&hexcore.Disk{ID: "36", Name: "vhdtest1", Region: ptr("RU-SPB-4"), Size: 1, Price: ptr("3.00"), CreatedAt: "2024-02-01 18:04:01", NextBillDate: "2024-03-01", Status: ptr("plugged"), VDSID: ptr("8877")}},
		{"create_new_Disk", "POST", "/disk", `{"name":"new-disk-name","size":5,"region":"ru-spb-10","vds_id":1782}`, 202,
			func(c *hexcore.Client) (any, error) {
				return c.Disks.Create(ctx, hexcore.CreateDiskRequest{Name: "new-disk-name", Size: 5, Region: "ru-spb-10", VDSID: ptr("1782")})
			}, &diskCreated},
		{"get_all_Disks", "GET", "/disk_list", "", 200, func(c *hexcore.Client) (any, error) { return c.Disks.List(ctx) },
			[]hexcore.Disk{
				{ID: "36", Name: "vhd1", Region: ptr("RU-SPB-4"), Size: 1, Price: ptr("2.00"), CreatedAt: "2023-09-01 18:04:01", NextBillDate: "2023-10-01", Status: ptr("plugged"), VDSID: ptr("136")},
				{ID: "37", Name: "vhd2", Region: ptr("RU-MSK-4"), Size: 10, Price: ptr("30.00"), CreatedAt: "2024-02-01 18:04:01", NextBillDate: "2024-03-01", Status: ptr("not_attached"), VDSID: ptr("378")},
				{ID: "38", Name: "vhd3", Region: ptr("RU-SPB-10"), Size: 100, Price: ptr("300.00"), CreatedAt: "2024-03-11 21:10:00", NextBillDate: "2024-04-11", Status: ptr("unplugged"), VDSID: ptr("1429")}}},
		{"plug_Disk", "POST", "/disk/5749/action", `{"type":"plug"}`, 200,
			func(c *hexcore.Client) (any, error) { return c.Disks.Plug(ctx, "5749") },
			&hexcore.Action{ID: "185160", Status: hexcore.ActionInProgress, Type: "plug", StartedAt: "2024-05-14T16:45:01Z", ResourceID: "5749", ResourceType: "disk"}},
		{"unplug_Disk", "POST", "/disk/4536/action", `{"type":"unplug"}`, 200,
			func(c *hexcore.Client) (any, error) { return c.Disks.Unplug(ctx, "4536") },
			&hexcore.Action{ID: "185160", Status: hexcore.ActionInProgress, Type: "plug", StartedAt: "2024-05-14T16:45:01Z", ResourceID: "4536", ResourceType: "disk"}},
		{"resize_Disk", "POST", "/disk/7852/action", `{"type":"resize","size":15}`, 200,
			func(c *hexcore.Client) (any, error) { return c.Disks.Resize(ctx, "7852", 15) },
			&hexcore.Action{ID: "185276", Status: hexcore.ActionInProgress, Type: "resize", StartedAt: "2024-05-14T16:45:01Z", ResourceID: "7852", ResourceType: "disk"}},
		{"create_new_Vds", "POST", "/vds", `{"name":"example.com","size":"1vcpu-1gb-10disk","image":"centos_7","region":"ru-spb-4","password":"example-password"}`, 202,
			func(c *hexcore.Client) (any, error) {
				return c.VDS.Create(ctx, hexcore.CreateVDSRequest{Name: "example.com", Size: "1vcpu-1gb-10disk", Image: "centos_7", Region: "ru-spb-4", Password: "example-password"})
			}, &vdsCreated},
		{"create_new_custom_Vds", "POST", "/vds", `{"name":"example.com","size":0,"custom_size":1,"custom_size_data":{"vcpu":1,"memory":3,"disk":13},"image":"centos_7","region":"ru-spb-4"}`, 202,
			func(c *hexcore.Client) (any, error) {
				return c.VDS.CreateCustom(ctx, hexcore.CreateCustomVDSRequest{Name: "example.com", Size: hexcore.CustomSize{VCPU: 1, Memory: 3, Disk: 13}, Image: "centos_7", Region: "ru-spb-4"})
			}, &vdsCustom},
		{"list_Vds", "GET", "/vds_list", "", 200, func(c *hexcore.Client) (any, error) { return c.VDS.List(ctx) },
			[]hexcore.VDS{
				{ID: "111", Name: "Terminal", VCPU: 2, Memory: 4096, Disk: 30, IPv4: ptr("1.1.1.1"), Status: ptr("Running"), Price: ptr("1900.00"), Region: ptr("EE-TLL-1"), NextBillDate: "2018-01-27"},
				{ID: "333", Name: "blog", VCPU: 2, Memory: 2048, Disk: 30, IPv4: ptr("2.2.2.2"), Status: ptr("Running"), Price: ptr("1000.00"), Region: ptr("EE-TLL-3"), NextBillDate: "2018-02-08"}}},
		{"get_Vds_ById", "GET", "/vds/254617", "", 200, func(c *hexcore.Client) (any, error) { return c.VDS.Get(ctx, "254617") },
			&hexcore.VDS{ID: "254617", Name: "Terminal", VCPU: 2, Memory: 4096, Disk: 30, IPv4: ptr("85.143.202.123"), Price: ptr("1900.00"), NextBillDate: "2018-01-27"}},
		{"destroy_Vds", "POST", "/vds/384762/action", `{"type":"destroy"}`, 204,
			func(c *hexcore.Client) (any, error) { return nil, c.VDS.Destroy(ctx, "384762") }, nil},
		{"list_regions", "GET", "/regions", "", 200, func(c *hexcore.Client) (any, error) { return c.Regions.List(ctx) },
			[]hexcore.Region{{Name: "ru-4", Slug: "ru-spb-4", Available: true}, {Name: "ru-5", Slug: "ru-msk-1", Available: true}}},
		{"list_sizes", "GET", "/sizes", "", 200, func(c *hexcore.Client) (any, error) { return c.Sizes.List(ctx) },
			[]hexcore.Size{{Slug: "1vcpu-1gb-15disk", VCPU: 1, Memory: 1024, Disk: 15, MonthPrice: "250", HourPrice: "0.34"}, {Slug: "2vcpu-2gb-20disk", VCPU: 2, Memory: 2048, Disk: 20, MonthPrice: "500", HourPrice: "0.68"}}},
		{"list_images", "GET", "/images", "", 200, func(c *hexcore.Client) (any, error) { return c.Images.List(ctx) },
			[]hexcore.Image{{ID: "1", Name: "Debian", Version: "8", Slug: "debian_8", MinDiskSize: 10}, {ID: "2", Name: "Ubuntu", Version: "16.04", Slug: "debian_16.04", MinDiskSize: 10}}},
		{"list_applications", "GET", "/applications", "", 200, func(c *hexcore.Client) (any, error) { return c.Applications.List(ctx) },
			[]hexcore.Application{{ID: "23", Name: "VestaCP", Version: "1", Slug: "vestacp", MinDiskSize: 10}, {ID: "24", Name: "LAMP", Version: "1", Slug: "lamp", MinDiskSize: 10}}},
	}
	for _, action := range []struct {
		op, id, kind, actionID string
		invoke                 func(*hexcore.Client, context.Context, string) (*hexcore.Action, error)
	}{
		{"reboot_Vds", "3946265", "reboot", "184057", func(c *hexcore.Client, ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.Reboot(ctx, id)
		}},
		{"hard_reboot_Vds", "1356265", "hard_reboot", "184159", func(c *hexcore.Client, ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.HardReboot(ctx, id)
		}},
		{"shutdown_Vds", "495847", "shutdown", "184957", func(c *hexcore.Client, ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.Shutdown(ctx, id)
		}},
		{"hard_shutdown_Vds", "145634", "hard_shutdown", "170239", func(c *hexcore.Client, ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.HardShutdown(ctx, id)
		}},
		{"power_on_Vds", "384751", "power_on", "146539", func(c *hexcore.Client, ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.PowerOn(ctx, id)
		}},
		{"reset_password_Vds", "1356266", "reset_password", "1356266", func(c *hexcore.Client, ctx context.Context, id string) (*hexcore.Action, error) {
			return c.VDS.ResetPassword(ctx, id, "example-password")
		}},
	} {
		body := `{"type":"` + action.kind + `"}`
		if action.kind == "reset_password" {
			body = `{"type":"reset_password","password":"example-password"}`
		}
		tests = append(tests, struct {
			operation, method, path, body string
			status                        int
			invoke                        func(*hexcore.Client) (any, error)
			want                          any
		}{action.op, "POST", "/vds/" + action.id + "/action", body, 200,
			func(c *hexcore.Client) (any, error) { return action.invoke(c, ctx, action.id) },
			&hexcore.Action{ID: action.actionID, Status: hexcore.ActionInProgress, Type: action.kind, StartedAt: "2018-11-14T16:31:00Z", ResourceID: "3164450", ResourceType: "vds"}})
	}
	if len(tests) != len(responses) || len(tests) != 24 {
		t.Fatalf("operations: tests=%d fixtures=%d; want 24", len(tests), len(responses))
	}
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			fixture, exists := responses[tt.operation]
			if !exists {
				t.Fatal("missing documented fixture")
			}
			var calls atomic.Int32
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tt.method || r.URL.Path != "/api"+tt.path || r.URL.RawQuery != "" {
					t.Errorf("request = %s %s; want %s /api%s", r.Method, r.URL, tt.method, tt.path)
				}
				for name, want := range map[string]string{"Authorization": "Bearer test-token", "Accept": "application/json", "Content-Type": "application/json", "User-Agent": "test-provider"} {
					if r.Header.Get(name) != want {
						t.Errorf("incorrect %s header", name)
					}
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if tt.body == "" {
					if len(body) != 0 {
						t.Error("GET request has a body")
					}
				} else {
					equalJSON(t, string(body), tt.body)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				if tt.status != 204 {
					_, _ = w.Write(fixture)
				}
			})
			got, err := tt.invoke(c)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("result = %#v; want %#v", got, tt.want)
			}
			if calls.Load() != 1 {
				t.Errorf("made %d requests; want exactly one", calls.Load())
			}
		})
	}
}
