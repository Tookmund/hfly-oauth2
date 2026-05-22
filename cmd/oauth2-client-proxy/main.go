package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2/clientcredentials"
)

type Configuration struct {
	Paths []string
	Host  string
	clientcredentials.Config
}

func setup() (Configuration, error) {
	var conf Configuration
	err := envconfig.Process("OAUTH2_CLIENT_PROXY", &conf)
	return conf, err
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	conf, err := setup()
	if err != nil {
		fmt.Printf("setup failed: %v\n", err)
		os.Exit(1)
	}

	ts := conf.TokenSource(context.Background())

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		token, err := ts.Token()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to fetch oauth2 token: %v", err), http.StatusServiceUnavailable)
			return
		}

		if !slices.Contains(conf.Paths, req.URL.Path) {
			http.Error(w, fmt.Sprintf("permitted paths list does not contain: %s", req.URL.Path), http.StatusBadRequest)
			return
		}

		// Switch requested host to configured host
		req.URL.Host = conf.Host

		req.Header.Add("Authentication", token.Type()+" "+token.AccessToken)

		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
