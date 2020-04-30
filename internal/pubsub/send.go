package pubsub

import (
	"context"
	"encoding/json"
	"math/rand"
	"strconv"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// use debug logger for errors returned to caller

// routeToService maps a service to a partition (keep trying)
func routeToService(ctx context.Context, service string) (partition int32, sidecar string, err error) {
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
		select {
		case <-ch:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
		// TODO timeout
	}
}

// routeToSidecar maps a sidecar to a partition (no retries)
func routeToSidecar(sidecar string) (int32, error) {
	mu.RLock()
	partitions := routes[sidecar]
	mu.RUnlock()
	if len(partitions) == 0 { // no partition matching this sidecar
		logger.Debug("no partition for sidecar %s", sidecar)
		return 0, ErrUnknownSidecar
	}
	return partitions[rand.Int31n(int32(len(partitions)))], nil // select random partition from list
}

// routeToActor maps an actor to a stable sidecar to a random partition (keep trying)
// only switching to a new sidecar if the existing sidecar has died
func routeToActor(ctx context.Context, t, id string) (partition int32, sidecar string, err error) {
	for { // keep trying
		sidecar, err = GetSidecar(t, id) // retrieve already assigned sidecar if any
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
			select {
			case <-ch:
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
			// TODO timeout
		}
		logger.Debug("trying to save new sidecar %s for actor type %s, id %s", sidecar, t, id)
		_, err = CompareAndSetSidecar(t, id, expected, sidecar) // try saving sidecar
		if err != nil {
			return // store error
		}
		// loop around
	}
}

// Send sends message to receiver
func Send(ctx context.Context, msg map[string]string) error {
	select { // make sure we have joined
	case <-joined:
	case <-ctx.Done():
		return ctx.Err()
	}
	var partition int32
	var err error
	switch msg["protocol"] {
	case "service": // route to service
		partition, msg["sidecar"], err = routeToService(ctx, msg["service"])
		if err != nil {
			logger.Debug("failed to route to service %s: %v", msg["service"], err)
			return err
		}
	case "actor": // route to actor
		partition, msg["sidecar"], err = routeToActor(ctx, msg["type"], msg["id"])
		if err != nil {
			logger.Debug("failed to route to actor type %s, id %s: %v", msg["type"], msg["id"], err)
			return err
		}
	case "sidecar": // route to sidecar
		partition, err = routeToSidecar(msg["sidecar"])
		if err != nil {
			logger.Debug("failed to route to sidecar %s: %v", msg["sidecar"], err)
			return err
		}
	case "partition": // route to partition
		p, err := strconv.ParseInt(msg["partition"], 10, 32)
		if err != nil {
			logger.Debug("failed to route to partition %s: %v", msg["partition"], err)
			return err
		}
		partition = int32(p)
	}
	m, err := json.Marshal(msg)
	if err != nil {
		logger.Debug("failed to marshal message: %v", err)
		return err
	}
	_, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Value:     sarama.ByteEncoder(m),
	})
	if err != nil {
		logger.Debug("failed to send message to partition %d: %v", partition, err)
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
