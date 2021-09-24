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
	"encoding/json"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
)

func createTopic(conf *Config, topic string, parameters string) error {
	var params sarama.TopicDetail
	var err error

	if parameters != "" {
		err = json.Unmarshal([]byte(parameters), &params)
		if err != nil {
			return err
		}
	}
	admin, err := sarama.NewClusterAdmin(conf.Brokers, configureClient(conf))
	if err != nil {
		return err
	}
	defer admin.Close()
	if parameters == "" { // no parameters given, attempt default creation values
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
		if err != nil {
			err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
		}
	} else {
		err = admin.CreateTopic(topic, &params, false)
	}
	return err
}

func deleteTopic(conf *Config, topic string) error {
	admin, err := sarama.NewClusterAdmin(conf.Brokers, configureClient(conf))
	if err != nil {
		return err
	}
	defer admin.Close()
	err = admin.DeleteTopic(topic)
	if err != sarama.ErrUnknownTopicOrPartition {
		return err
	}
	return nil
}

type publisher struct {
	producer sarama.SyncProducer
}

func newPublisher(conf *Config) (Publisher, error) {
	p, err := sarama.NewSyncProducer(conf.Brokers, configureClient(conf))
	if err != nil {
		return nil, err
	}
	return publisher{producer: p}, nil
}

func (p publisher) Close() error {
	return p.producer.Close()
}

func (p publisher) Publish(topic string, value []byte) error {
	_, _, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(value),
	})
	return err
}

type subscriber struct {
	handler func([]byte)
}

func (s *subscriber) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (*subscriber) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (s *subscriber) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		s.handler(msg.Value)
		session.MarkMessage(msg, "")
	}
	return nil
}

func subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, handler func([]byte)) error {
	config := configureClient(conf)
	if oldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	cg, err := sarama.NewConsumerGroup(conf.Brokers, group, configureClient(conf))
	if err != nil {
		return err
	}

	go func() {
		for {
			if err1 := cg.Consume(ctx, []string{topic}, &subscriber{handler: handler}); err1 != nil {
				logger.Error("subscriber error: %v", err1)
				break
			}
			if ctx.Err() != nil {
				break
			}
		}
		cg.Close()
	}()

	return nil
}
