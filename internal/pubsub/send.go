package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"

	"github.com/Shopify/sarama"
	"github.com/cenkalti/backoff/v4"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// use debug logger for errors returned to caller

// routeToService maps a service to a partition (keep trying)
func routeToService(ctx context.Context, service string) (partition int32, sidecar string, err error) {
	err = backoff.Retry(func() error { // keep trying
		mu.RLock()
		sidecars := replicas[service]
		if len(sidecars) == 0 { // no sidecar matching this service
			mu.RUnlock()
			logger.Info("no sidecar for service %s, retrying", service)
			return errors.New("no sidecar for service " + service) // keep trying
		}
		sidecar = sidecars[rand.Int31n(int32(len(sidecars)))]       // select random sidecar from list
		partitions := routes[sidecar]                               // a live sidecar always has partitions
		partition = partitions[rand.Int31n(int32(len(partitions)))] // select a random partition from list
		mu.RUnlock()
		return nil
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	return
}

// routeToSidecar maps a sidecar to a partition (no retries)
func routeToSidecar(sidecar string) (int32, error) {
	mu.RLock()
	partitions := routes[sidecar]
	mu.RUnlock()
	if len(partitions) == 0 { // no partition matching this sidecar
		logger.Debug("no partition for sidecar %s", sidecar)
		return 0, errors.New("no partition for sidecar " + sidecar)
	}
	return partitions[rand.Int31n(int32(len(partitions)))], nil // select random partition from list
}

// routeToActor maps an actor to a stable sidecar to a random partition
// only switching to a new sidecar if the existing sidecar has died
func routeToActor(ctx context.Context, t, id string) (partition int32, sidecar string, err error) {
	for { // keep trying
		sidecar, err = GetSidecar(t, id) // retrieve already assigned sidecar if any
		if err != nil {
			return // store error
		}
		if sidecar != "" { // sidecar is already assigned
			_, _, err = routeToService(ctx, config.ServiceName) // make sure routes have been initialized
			if err != nil {
				return // abort
			}
			partition, err = routeToSidecar(sidecar) // find partition for sidecar
			if err == nil {
				return // found sidecar and partition
			}
			logger.Debug("sidecar %s for actor type %s id %s is no longer available", sidecar, t, id)
		}
		expected := sidecar
		err = backoff.Retry(func() error { // keep trying
			mu.RLock()
			sidecars := hosts[t]
			if len(sidecars) == 0 { // no sidecar matching this actor type
				mu.RUnlock()
				logger.Info("no sidecar for actor type %s, retrying", t)
				return errors.New("no sidecar for actor type " + t)
			}
			sidecar = sidecars[rand.Int31n(int32(len(sidecars)))] // select random sidecar from list
			mu.RUnlock()
			return nil
		}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
		if err != nil {
			return // abort
		}
		logger.Debug("trying to save new sidecar %s for actor type %s id %s", sidecar, t, id)
		_, err = CompareAndSetSidecar(t, id, expected, sidecar) // try saving sidecar
		if err != nil {
			return // store error
		}
		// loop around
	}
}

// Send sends message to receiver
func Send(ctx context.Context, msg map[string]string) error {
	var partition int32
	var err error
	switch msg["protocol"] {
	case "service": // route to service
		var sidecar string
		partition, sidecar, err = routeToService(ctx, msg["service"])
		if err != nil {
			logger.Debug("failed to route to service %s: %v", msg["service"], err)
			return err
		}
		msg["sidecar"] = sidecar // add selected sidecar id to message
	case "actor": // route to actor
		var sidecar string
		partition, sidecar, err = routeToActor(ctx, msg["type"], msg["id"])
		if err != nil {
			logger.Debug("failed to route to actor type %s id $s %v: %v", msg["type"], msg["id"], err)
			return err
		}
		msg["sidecar"] = sidecar // add selected sidecar id to message
	case "sidecar": // route to sidecar
		partition, err = routeToSidecar(msg["sidecar"])
		if err != nil {
			logger.Debug("failed to route to sidecar %s: %v", msg["sidecar"], err)
			return err
		}
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
	logger.Debug("sent message on topic %s, at partition %d, offset %d", topic, partition, offset)
	return nil
}
