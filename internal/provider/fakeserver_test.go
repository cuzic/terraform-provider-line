package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// fakeLineServer is an in-memory stand-in for the real LINE Messaging API
// and LIFF server API, used by acceptance tests (TestAcc*) so the full
// Terraform CRUD lifecycle can be exercised without real LINE credentials.
// It intentionally implements only the subset of behavior this provider
// depends on.
type fakeLineServer struct {
	mu sync.Mutex

	webhookEndpoint string
	webhookActive   bool

	liffApps   map[string]map[string]any
	liffNextID int

	richMenus     map[string]map[string]any
	richMenuNext  int
	richMenuImage map[string][]byte

	APIServer  *httptest.Server
	DataServer *httptest.Server
}

func newFakeLineServer() *fakeLineServer {
	f := &fakeLineServer{
		liffApps:      map[string]map[string]any{},
		richMenus:     map[string]map[string]any{},
		richMenuImage: map[string][]byte{},
	}
	f.APIServer = httptest.NewServer(http.HandlerFunc(f.handleAPI))
	f.DataServer = httptest.NewServer(http.HandlerFunc(f.handleData))
	return f
}

func (f *fakeLineServer) Close() {
	f.APIServer.Close()
	f.DataServer.Close()
}

func (f *fakeLineServer) handleAPI(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v2/bot/channel/webhook/endpoint":
		writeJSON(w, http.StatusOK, map[string]any{"endpoint": f.webhookEndpoint, "active": f.webhookActive})

	case r.Method == http.MethodPut && r.URL.Path == "/v2/bot/channel/webhook/endpoint":
		var body struct {
			Endpoint string `json:"endpoint"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.webhookEndpoint = body.Endpoint
		f.webhookActive = body.Endpoint != ""
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodPost && r.URL.Path == "/liff/v1/apps":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.liffNextID++
		id := fmt.Sprintf("liff-%d", f.liffNextID)
		body["liffId"] = id
		f.liffApps[id] = body
		writeJSON(w, http.StatusOK, map[string]any{"liffId": id})

	case r.Method == http.MethodGet && r.URL.Path == "/liff/v1/apps":
		apps := make([]map[string]any, 0, len(f.liffApps))
		for _, a := range f.liffApps {
			apps = append(apps, a)
		}
		writeJSON(w, http.StatusOK, map[string]any{"apps": apps})

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/liff/v1/apps/"):
		id := strings.TrimPrefix(r.URL.Path, "/liff/v1/apps/")
		if _, ok := f.liffApps[id]; !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		body["liffId"] = id
		f.liffApps[id] = body
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/liff/v1/apps/"):
		id := strings.TrimPrefix(r.URL.Path, "/liff/v1/apps/")
		if _, ok := f.liffApps[id]; !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		delete(f.liffApps, id)
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodPost && r.URL.Path == "/v2/bot/richmenu":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.richMenuNext++
		id := fmt.Sprintf("richmenu-%d", f.richMenuNext)
		body["richMenuId"] = id
		f.richMenus[id] = body
		writeJSON(w, http.StatusOK, map[string]any{"richMenuId": id})

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/bot/richmenu/"):
		id := strings.TrimPrefix(r.URL.Path, "/v2/bot/richmenu/")
		rm, ok := f.richMenus[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, rm)

	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v2/bot/richmenu/"):
		id := strings.TrimPrefix(r.URL.Path, "/v2/bot/richmenu/")
		if _, ok := f.richMenus[id]; !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		delete(f.richMenus, id)
		delete(f.richMenuImage, id)
		w.WriteHeader(http.StatusOK)

	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "no route: " + r.Method + " " + r.URL.Path})
	}
}

func (f *fakeLineServer) handleData(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/bot/richmenu/") && strings.HasSuffix(r.URL.Path, "/content") {
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/bot/richmenu/"), "/content")
		if _, ok := f.richMenus[id]; !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.richMenuImage[id] = body
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"message": "no route: " + r.Method + " " + r.URL.Path})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
