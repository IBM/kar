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

package pubsub

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
)

// ErrRouteToActorTimeout indicates a timeout while waiting for a viable route to an Actor type.
var ErrRouteToActorTimeout = errors.New("timeout occurred while looking for actor type")

// ErrRouteToServiceTimeout indicates a timeout while waiting for a viable route to a Service endpoint.
var ErrRouteToServiceTimeout = errors.New("timeout occurred while looking for service instance")

// use debug logger for errors returned to caller

// RouteToService maps a service to a partition (keep trying) -- only public so rpc can call it
func RouteToService(ctx context.Context, service string) (partition int32, sidecar string, err error) {
	for {
		mu.RLock()
		sidecars := replicas[service]
		if len(sidecars) != 0 {
			sidecar = sidecars[rand.Int31n(int32(len(sidecars)))]       // select random sidecar from list
			partitions := routes[sidecar]                               // a live sidecar always has partitions
			partition = partitions[rand.Int31n(int32(len(partitions)))] // select a random partition from list
			mu.RUnlock()
			return
		}
		ch := tick
		mu.RUnlock()
		logger.Info("no sidecar for service %s, waiting for new session", service)

		if config.MissingComponentTimeout > 0 {
			select {
			case <-ch:
			case <-ctx.Done():
				err = ctx.Err()
				return
			case <-time.After(config.MissingComponentTimeout):
				err = ErrRouteToServiceTimeout
				return
			}
		} else {
			select {
			case <-ch:
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}
	}
}

// RouteToSidecar maps a sidecar to a partition (no retries)
func RouteToSidecar(sidecar string) (int32, error) {
	mu.RLock()
	partitions := routes[sidecar]
	mu.RUnlock()
	if len(partitions) == 0 { // no partition matching this sidecar
		logger.Debug("no partition for sidecar %s", sidecar)
		return 0, ErrUnknownSidecar
	}
	return partitions[rand.Int31n(int32(len(partitions)))], nil // select random partition from list
}

// RouteToActor maps an actor to a stable sidecar to a random partition (keep trying)
// only switching to a new sidecar if the existing sidecar has died
func RouteToActor(ctx context.Context, t, id string) (partition int32, sidecar string, err error) {
	for { // keep trying
		sidecar, err = GetSidecar(ctx, t, id) // retrieve already assigned sidecar if any
		if err != nil {
			return // store error
		}
		if sidecar != "" { // sidecar is already assigned
			partition, err = RouteToSidecar(sidecar) // find partition for sidecar
			if err == nil {
				return // found sidecar and partition
			}
			logger.Debug("sidecar %s for actor type %s, id %s is no longer available", sidecar, t, id)
		}
		// assign new sidecar
		expected := sidecar // remember current value for CAS
		for {
			mu.RLock()
			sidecars := hosts[t]
			if len(sidecars) != 0 {
				sidecar = sidecars[rand.Int31n(int32(len(sidecars)))] // select random sidecar from list
				mu.RUnlock()
				break
			}
			ch := tick
			mu.RUnlock()
			logger.Info("no sidecar for actor type %s, waiting for new session", t)
			if config.MissingComponentTimeout > 0 {
				select {
				case <-ch:
				case <-ctx.Done():
					err = ctx.Err()
					return
				case <-time.After(config.MissingComponentTimeout):
					err = ErrRouteToActorTimeout
					return
				}
			} else {
				select {
				case <-ch:
				case <-ctx.Done():
					err = ctx.Err()
					return
				}
			}
		}
		logger.Debug("trying to save new sidecar %s for actor type %s, id %s", sidecar, t, id)
		_, err = CompareAndSetSidecar(ctx, t, id, expected, sidecar) // try saving sidecar
		if err != nil {
			return // store error
		}
		// loop around
	}
}

func SendBytes(ctx context.Context, partition int32, m []byte) error {
	_, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Value:     sarama.ByteEncoder(m),
	})
	if err != nil {
		logger.Error("failed to send message to partition %d: %v", partition, err)
		return err
	}
	logger.Debug("sent message at partition %d, offset %d", partition, offset)
	return nil
}

// Sidecars returns all the reachable sidecars
func Sidecars() []string {
	mu.RLock()
	sidecars := []string{}
	for sidecar := range routes {
		sidecars = append(sidecars, sidecar)
	}
	mu.RUnlock()
	return sidecars
}
