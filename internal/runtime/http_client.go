package runtime

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.ibm.com/solsa/kar.git/internal/config"
	"golang.org/x/net/http2"
)

var (
	url    = fmt.Sprintf("http://127.0.0.1:%d", config.AppPort)
	client http.Client
)

func fakeDialTLS(network, addr string, cfg *tls.Config) (net.Conn, error) {
	return net.Dial(network, addr)
}

func init() {
	var transport http.RoundTripper
	if config.H2C {
		transport = &http2.Transport{AllowHTTP: true, DialTLS: fakeDialTLS}
	} else {
		t1 := http.DefaultTransport.(*http.Transport).Clone()
		t1.MaxIdleConnsPerHost = 256
		transport = t1
	}
	client = http.Client{Transport: transport} // TODO adjust timeout
}

// ReadAll converts a request or response body to a string
func ReadAll(r io.ReadCloser) string {
	buf, _ := ioutil.ReadAll(r) // TODO size limit?
	r.Close()
	return string(buf)
}

// discard discards a response body
func discard(r io.ReadCloser) {
	io.Copy(ioutil.Discard, r)
	r.Close()
}

// invoke sends an HTTP request to the service and returns the response
func invoke(ctx context.Context, method string, msg map[string]string) (*http.Response, error) {
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
