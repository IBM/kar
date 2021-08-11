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

// Package pubsub handles Kafka
package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/IBM/kar.git/core/internal/config"
	"github.com/IBM/kar.git/core/pkg/logger"
	"github.com/Shopify/sarama"
)

var (
	client   sarama.Client       // shared client
	producer sarama.SyncProducer // shared idempotent producer

	// routes
	topic     = "kar" + config.Separator + config.AppName
	replicas  map[string][]string // map services to sidecars
	hosts     map[string][]string // map actor types to sidecars
	routes    map[string][]int32  // map sidecards to partitions
	address   string              // host:port of sidecar http server (for peer-to-peer connections)
	addresses map[string]string   // map sidecards to addresses
	tick      = make(chan struct{})
	joined    = tick
	mu        = &sync.RWMutex{}

	manualPartitioner = sarama.NewManualPartitioner(topic)

	errTooFewPartitions = errors.New("too few partitions")

	// ErrUnknownSidecar error
	ErrUnknownSidecar = errors.New("unknown sidecar")
)

func partitioner(t string) sarama.Partitioner {
	if t == topic {
		return manualPartitioner
	}
	return sarama.NewRandomPartitioner(t)
}

// Dial connects Kafka producer
func Dial() error {
	conf, err := newConfig()
	if err != nil {
		return err
	}

	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Idempotent = true
	conf.Producer.Partitioner = partitioner
	conf.Net.MaxOpenRequests = 1

	client, err = sarama.NewClient(config.KafkaBrokers, conf)
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

// Close closes Kafka producer
func Close() {
	producer.Close()
	client.Close()
}

func newConfig() (*sarama.Config, error) {
	conf := sarama.NewConfig()
	var err error
	conf.Version, err = sarama.ParseKafkaVersion(config.KafkaVersion)
	if err != nil {
		logger.Error("failed to parse Kafka version: %v", err)
		return nil, err
	}
	conf.ClientID = "kar"
	if config.KafkaPassword != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = config.KafkaUsername
		conf.Net.SASL.Password = config.KafkaPassword
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	if config.KafkaEnableTLS {
		conf.Net.TLS.Enable = true
		// TODO support custom CA certificate
		if config.KafkaTLSSkipVerify {
			conf.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
	return conf, nil
}

// Partitions returns the set of partitions claimed by this sidecar and a channel for change notifications
func Partitions() ([]int32, <-chan struct{}) {
	mu.RLock()
	t := tick
	r := routes[config.ID]
	mu.RUnlock()
	return r, t
}

// Join joins the sidecar to the application and returns a channel of incoming messages
func Join(ctx context.Context, f func(Message), port int) (<-chan struct{}, error) {
	address = net.JoinHostPort(config.Hostname, strconv.Itoa(port))
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		logger.Error("failed to instantiate Kafka cluster admin: %v", err)
		return nil, err
	}
	err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
	if err != nil {
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
	}
	if err != nil {
		if e, ok := err.(*sarama.TopicError); !ok || e.Err != sarama.ErrTopicAlreadyExists { // ignore ErrTopicAlreadyExists
			logger.Error("failed to create Kafka topic: %v", err)
			return nil, err
		}
	}
	ch, _, err := Subscribe(ctx, topic, topic, &Options{master: true, OffsetOldest: true}, f)
	return ch, err
}

// CreateTopic attempts to create the specified topic using the given parameters
func CreateTopic(topic string, parameters string) error {
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
func DeleteTopic(topic string) error {
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

type sidecarData struct {
	Partitions []int32  `json:"partitions"`
	Address    string   `json:"address"`
	Actors     []string `json:"actors"`
	Services   []string `json:"services"`
}

// GetSidecars --
func GetSidecars(format string) (string, error) {
	information := make(map[string]*sidecarData)

	mu.RLock()
	for sidecar, partitions := range routes {
		information[sidecar] = &sidecarData{}
		information[sidecar].Partitions = append(information[sidecar].Partitions, partitions...)
	}
	for actor, sidecars := range hosts {
		for _, sidecar := range sidecars {
			information[sidecar].Actors = append(information[sidecar].Actors, actor)
		}
	}
	for service, sidecars := range replicas {
		for _, sidecar := range sidecars {
			information[sidecar].Services = append(information[sidecar].Services, service)
		}
	}
	for sidecar, address := range addresses {
		information[sidecar].Address = address
	}
	mu.RUnlock()

	if format == "json" || format == "application/json" {
		m, err := json.Marshal(information)
		if err != nil {
			logger.Error("failed to marshal sidecar information data: %v", err)
			return "", err
		}
		return string(m), nil
	}

	var str strings.Builder
	fmt.Fprint(&str, "\nSidecar\n : Actors\n : Services")
	for sidecar, sidecarInfo := range information {
		fmt.Fprintf(&str, "\n%v\n : %v\n : %v", sidecar, sidecarInfo.Actors, sidecarInfo.Services)
	}
	return str.String(), nil
}

// GetSidecarID --
func GetSidecarID(format string) (string, error) {
	if format == "json" || format == "application/json" {
		return fmt.Sprintf("{\"id\":\"%s\"}", config.ID), nil
	}

	return config.ID + "\n", nil
}

// Purge deletes the application topic
func Purge() error {
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return err
	}
	err = admin.DeleteTopic(topic)
	if err != sarama.ErrUnknownTopicOrPartition {
		return err
	}
	return nil
}

// IsLiveSidecar return true if the argument sidecar is currently part of the application mesh
func IsLiveSidecar(sidecar string) bool {
	_, ok := addresses[sidecar]
	return ok
}
