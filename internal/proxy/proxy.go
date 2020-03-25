package proxy

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.ibm.com/solsa/kar.git/internal/config"
)

var (
	url    = fmt.Sprintf("http://127.0.0.1:%d", config.ServicePort)
	client http.Client
)

func init() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 256
	client = http.Client{Transport: transport} // TODO adjust timeout
}

// Read converts a request or response body to a string
func Read(r io.ReadCloser) string {
	buf, _ := ioutil.ReadAll(r) // TODO size limit?
	r.Close()
	return string(buf)
}

// Flush discards a response body
func Flush(r io.ReadCloser) {
	io.Copy(ioutil.Discard, r)
	r.Close()
}

// Do sends an HTTP request to the service and returns the response
func Do(ctx context.Context, method string, msg map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url+msg["path"], strings.NewReader(msg["payload"]))
	if err != nil {
		return nil, err
	}
	if msg["content-type"] != "" {
		req.Header.Set("Content-Type", msg["content-type"])
	}
	if msg["accept"] != "" {
		req.Header.Set("Accept", msg["accept"])
	}
	var res *http.Response
	err = backoff.Retry(func() error {
		res, err = client.Do(req)
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	return res, err
}
