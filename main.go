package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"

	hoverfly "github.com/SpectoLabs/hoverfly/core/handlers/v2"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type RequestResponsePair hoverfly.RequestResponsePairViewV1

type Configuration struct {
	Paths []string
	Host  *string
	clientcredentials.Config
}

func setup() (Configuration, error) {
	var conf Configuration
	err := envconfig.Process("HFLY_OAUTH2", &conf)
	return conf, err
}

func decode(r io.Reader) (*RequestResponsePair, error) {
	var reqRespPair RequestResponsePair
	err := json.NewDecoder(r).Decode(&reqRespPair)
	if err != nil {
		return nil, fmt.Errorf("failed to decode from stdin: %w", err)
	}
	return &reqRespPair, nil
}

func encode(w io.Writer, reqRespPair *RequestResponsePair) {
	json.NewEncoder(w).Encode(reqRespPair)
}

func handle(conf Configuration, ts oauth2.TokenSource, reqRespPair *RequestResponsePair) error {
	if reqRespPair.Request.Destination != nil && conf.Host != nil && *reqRespPair.Request.Destination != *conf.Host {
		log.Println("Host does not match configured host, skipping...")
		return nil
	}
	if reqRespPair.Request.Path != nil && len(conf.Paths) != 0 && !slices.Contains(conf.Paths, *reqRespPair.Request.Path) {
		log.Println("Path does not match configured paths, skipping...")
		return nil
	}
	if val, ok := reqRespPair.Request.Headers["Authentication"]; ok && len(val) > 0 {
		log.Println("Authentication header already present, skipping...")
		return nil
	}

	token, err := ts.Token()
	if err != nil {
		return fmt.Errorf("failed to fetch oauth2 token: %v", err)
	}
	reqRespPair.Request.Headers["Authentication"] = []string{token.Type() + " " + token.AccessToken}
	log.Println("Successfully set Authentication Header")
	return nil
}

func main() {
	log.SetOutput(os.Stderr)
	conf, err := setup()
	if err != nil {
		fmt.Printf("setup failed: %v\n", err)
		os.Exit(1)
	}

	reqRespPair, err := decode(os.Stdin)
	if err != nil {
		fmt.Printf("decode failed: %v\n", err)
		os.Exit(1)
	}
	defer encode(os.Stdout, reqRespPair)

	ts := conf.TokenSource(context.Background())

	err = handle(conf, ts, reqRespPair)
	if err != nil {
		fmt.Printf("handle failed: %v\n", err)
		os.Exit(1)
	}
}
