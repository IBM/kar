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

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/pubsub"
	"github.com/IBM/kar/core/pkg/logger"
)

// Register method handler
func register(method string, handler Handler) {

}

// Connect to Kafka
func connect(ctx context.Context, conf *Config, services ...string) (<-chan struct{}, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

// Call method and wait for result
func call(ctx context.Context, target Target, method string, value []byte) ([]byte, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

// Call method and return immediately (result will be discarded)
func tell(ctx context.Context, target Target, method string, value []byte) error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

// Call method and return a request id and a result channel
func async(ctx context.Context, target Target, method string, value []byte) (string, <-chan Result, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return "", nil, nil
}

// Reclaim resources associated with async request id
func reclaim(requestID string) {
	logger.Fatal("Unimplemented rpc-shim function")
}

// GetNodeID returns the node id for the current node
func getNodeID() string {
	return config.ID
}

// GetNodeIDs returns the sorted list of live node ids
func getNodeIDs() []string {
	return pubsub.Sidecars()
}

// GetPartition returns the partition for the current node
func getPartition() int32 {
	logger.Fatal("Unimplemented rpc-shim function")
	return 0
}

// GetPartitions returns the sorted list of partitions in use
func getPartitions() []int32 {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}
