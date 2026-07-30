package provider

import (
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
// CRUD lifecycle without needing a live LINE channel access token. It's torn
// down automatically at the end of the test.
func testAccPreCheckWithFakeServer(t *testing.T) *fakeLineServer {
	t.Helper()
	srv := newFakeLineServer()
	t.Cleanup(srv.Close)
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
	t.Setenv("LINE_API_BASE_URL", srv.APIServer.URL)
	t.Setenv("LINE_DATA_API_BASE_URL", srv.DataServer.URL)
	return srv
}
