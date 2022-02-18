//
// Copyright IBM Corporation 2020,2022
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package runtime

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/cenkalti/backoff/v4"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/http2"
)

var (
	url                       = fmt.Sprintf("http://127.0.0.1:%d", config.AppPort)
	client                    http.Client
	userCodeDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kar_user_code_invocation_durations_histogram_seconds",
		Help:    "KAR Actor/Service local invocation duration distributions.",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
	}, []string{"path"})
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
	client = http.Client{Transport: transport}
	if config.RequestRetryLimit >= 0 {
		client.Timeout = config.RequestRetryLimit
	}

	prometheus.MustRegister(userCodeDurationHistogram)
}

// ReadAll converts the body of a request to a string
func ReadAll(r *http.Request) string {
	buf, _ := ioutil.ReadAll(r.Body) // TODO size limit?
	return string(buf)
}

// invoke sends an HTTP request to the service and returns the response
func invoke(ctx context.Context, method string, msg map[string]string, metricLabel string) (*Reply, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	req, err := http.NewRequestWithContext(ctx, method, url+msg["path"], strings.NewReader(msg["payload"]))

	if err != nil {
		return nil, err
	}
	if msg["header"] != "" {
		var head map[string][]string
		err := json.Unmarshal([]byte(msg["header"]), &head)
		if err != nil {
			logger.Error("failed to properly unmarshal header: %v", err)
		}
		for field, vals := range head {
			for _, val := range vals {
				req.Header.Add(field, val)
			}
		}
	} else {
		if msg["content-type"] != "" {
			req.Header.Set("Content-Type", msg["content-type"])
		}
		if msg["accept"] != "" {
			req.Header.Set("Accept", msg["accept"])
		}
	}
	var reply *Reply
	b := backoff.NewExponentialBackOff()
	if config.RequestRetryLimit >= 0 {
		b.MaxElapsedTime = config.RequestRetryLimit
	}
	err = backoff.Retry(func() error {
		var res *http.Response
		start := time.Now()
		res, err = client.Do(req)
		elapsed := time.Now().Sub(start)
		if metricLabel != "" {
			userCodeDurationHistogram.WithLabelValues(metricLabel).Observe(elapsed.Seconds())
		}
		if config.ActorBusyTimeout > 0 && elapsed > config.ActorBusyTimeout/2 {
			if err != nil {
				logger.Info("%v with path %v completed with an error in %v seconds", method, msg["path"], elapsed.Seconds())
			} else {
				logger.Info("%v with path %v completed normally in %v seconds", method, msg["path"], elapsed.Seconds())
			}
		}
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				reply = &Reply{StatusCode: http.StatusRequestTimeout, Payload: err.Error(), ContentType: "text/plain"}
				return nil
			}
			logger.Warning("failed to invoke %s: %v", msg["path"], err)
			if err == ctx.Err() {
				return backoff.Permanent(err)
			}
			return err
		}
		buf, err := ioutil.ReadAll(res.Body) // TODO size limit?
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				reply = &Reply{StatusCode: http.StatusRequestTimeout, Payload: err.Error(), ContentType: "text/plain"}
				return nil
			}
			logger.Warning("failed to invoke %s: %v", msg["path"], err)
			if err == ctx.Err() {
				return backoff.Permanent(err)
			}
			return err
		}
		res.Body.Close()
		if length, err := strconv.Atoi(res.Header.Get("Content-Length")); err == nil && len(buf) != length {
			logger.Warning("failed to invoke %s: unexpected content length (%d != %d)", msg["path"], length, len(buf))
			return errors.New("unexpected content length")
		}
		reply = &Reply{StatusCode: res.StatusCode, Payload: string(buf), ContentType: res.Header.Get("Content-Type")}
		return nil
	}, backoff.WithContext(b, ctx))
	if ctx.Err() != nil {
		CloseIdleConnections() // don't keep connection alive once ctx is cancelled
	}
	return reply, err
}

// CloseIdleConnections closes idle connections
func CloseIdleConnections() {
	client.CloseIdleConnections()
}
