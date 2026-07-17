package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"whisper/internal/chat"
)

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	directory := t.TempDir()
	for _, name := range []string{"index.html", "styles.css", "app.js"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := chat.NewStore(chat.StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(NewServer(Config{StaticDir: directory}, store, logger).Handler())
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return server, &http.Client{Jar: jar}
}

func jsonRequest(t *testing.T, client *http.Client, method, address string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, address, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestAuthenticationAndWebSocketMessage(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()

	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	parsed, _ := url.Parse(server.URL)
	headers := http.Header{}
	for _, cookie := range client.Jar.Cookies(parsed) {
		headers.Add("Cookie", cookie.String())
	}
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	connection, _, err := websocket.DefaultDialer.Dial(websocketURL, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	if err := connection.WriteJSON(map[string]string{
		"type": "message", "scope": "group", "text": "hello",
	}); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		var event map[string]any
		if err := connection.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event["type"] == "message" {
			if event["conversation"] != "group" {
				t.Fatalf("conversation = %#v", event["conversation"])
			}
			return
		}
	}
	t.Fatal("group message event not received")
}

func TestFriendColorAPI(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()

	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response = jsonRequest(t, client, http.MethodPatch, server.URL+"/api/friends/color", map[string]string{
		"name": "coco", "color": "rose",
	})
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("friend color status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if got := payload.FriendColors[chat.CocoName]; got != "rose" {
		t.Fatalf("friend color = %q, want rose", got)
	}
}
