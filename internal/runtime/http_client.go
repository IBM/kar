package runtime

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
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

// ReadAll converts the body of a request to a string
func ReadAll(r *http.Request) string {
	buf, _ := ioutil.ReadAll(r.Body) // TODO size limit?
	return string(buf)
}

// invoke sends an HTTP request to the service and returns the response
func invoke(ctx context.Context, method string, msg map[string]string) (*Reply, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	req, err := http.NewRequestWithContext(ctx, method, url+msg["path"], strings.NewReader(msg["payload"]))
	if err != nil {
		return nil, err
	}
	if msg["content-type"] != "" {
		req.Header.Set("Content-Type", msg["content-type"])
	}
	if msg["accept"] != "" {
		req.Header.Set("Accept", msg["accept"])
	}
	var reply *Reply
	err = backoff.Retry(func() error {
		var res *http.Response
		res, err = client.Do(req) // TODO adjust timeout
		if err != nil {
			logger.Warning("failed to invoke %s: %v", msg["path"], err)
			if err == ctx.Err() {
				return backoff.Permanent(err)
			}
			return err
		}
		buf, _ := ioutil.ReadAll(res.Body) // TODO size limit?
		res.Body.Close()
		if length, err := strconv.Atoi(res.Header.Get("Content-Length")); err == nil && len(buf) != length {
			logger.Warning("failed to invoke %s: unexpected content length (%d != %d)", msg["path"], length, len(buf))
			return errors.New("unexpected content length")
		}
		reply = &Reply{StatusCode: res.StatusCode, Payload: string(buf), ContentType: res.Header.Get("Content-Type")}
		return nil
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	if ctx.Err() != nil {
		CloseIdleConnections() // don't keep connection alive once ctx is cancelled
	}
	return reply, err
}

// CloseIdleConnections closes idle connections
func CloseIdleConnections() {
	client.CloseIdleConnections()
}
