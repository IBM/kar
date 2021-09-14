package rpc

import (
	"context"
)

// Config specifies the Kafka configuration
type Config struct {
	Topic   string // Kafka application topic
	Version string // Kafka version
	Brokers string // comma-separated list of Kafka brokers
}

// Target of an invocation
type Target interface {
	target() // private marker
}

// Service implements Target
type Service struct {
	Name string
}

// Session implements Target
type Session struct {
	Name string
	ID   string
}

// Node implements Target
type Node struct {
	ID string
}

// Partition implements Target
type Partition struct {
	ID int32
}

// Handler for method
type Handler func(Target, []byte) ([]byte, error)

// Result of async call
type Result struct {
	Value []byte
	Err   error
}

// Register method handler
func Register(method string, handler Handler) {
	register(method, handler)
}

// Connect to Kafka
func Connect(ctx context.Context, conf *Config, services ...string) (<-chan struct{}, error) {
	return connect(ctx, conf, services...)
}

// Call method and wait for result
func Call(ctx context.Context, target Target, method string, value []byte) ([]byte, error) {
	return call(ctx, target, method, value)
}

// Call method and return immediately (result will be discarded)
func Tell(ctx context.Context, target Target, method string, value []byte) error {
	return tell(ctx, target, method, value)
}

// Call method and return a request id and a result channel
func Async(ctx context.Context, target Target, method string, value []byte) (string, <-chan Result, error) {
	return async(ctx, target, method, value)
}

// Reclaim resources associated with async request id
func Reclaim(requestID string) {
	reclaim(requestID)
}

// GetNodeID returns the node id for the current node
func GetNodeID() string {
	return getNodeID()
}

// GetNodeIDs returns the sorted list of live node ids
func GetNodeIDs() []string {
	return getNodeIDs()
}

// GetPartition returns the partition for the current node
func GetPartition() int32 {
	return getPartition()
}

// GetPartitions returns the sorted list of partitions in use
func GetPartitions() []int32 {
	return getPartitions()
}
