package rpc

import (
	"context"
	"time"

	rpclib "github.com/IBM/kar/core/pkg/rpc"
)

// Config specifies the Kafka configuration
type Config = rpclib.Config

// Target of an invocation
type Target = rpclib.Target

// Service implements Target
type Service = rpclib.Service

// Session implements Target
type Session = rpclib.Session

// Node implements Target
type Node = rpclib.Node

// Destination is a Target and a method
type Destination = rpclib.Destination

// Handler for method
type Handler = rpclib.Handler

// Data transformer applied to convert external events to Tell payloads
type Transformer = rpclib.Transformer

// Result of async call
type Result = rpclib.Result

// A Publisher makes it possible to publish events to Kafka
type Publisher = rpclib.Publisher

// use new implementation
func UsePlacementCache(placementCache bool) {
	rpclib.PlacementCache = placementCache
}

// Register method handler
func Register(method string, handler Handler) {
	rpclib.Register(method, handler)
}

// Connect to Kafka
func Connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	return rpclib.Connect(ctx, topic, conf, services...)
}

// Call method and wait for result
func Call(ctx context.Context, dest Destination, deadline time.Time, value []byte) ([]byte, error) {
	return rpclib.Call(ctx, dest, deadline, value)
}

// Call method and return immediately (result will be discarded)
func Tell(ctx context.Context, dest Destination, deadline time.Time, value []byte) error {
	return rpclib.Tell(ctx, dest, deadline, value)
}

// Call method and return a request id and a result channel
func Async(ctx context.Context, dest Destination, deadline time.Time, value []byte) (string, <-chan Result, error) {
	return rpclib.Async(ctx, dest, deadline, value)
}

// Reclaim resources associated with async request id
func Reclaim(requestID string) {
	rpclib.Reclaim(requestID)
}

// GetTopology returns a map from node ids to services
func GetTopology() (map[string][]string, <-chan struct{}) {
	return rpclib.GetTopology()
}

// GetServices returns the sorted list of services currently available
func GetServices() ([]string, <-chan struct{}) {
	return rpclib.GetServices()
}

// GetAllSessions returns a map from Session names to all known IDs for each name
func GetAllSessions(ctx context.Context, sessionPrefixFilter string) (map[string][]string, error) {
	return rpclib.GetAllSessions(ctx, sessionPrefixFilter)
}

// GetNodeID returns the node id for the current node
func GetNodeID() string {
	return rpclib.GetNodeID()
}

// GetNodeIDs returns the sorted list of live node ids and a channel to be notified of changes
func GetNodeIDs() ([]string, <-chan struct{}) {
	return rpclib.GetNodeIDs()
}

// GetServiceNodeIDs returns the sorted list of live node ids for a given service
func GetServiceNodeIDs(service string) ([]string, <-chan struct{}) {
	return rpclib.GetServiceNodeIDs(service)
}

// GetPartition returns the partition for the current node
func GetPartition() int32 {
	return rpclib.GetPartition()
}

// GetPartitions returns the sorted list of partitions in use and a channel to be notified of changes
func GetPartitions() ([]int32, <-chan struct{}) {
	return rpclib.GetPartitions()
}

// GetSessionNodeId returns the node responsible for the specified session if defined or "" if not
func GetSessionNodeID(ctx context.Context, session Session) (string, error) {
	return rpclib.GetSessionNodeID(ctx, session)
}

// DelSession forgets the node id responsible for the specified session
func DelSession(ctx context.Context, session Session) error {
	return rpclib.DelSession(ctx, session)
}

// CreateTopic attempts to create the specified topic using the given parameters
func CreateTopic(conf *Config, topic string, parameters string) error {
	return rpclib.CreateTopic(conf, topic, parameters)
}

// DeleteTopic attempts to delete the specified topic
func DeleteTopic(conf *Config, topic string) error {
	return rpclib.DeleteTopic(conf, topic)
}

// NewPublisher returns a new event publisher
func NewPublisher(conf *Config) (Publisher, error) {
	return rpclib.NewPublisher(conf)
}

// Subscribe to a topic
func Subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, dest Destination, transform Transformer) (<-chan struct{}, error) {
	return rpclib.Subscribe(ctx, conf, topic, group, oldest, dest, transform)
}

func ChoosePartition() int32 {
	return 0
}
