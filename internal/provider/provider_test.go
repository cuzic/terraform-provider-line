package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider into
// terraform-plugin-testing's acceptance test runner, which drives a real
// `terraform` binary against it via gRPC.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"line": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheckWithFakeServer starts an in-memory fake LINE API and points
// the provider at it via the LINE_API_BASE_URL / LINE_DATA_API_BASE_URL test
// hooks (see provider.go), so acceptance tests exercise the real Terraform
// CRUD lifecycle without needing a live LINE channel access token.
//
// If LINE_CHANNEL_ACCESS_TOKEN is already set in the environment when the
// test process starts (e.g. by a maintainer running acceptance tests
// locally, or by .github/workflows/acceptance-live.yml), this is a no-op:
// the very same TestAcc* test bodies then run against the real LINE API
// instead, since they're written against the real wire protocol. That's
// deliberate — one set of tests, two backends.
func testAccPreCheckWithFakeServer(t *testing.T) {
	t.Helper()
	if os.Getenv("LINE_CHANNEL_ACCESS_TOKEN") != "" {
		t.Log("LINE_CHANNEL_ACCESS_TOKEN is set: running against the real LINE API, not a fake server")
		return
	}
	srv := newFakeLineServer()
	t.Cleanup(srv.Close)
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
	t.Setenv("LINE_API_BASE_URL", srv.APIServer.URL)
	t.Setenv("LINE_DATA_API_BASE_URL", srv.DataServer.URL)
}
