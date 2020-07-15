package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kataras/iris/v12/context"
)

const baseURL = "http://localhost:8080"

// Available options:
// - "gzip",
// - "deflate",
// - "br" (for brotli),
// - "snappy" and
// - "s2"
const encoding = context.BROTLI

var client = http.DefaultClient

func main() {
	fmt.Printf("Running client example on: %s\n", baseURL)

	getExample()
	postExample()
}

func getExample() {
	endpoint := baseURL + "/"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		panic(err)
	}
	// Required to receive server's compressed data.
	req.Header.Set("Accept-Encoding", encoding)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// decompress server's compressed reply.
	cr, err := context.NewCompressReader(resp.Body, encoding)
	if err != nil {
		panic(err)
	}
	defer cr.Close()

	body, err := ioutil.ReadAll(cr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Received from server: %s", string(body))
}

type payload struct {
	Username string `json:"username"`
}

func postExample() {
	buf := new(bytes.Buffer)

	// Compress client's data.
	cw, err := context.NewCompressWriter(buf, encoding, -1)
	if err != nil {
		panic(err)
	}

	json.NewEncoder(cw).Encode(payload{Username: "Edward"})

	// `Close` or `Flush` required before `NewRequest` call.
	cw.Close()

	endpoint := baseURL + "/"

	req, err := http.NewRequest(http.MethodPost, endpoint, buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Required to send gzip compressed data to the server.
	req.Header.Set("Content-Encoding", encoding)
	// Required to receive server's compressed data.
	req.Header.Set("Accept-Encoding", encoding)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Decompress server's compressed reply.
	cr, err := context.NewCompressReader(resp.Body, encoding)
	if err != nil {
		panic(err)
	}
	defer cr.Close()

	body, err := ioutil.ReadAll(cr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Server replied with: %s", string(body))
}
