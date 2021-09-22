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
	"math/rand"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
)

// routeToService maps a service to a partition (keep trying)
func routeToService(ctx context.Context, service string, deadline time.Time) (partition int32, sidecar string, err error) {
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

		if !deadline.IsZero() {
			ctx2, cancel2 := context.WithDeadline(ctx, deadline)
			select {
			case <-ch:
				cancel2()
			case <-ctx2.Done():
				err = ctx2.Err()
				cancel2()
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

// routeToSidecar maps a sidecar to a partition (no retries)
func routeToSidecar(sidecar string) (int32, error) {
	mu.RLock()
	partitions := routes[sidecar]
	mu.RUnlock()
	if len(partitions) == 0 { // no partition matching this sidecar
		logger.Debug("no partition for sidecar %s", sidecar)
		return 0, errUnknownSidecar
	}
	return partitions[rand.Int31n(int32(len(partitions)))], nil // select random partition from list
}

// routeToActor maps an actor to a stable sidecar to a random partition (keep trying)
// only switching to a new sidecar if the existing sidecar has died
func routeToActor(ctx context.Context, t, id string, deadline time.Time) (partition int32, sidecar string, err error) {
	for { // keep trying
		sidecar, err = getSidecar(ctx, t, id) // retrieve already assigned sidecar if any
		if err != nil {
			return // store error
		}
		if sidecar != "" { // sidecar is already assigned
			partition, err = routeToSidecar(sidecar) // find partition for sidecar
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
			if !deadline.IsZero() {
				ctx2, cancel2 := context.WithDeadline(ctx, deadline)
				select {
				case <-ch:
					cancel2()
				case <-ctx2.Done():
					err = ctx2.Err()
					cancel2()
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
		_, err = compareAndSetSidecar(ctx, t, id, expected, sidecar) // try saving sidecar
		if err != nil {
			return // store error
		}
		// loop around
	}
}

func sendBytes(ctx context.Context, partition int32, m []byte) error {
	_, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     myTopic,
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

// sidecars returns all the reachable sidecars
func sidecars() []string {
	mu.RLock()
	sidecars := []string{}
	for sidecar := range routes {
		sidecars = append(sidecars, sidecar)
	}
	mu.RUnlock()
	return sidecars
}
