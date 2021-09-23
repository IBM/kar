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
	"crypto/tls"
	"encoding/json"
	"errors"
	"sync"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

var (
	client   sarama.Client       // shared client
	producer sarama.SyncProducer // shared idempotent producer

	// id is the unique id of this sidecar instance
	id = uuid.New().String()

	// routes
	myTopic   string
	replicas  map[string][]string // map services to sidecars
	hosts     map[string][]string // map actor types to sidecars
	routes    map[string][]int32  // map sidecards to partitions
	address   string              // host:port of sidecar http server (for peer-to-peer connections)
	addresses map[string]string   // map sidecards to addresses
	tick      = make(chan struct{})
	joined    = tick
	mu        = &sync.RWMutex{}

	manualPartitioner = sarama.NewManualPartitioner(myTopic)

	errTooFewPartitions = errors.New("too few partitions")

	errUnknownSidecar = errors.New("unknown sidecar")
)

func partitioner(t string) sarama.Partitioner {
	if t == myTopic {
		return manualPartitioner
	}
	return sarama.NewRandomPartitioner(t)
}

// dial connects Kafka producer
func dial() error {
	conf, err := newConfig()
	if err != nil {
		return err
	}

	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Idempotent = true
	conf.Producer.Partitioner = partitioner
	conf.Net.MaxOpenRequests = 1

	client, err = sarama.NewClient(myConfig.Brokers, conf)
	if err != nil {
		logger.Error("failed to instantiate Kafka client: %v", err)
		return err
	}

	producer, err = sarama.NewSyncProducerFromClient(client)
	if err != nil {
		logger.Error("failed to instantiate Kafka producer: %v", err)
		return err
	}

	return nil
}

func newConfig() (*sarama.Config, error) {
	conf := sarama.NewConfig()
	var err error
	conf.Version, err = sarama.ParseKafkaVersion(myConfig.Version)
	if err != nil {
		logger.Error("failed to parse Kafka version: %v", err)
		return nil, err
	}
	conf.ClientID = "kar"
	if myConfig.Password != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = myConfig.User
		conf.Net.SASL.Password = myConfig.Password
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	if myConfig.EnableTLS {
		conf.Net.TLS.Enable = true
		// TODO support custom CA certificate
		if myConfig.TLSSkipVerify {
			conf.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
	return conf, nil
}

// partitions returns the set of partitions claimed by this sidecar and a channel for change notifications
func partitions() ([]int32, <-chan struct{}) {
	mu.RLock()
	t := tick
	r := routes[id]
	mu.RUnlock()
	return r, t
}

// joinSidecarToMesh joins the sidecar to the application mesh and returns a channel that will be closed when the sidecar leaves the mesh
func joinSidecarToMesh(ctx context.Context, f func(ctx context.Context, value []byte, markAsDone func())) (<-chan struct{}, error) {
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		logger.Error("failed to instantiate Kafka cluster admin: %v", err)
		return nil, err
	}
	err = admin.CreateTopic(myTopic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
	if err != nil {
		err = admin.CreateTopic(myTopic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
	}
	if err != nil {
		if e, ok := err.(*sarama.TopicError); !ok || e.Err != sarama.ErrTopicAlreadyExists { // ignore ErrTopicAlreadyExists
			logger.Error("failed to create Kafka topic: %v", err)
			return nil, err
		}
	}
	ch, _, err := Subscribe_PS(ctx, myTopic, myTopic, &Options_PS{master: true, OffsetOldest: true}, f)
	return ch, err
}

// CreateTopic attempts to create the specified topic using the given parameters
func createTopic(conf *Config, topic string, parameters string) error {
	var params sarama.TopicDetail
	var err error

	if parameters != "" {
		err = json.Unmarshal([]byte(parameters), &params)
		if err != nil {
			logger.Error("failed to unmarshal parameters to createTopic %v: %v", topic, err)
			return err
		}
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		logger.Error("failed to instantiate Kafka cluster admin: %v", err)
		return err
	}

	if parameters == "" { // No parameters given, attempt default creation values
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
		if err != nil {
			err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
		}
	} else {
		err = admin.CreateTopic(topic, &params, false)
	}
	if err != nil {
		logger.Error("failed to create Kafka topic %v: %v", topic, err)
		return err
	}
	return nil
}

// DeleteTopic attempts to delete the specified topic
func deleteTopic(conf *Config, topic string) error {
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		logger.Error("failed to instantiate Kafka cluster admin: %v", err)
		return err
	}
	err = admin.DeleteTopic(topic)
	if err != nil {
		logger.Error("failed to delete Kafka topic %v: %v", topic, err)
		return err
	}
	return nil
}

func getTopology() (map[string][]string, <-chan struct{}) {
	toplogy := make(map[string][]string)

	mu.RLock()
	for sidecar, _ := range addresses {
		toplogy[sidecar] = []string{}
	}
	for service, sidecars := range replicas {
		for _, sidecar := range sidecars {
			toplogy[sidecar] = append(toplogy[sidecar], service)
		}
	}
	for actor, sidecars := range hosts {
		for _, sidecar := range sidecars {
			toplogy[sidecar] = append(toplogy[sidecar], actor)
		}
	}
	mu.RUnlock()

	return toplogy, nil // TODO: Kar doesn't use the notification channel, so not bothering to implement it
}

// isLiveSidecar return true if the argument sidecar is currently part of the application mesh
func isLiveSidecar(sidecar string) bool {
	_, ok := addresses[sidecar]
	return ok
}
