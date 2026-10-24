package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/services/genieacs"
)

const unsignedFlagPath = "InternetGatewayDevice.LANDevice.1.LANEthernetInterfaceConfig.1"

func newUnsignedFlagServer(
	t *testing.T,
	onTask func(*testing.T, map[string]any),
) *httptest.Server {
	t.Helper()
	t.Setenv("TARGET_ALLOWLIST", "127.0.0.1")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/devices" {
			if got := r.URL.Query().Get("projection"); got != unsignedFlagPath {
				t.Errorf("projection = %q, want %q", got, unsignedFlagPath)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"_id":"router-1","InternetGatewayDevice":{"LANDevice":{"1":{"LANEthernetInterfaceConfig":{"1":{"X_HW_L3Enable":{"_value":0,"_type":"xsd:unsignedInt"}}}}}}}]`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/devices/router-1/tasks" {
			var task map[string]any
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Errorf("decode task: %v", err)
			} else {
				onTask(t, task)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	}))
}

func TestSetParametersConvertsBooleanForUnsignedField(t *testing.T) {
	var wantValue float64
	server := newUnsignedFlagServer(t, func(t *testing.T, task map[string]any) {
		t.Helper()
		parameters, ok := task["parameterValues"].([]any)
		if !ok || len(parameters) != 1 {
			t.Fatalf("parameterValues = %#v, want one parameter", task["parameterValues"])
		}
		parameter, ok := parameters[0].([]any)
		if !ok || len(parameter) != 3 {
			t.Fatalf("parameter = %#v, want path, value, and type", parameters[0])
		}
		if parameter[0] != unsignedFlagPath+".X_HW_L3Enable" ||
			parameter[1] != wantValue ||
			parameter[2] != "xsd:unsignedInt" {
			t.Errorf("parameter = %#v, want value %v as xsd:unsignedInt", parameter, wantValue)
		}
	})
	defer server.Close()

	for raw, want := range map[string]float64{"true": 1, "TRUE": 1, "1": 1, "false": 0, "0": 0} {
		t.Run(raw, func(t *testing.T) {
			wantValue = want
			applied, err := genieacs.SetParameters(
				context.Background(),
				&models.GenieACS{Host: server.URL},
				"router-1",
				"lan/ports",
				unsignedFlagPath,
				map[string]string{"l3_enable": raw},
			)
			if err != nil {
				t.Fatalf("SetParameters() error = %v", err)
			}
			if !applied {
				t.Fatal("SetParameters() applied = false, want true")
			}
		})
	}
}

func TestSetParametersRejectsInvalidBooleanForUnsignedField(t *testing.T) {
	server := newUnsignedFlagServer(t, func(t *testing.T, _ map[string]any) {
		t.Error("unexpected task for invalid boolean")
	})
	defer server.Close()

	_, err := genieacs.SetParameters(
		context.Background(),
		&models.GenieACS{Host: server.URL},
		"router-1",
		"lan/ports",
		unsignedFlagPath,
		map[string]string{"l3_enable": "maybe"},
	)
	if !errors.Is(err, genieacs.ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter, got %v", err)
	}
}

func TestSetWANVLANUsesNokiaLinkConfig(t *testing.T) {
	t.Setenv("TARGET_ALLOWLIST", "127.0.0.1")
	const (
		wanPath        = "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.2.WANPPPConnection.1"
		connDevicePath = "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.2"
		vlanPath       = connDevicePath + ".X_CT-COM_WANGponLinkConfig.VLANIDMark"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/devices" {
			projection := r.URL.Query().Get("projection")
			if projection != wanPath && projection != connDevicePath {
				t.Errorf("unexpected projection: %s", projection)
			}
			w.Header().Set("Content-Type", "application/json")
			connectionDevice := map[string]any{
				"WANPPPConnection": map[string]any{
					"1": map[string]any{
						"ConnectionType": map[string]any{"_value": "IP_Routed"},
					},
				},
				"X_CT-COM_WANGponLinkConfig": map[string]any{
					"VLANIDMark": map[string]any{
						"_value": 300,
						"_type":  "xsd:unsignedInt",
					},
				},
			}
			device := map[string]any{
				"_id": "router-1",
				"InternetGatewayDevice": map[string]any{
					"WANDevice": map[string]any{
						"1": map[string]any{
							"WANConnectionDevice": map[string]any{
								"2": connectionDevice,
							},
						},
					},
				},
			}
			if err := json.NewEncoder(w).Encode([]map[string]any{device}); err != nil {
				t.Errorf("encode device: %v", err)
			}
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/devices/router-1/tasks" {
			var task struct {
				Name            string  `json:"name"`
				ObjectName      string  `json:"objectName"`
				ParameterValues [][]any `json:"parameterValues"`
			}
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Errorf("decode task: %v", err)
			} else if task.Name == "setParameterValues" {
				if len(task.ParameterValues) != 1 || len(task.ParameterValues[0]) != 3 {
					t.Errorf("parameterValues = %#v, want one three-part parameter", task.ParameterValues)
				} else if task.ParameterValues[0][0] != vlanPath ||
					task.ParameterValues[0][1] != float64(3010) ||
					task.ParameterValues[0][2] != "xsd:unsignedInt" {
					t.Errorf("parameter = %#v, want %s = 3010 (xsd:unsignedInt)", task.ParameterValues[0], vlanPath)
				}
			} else if task.Name != "refreshObject" || task.ObjectName != "InternetGatewayDevice.WANDevice.1.WANConnectionDevice" {
				t.Errorf("unexpected task: %+v", task)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	}))
	defer server.Close()

	applied, err := genieacs.SetParameters(
		context.Background(),
		&models.GenieACS{Host: server.URL},
		"router-1",
		"wan",
		wanPath,
		map[string]string{"vlan": "3010"},
	)
	if err != nil {
		t.Fatalf("SetParameters() error = %v", err)
	}
	if !applied {
		t.Fatal("SetParameters() applied = false, want true")
	}
}
