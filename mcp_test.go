package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An agent's session: handshake, tool list, a render that writes its file,
// and a bad option that comes back as a tool error, not a protocol one.
func TestMCP(t *testing.T) {
	out := filepath.Join(t.TempDir(), "a.gif")
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"render","arguments":{"args":["-w=30","-h=20","-gens=5","-at=3:palette=candy","-o=` + out + `"]}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"render","arguments":{"args":["-rule=nope"]}}}`,
	}, "\n")
	var buf bytes.Buffer
	if err := mcp(strings.NewReader(in), &buf); err != nil {
		t.Fatal(err)
	}
	type reply struct {
		Result struct {
			ProtocolVersion string
			Tools           []struct{ Name string }
			IsError         bool
			Content         []struct{ Text, MimeType, Data string }
		}
	}
	var replies []reply
	for dec := json.NewDecoder(&buf); dec.More(); {
		var r reply
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		replies = append(replies, r)
	}
	if len(replies) != 4 {
		t.Fatalf("%d replies, want 4 (none for the notification):\n%s", len(replies), buf.String())
	}
	if r := replies[0].Result; r.ProtocolVersion != "2025-06-18" {
		t.Errorf("initialize: protocol %q", r.ProtocolVersion)
	}
	if r := replies[1].Result; len(r.Tools) != 1 || r.Tools[0].Name != "render" {
		t.Errorf("tools/list: %+v", r.Tools)
	}
	r := replies[2].Result
	if r.IsError || len(r.Content) != 2 || !strings.HasPrefix(r.Content[0].Text, "wrote "+out+"\n") ||
		!strings.Contains(r.Content[0].Text, "% live") || r.Content[1].MimeType != "image/png" {
		t.Fatalf("render: %+v", r)
	}
	// The contact sheet: 3×2 tiles of the 30×20 grid, 4 pixels apart.
	data, _ := base64.StdEncoding.DecodeString(r.Content[1].Data)
	if sheet, err := png.Decode(bytes.NewReader(data)); err != nil || sheet.Bounds() != image.Rect(0, 0, 3*30+2*4, 2*20+4) {
		t.Errorf("sheet: %v", err)
	}
	if data, err := os.ReadFile(out); err != nil || !bytes.HasPrefix(data, []byte("GIF")) {
		t.Errorf("render wrote no GIF: %v", err)
	}
	if r := replies[3].Result; !r.IsError || !strings.Contains(r.Content[0].Text, "nope") {
		t.Errorf("bad rule: %+v", r)
	}
}
