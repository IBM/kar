//
// Copyright IBM Corporation 2020,2021
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

package rpc

import (
	"context"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
)

func register(method string, handler Handler) {
	handlers[method] = handler
}

func connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	myServices = append(myServices, services...)
	myTopic = topic
	myConfig = conf

	if err := dial(); err != nil {
		return nil, err
	}
	return joinSidecarToMesh(ctx, processMsg)
}

func getServices() ([]string, <-chan struct{}) {
	return myServices, nil // TODO: Kar doesn't use the notification channel, so not bothering to implement it
}

func getNodeID() string {
	return id
}

func getNodeIDs() ([]string, <-chan struct{}) {
	return sidecars(), nil // TODO: Kar doesn't use the notification channel, so not bothering to implement it
}

func getServiceNodeIDs(service string) ([]string, <-chan struct{}) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

func getPartition() int32 {
	logger.Fatal("Unimplemented rpc-shim function")
	return 0
}

func getSessionNodeID(ctx context.Context, session Session) (string, error) {
	return getSidecar(ctx, session.Name, session.ID)
}

func getPartitions() ([]int32, <-chan struct{}) {
	return partitions()
}

func delSession(ctx context.Context, session Session) error {
	_, err := compareAndSetSidecar(ctx, session.Name, session.ID, getNodeID(), "")
	return err
}

func newPublisher(conf *Config) (publisher, error) {
	// Config completely ignored in legacy impl; one shared Publisher for all events that is created unconditionally during startup
	return publisher{producer: producer}, nil
}

// Close publisher
func (p publisher) Close() error {
	// Nothing to do in legacy implementation; one shared Publisher for all events that is closed during shutdown
	return nil
}

func subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, dest Destination, transform Transformer) (<-chan struct{}, error) {
	f := func(ctx context.Context, msg message) {
		transformed, err := transform(ctx, msg.Value)
		if err != nil {
			logger.Error("failed to transform event from topic %s: %v", topic, err)
		}
		err = Tell(ctx, dest, time.Time{}, transformed)
		if err != nil {
			logger.Error("failed to tell target %v of event from topic %s: %v", dest.Target, topic, err)
		} else {
			msg.Mark()
		}
	}

	ch, err := subscribeImpl(ctx, topic, group, &subscriptionOptions{OffsetOldest: oldest, master: false}, f)
	return ch, err
}
