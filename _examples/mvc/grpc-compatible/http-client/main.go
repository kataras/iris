package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	pb "github.com/kataras/iris/v12/_examples/mvc/grpc-compatible/helloworld"
)

func main() {
	b, err := ioutil.ReadFile("../server.crt")
	if err != nil {
		log.Fatal(err)
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		log.Fatal("credentials: failed to append certificates")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: cp,
		},
	}

	client := http.Client{Transport: transport}
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(pb.HelloRequest{Name: "world"})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Post("https://localhost/helloworld.Greeter/SayHello", "application/json", buf)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var reply pb.HelloReply
	err = json.NewDecoder(resp.Body).Decode(&reply)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Greeting: %s", reply.GetMessage())
}
