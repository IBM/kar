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
	"net/http"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

// Publish publishes a message on a topic
func Publish(topic string, message []byte) ( /* httpStatusCode */ int, error) {
	partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	})
	if err != nil {
		logger.Error("failed to send message on topic %s: %v", topic, err)
		if err == sarama.ErrUnknownTopicOrPartition {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	logger.Debug("sent message on topic %s, partition %d, offset %d", topic, partition, offset)
	return http.StatusOK, nil
}
