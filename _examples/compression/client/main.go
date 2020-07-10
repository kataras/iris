package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var client = http.DefaultClient

const baseURL = "http://localhost:8080"

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
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// decompress server's compressed reply.
	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	body, err := ioutil.ReadAll(r)
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
	w := gzip.NewWriter(buf)

	b, err := json.Marshal(payload{Username: "Edward"})
	if err != nil {
		panic(err)
	}
	w.Write(b)
	w.Close()

	endpoint := baseURL + "/"

	req, err := http.NewRequest(http.MethodPost, endpoint, buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Required to send gzip compressed data to the server.
	req.Header.Set("Content-Encoding", "gzip")
	// Required to receive server's compressed data.
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Decompress server's compressed reply.
	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	body, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Server replied with: %s", string(body))
}
