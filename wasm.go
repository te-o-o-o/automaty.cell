//go:build js && wasm

package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/url"
	"syscall/js"
	"time"
)

// The WebAssembly build, run by worker.js for the static page (-site). Each
// function takes the query string the page would send to -serve and returns
// {body, type} like a response would, or {error}.
func main() {
	js.Global().Set("automaty", js.ValueOf(map[string]any{
		"render":   js.FuncOf(func(_ js.Value, args []js.Value) any { return reply(render(args[0].String(), args[1])) }),
		"surprise": js.FuncOf(func(_ js.Value, args []js.Value) any { return reply(surpriseJSON(args[0].String())) }),
		"mutate":   js.FuncOf(func(_ js.Value, args []js.Value) any { return reply(mutateJSON(args[0].String())) }),
	}))
	select {} // keep the functions alive
}

func reply(body []byte, typ string, err error) any {
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	out := js.Global().Get("Uint8Array").New(len(body))
	js.CopyBytesToJS(out, body)
	return map[string]any{"body": out, "type": typ}
}

func query(s string) (*options, error) {
	q, err := url.ParseQuery(s)
	if err != nil {
		return nil, err
	}
	return parseQuery(q)
}

// render makes the image; mask is the uploaded image's bytes, or null.
func render(s string, mask js.Value) ([]byte, string, error) {
	o, err := query(s)
	if err != nil {
		return nil, "", err
	}
	if mask.Truthy() {
		o.maskData = make([]byte, mask.Length())
		js.CopyBytesToGo(o.maskData, mask)
	}
	if err := checkLimits(o); err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	if err := o.generate(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), o.contentType(), nil
}

func surpriseJSON(s string) ([]byte, string, error) {
	o, err := query(s)
	if err == nil {
		err = checkLimits(o)
	}
	if err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(surpriseReply(surprise(rand.New(rand.NewSource(time.Now().UnixNano())), *o)))
	return b, "application/json", err
}

func mutateJSON(s string) ([]byte, string, error) {
	o, err := query(s)
	if err == nil {
		err = checkMutate(o)
	}
	if err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(map[string]string{"rule": mutateLively(rand.New(rand.NewSource(time.Now().UnixNano())), *o)})
	return b, "application/json", err
}
