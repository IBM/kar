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
	lru "github.com/hashicorp/golang-lru"
)

// The info provided by each live node when rebalancing
type info struct {
	Node      string   // the uuid of the node
	Services  []string // the services provided by the node
	Partition int32    // the partition > 0 assigned to the node if known or 0 if not yet decided
}

var (
	// Application topic
	appTopic string

	// Kafka clients
	producerClient sarama.Client       // the client for the producer
	producer       sarama.SyncProducer // the producer used to send messages
	consumerClient sarama.Client       // the client for the consumer group
	admin          sarama.ClusterAdmin // the cluster admin

	// routing tables (partition 0 is reserved)
	self              = info{Node: uuid.New().String()} // service, node, partition (initially unknown == 0)
	service2nodes     map[string][]string               // the map from services to nodes providing these services
	node2partition    = map[string]int32{}              // the map from nodes to their assigned partitions
	session2NodeCache *lru.ARCCache                     // a cache of the mapping from sessions to their assigned Node
	mu                = new(sync.RWMutex)               // a RW mutex held when rebalancing (W) and sending messages (R)
	tick              = make(chan struct{})             // a channel closed at replaced at the end of rebalance

	head      int64                 // the next offset to read
	processor func(Message)         // the function to invoke on each incoming message
	closed    = make(chan struct{}) // channel closed after disconnecting from Kafka

	// recovery info
	recovery map[int32]bool  // map partition to true if connected to live process, false if not
	newest   map[int32]int64 // newest offset for non-empty partitions
	offset0  int64           // how far have we processed messages for unavailable services since the last node addition
	max0     int64           // newest partition 0 offset

	// errors
	ErrUnavailable      = errors.New("unavailable")
	errTooFewPartitions = errors.New("too few partitions")
)

func init() {
	session2NodeCache, _ = lru.NewARC(4096)
}

func configureClient(config *Config) *sarama.Config {
	conf := sarama.NewConfig()
	conf.Version, _ = sarama.ParseKafkaVersion(config.Version)

	if config.Password != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = config.User
		conf.Net.SASL.Password = config.Password
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	if config.EnableTLS {
		conf.Net.TLS.Enable = true
		// TODO support custom CA certificate
		if config.TLSSkipVerify {
			conf.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}

	conf.Producer.Return.Successes = true

	return conf
}

// Configure idempotent producer
func configureProducer(config *Config) *sarama.Config {
	conf := configureClient(config)

	conf.Net.MaxOpenRequests = 1
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Idempotent = true

	// we decide which partitions to send to
	conf.Producer.Partitioner = sarama.NewManualPartitioner

	return conf
}

// Configure consumer
func configureConsumer(config *Config) *sarama.Config {
	conf := configureClient(config)

	// in the absence of committed cursors, read all messages
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	// we decide how to assign partitions to nodes
	conf.Consumer.Group.Rebalance.Strategy = new(strategy)

	return conf
}

// Connect to Kafka and return a channel closed after disconnecting from Kafka
func Dial(ctx context.Context, topic string, conf *Config, services []string, f func(Message)) (<-chan struct{}, error) {
	appTopic = topic
	self.Services = services
	processor = f

	var err error

	// initialize producer client
	producerClient, err = sarama.NewClient(conf.Brokers, configureProducer(conf))
	if err != nil {
		return nil, err
	}

	// initialize producer
	producer, err = sarama.NewSyncProducerFromClient(producerClient)
	if err != nil {
		return nil, err
	}

	// initialize consumer client
	consumerClient, err = sarama.NewClient(conf.Brokers, configureConsumer(conf))
	if err != nil {
		return nil, err
	}

	// initialize cluster admin
	admin, err = sarama.NewClusterAdminFromClient(consumerClient)
	if err != nil {
		return nil, err
	}

	// marshal info
	consumerClient.Config().Consumer.Group.Member.UserData, _ = json.Marshal(self)

	// acquire W mutex
	mu.Lock()

	// initialize consumer group
	cg, err := sarama.NewConsumerGroupFromClient(appTopic, consumerClient)
	if err != nil {
		return nil, err
	}

	err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
	if err != nil {
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
	}
	if err != nil {
		if e, ok := err.(*sarama.TopicError); !ok || e.Err != sarama.ErrTopicAlreadyExists { // ignore ErrTopicAlreadyExists
			return nil, err
		}
	}

	go func() {
		for {
			if err1 := cg.Consume(ctx, []string{appTopic}, new(handler)); err1 != nil && err1 != errTooFewPartitions {
				logger.Fatal("Consumer error: %v", err1)
			}
			if ctx.Err() != nil {
				break
			}
		}
		cg.Close()
		consumerClient.Close()
		producer.Close()
		producerClient.Close()
		service2nodes = nil
		node2partition = nil
		session2NodeCache = nil
		close(closed)
		mu.Unlock()
	}()

	return closed, nil
}
