package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func runVMCreate(t *testing.T, handler http.HandlerFunc, args ...string) (string, string, error) {
	t.Helper()
	isolateCreds(t)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	var out, progress bytes.Buffer
	rt := &app.Runtime{
		Config:  &config.Config{},
		Streams: term.Streams{Out: &out, ErrOut: &progress},
	}
	root := NewRoot(rt)
	root.SetOut(&out)
	root.SetErr(&progress)
	root.SetArgs(append([]string{"--api-url", server.URL, "--token", "test-token", "--ws", "acme", "vm", "create"}, args...))
	err := root.Execute()
	return out.String(), progress.String(), err
}

func TestVMCreate_queuesVM(t *testing.T) {
	out, _, err := runVMCreate(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST /v1/vms", r.Method+" "+r.URL.Path)
		assert.Equal(t, "acme", r.Header.Get("X-Workspace-Slug"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.Header.Get("X-Idempotency"))
		var body json.RawMessage
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.JSONEq(t, `{"name":"Web One","plan_slug":"micro","image_slug":"ubuntu-24-04","vpc_slug":"default","zone_slug":"lagos-1","ssh_key_uids":["key-1","key-2"]}`, string(body))
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"success":true,"data":{"operation_id":"op-1","vm_uid":"vm-1","name":"Web One","status":"queued","vm_status":"provisioning"}}`)
	}, " Web One ", "--plan", "micro", "--image", "ubuntu-24-04", "--zone", "lagos-1", "--ssh-key", "key-1", "--ssh-key", "key-2")
	require.NoError(t, err)
	assert.Contains(t, out, "Queued VM creation for Web One")
	assert.Contains(t, out, "vm-1")
	assert.Contains(t, out, "op-1")
}

func TestVMCreate_wait(t *testing.T) {
	for _, status := range []string{"succeeded", "failed"} {
		t.Run(status, func(t *testing.T) {
			out, progress, err := runVMCreate(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.URL.Path {
				case "POST /v1/vms":
					var body map[string]json.RawMessage
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.NotContains(t, body, "zone_slug")
					w.WriteHeader(http.StatusAccepted)
					_, _ = io.WriteString(w, `{"success":true,"data":{"operation_id":"op-1","vm_uid":"vm-1","name":"Web One","status":"queued","vm_status":"provisioning"}}`)
				case "GET /v1/operations/op-1":
					_, _ = fmt.Fprintf(w, `{"success":true,"data":{"status":%q}}`, status)
				case "GET /v1/vms":
					assert.Equal(t, "succeeded", status)
					_, _ = io.WriteString(w, `{"success":true,"data":[{"uid":"other-vm","slug":"vm-1","status":"running"},{"uid":"vm-1","slug":"web-one","display_status":"running"}]}`)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}, "Web One", "--plan", "micro", "--image", "ubuntu-24-04", "--ssh-key", "key-1", "--wait")
			assert.Contains(t, progress, "Waiting for Web One to start")
			if status == "failed" {
				require.ErrorContains(t, err, "create failed for Web One")
				assert.NotContains(t, out, "is running")
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, "Web One is running")
		})
	}
}

func TestVMCreate_usageErrors(t *testing.T) {
	cases := map[string][]string{
		"missing VM name":             {},
		"VM name is empty":            {" "},
		"--plan is required":          {"web", "--image", "ubuntu-24-04", "--ssh-key", "key-1"},
		"--image is required":         {"web", "--plan", "micro", "--ssh-key", "key-1"},
		"--ssh-key is required":       {"web", "--plan", "micro", "--image", "ubuntu-24-04"},
		"--ssh-key must not be empty": {"web", "--plan", "micro", "--image", "ubuntu-24-04", "--ssh-key", "key-1", "--ssh-key", " "},
	}
	for want, args := range cases {
		t.Run(want, func(t *testing.T) {
			_, _, err := runVMCreate(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusBadRequest)
			}, args...)
			require.ErrorContains(t, err, want)
			assert.True(t, config.IsUsageError(err))
		})
	}
}

func TestVMCreate_reportsAPIError(t *testing.T) {
	out, _, err := runVMCreate(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = io.WriteString(w, `{"success":false,"message":"Your account is past due.","action":"past_due"}`)
	}, "web", "--plan", "micro", "--image", "ubuntu-24-04", "--ssh-key", "key-1")
	require.ErrorContains(t, err, "Your account is past due.")
	assert.Empty(t, strings.TrimSpace(out))
}
