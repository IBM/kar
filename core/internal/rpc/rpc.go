package rpc

import (
	"context"
	"math/rand"
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

// Handler for method
type Handler = rpclib.Handler

// Data transformer applied to convert external events to Tell payloads
type Transformer = rpclib.Transformer

// Result of async call
type Result = rpclib.Result

// A Publisher makes it possible to publish events to Kafka
type Publisher = rpclib.Publisher

// which implementation to use
var rpcLib = false

// use new implementation
func UseRpcLib() {
	rpcLib = true
}

// Register method handler
func Register(method string, handler Handler) {
	if rpcLib {
		rpclib.Register(method, handler)
	} else {
		register(method, handler)
	}
}

// Connect to Kafka
func Connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	if rpcLib {
		return rpclib.Connect(ctx, topic, conf, services...)
	}
	return connect(ctx, topic, conf, services...)
}

// Call method and wait for result
func Call(ctx context.Context, target Target, method string, deadline time.Time, value []byte) ([]byte, error) {
	if rpcLib {
		return rpclib.Call(ctx, target, method, deadline, value)
	}
	return call(ctx, target, method, deadline, value)
}

// Call method and return immediately (result will be discarded)
func Tell(ctx context.Context, target Target, method string, deadline time.Time, value []byte) error {
	if rpcLib {
		return rpclib.Tell(ctx, target, method, deadline, value)
	}
	return tell(ctx, target, method, deadline, value)
}

// Call method and return a request id and a result channel
func Async(ctx context.Context, target Target, method string, deadline time.Time, value []byte) (string, <-chan Result, error) {
	if rpcLib {
		return rpclib.Async(ctx, target, method, deadline, value)
	}
	return async(ctx, target, method, deadline, value)
}

// Reclaim resources associated with async request id
func Reclaim(requestID string) {
	if rpcLib {
		rpclib.Reclaim(requestID)
	} else {
		reclaim(requestID)
	}
}

// GetTopology returns a map from node ids to services
func GetTopology() (map[string][]string, <-chan struct{}) {
	if rpcLib {
		rpclib.GetTopology()
	}
	return getTopology()
}

// GetServices returns the sorted list of services currently available
func GetServices() ([]string, <-chan struct{}) {
	if rpcLib {
		rpclib.GetServices()
	}
	return getServices()
}

// GetAllSessions returns a map from Session names to all known IDs for each name
func GetAllSessions(ctx context.Context, sessionPrefixFilter string) (map[string][]string, error) {
	if rpcLib {
		rpclib.GetAllSessions(ctx, sessionPrefixFilter)
	}
	return getAllSessions(ctx, sessionPrefixFilter)
}

// GetNodeID returns the node id for the current node
func GetNodeID() string {
	if rpcLib {
		return rpclib.GetNodeID()
	}
	return getNodeID()
}

// GetNodeIDs returns the sorted list of live node ids and a channel to be notified of changes
func GetNodeIDs() ([]string, <-chan struct{}) {
	if rpcLib {
		return getNodeIDs()
	}
	return getNodeIDs()
}

// GetServiceNodeIDs returns the sorted list of live node ids for a given service
func GetServiceNodeIDs(service string) ([]string, <-chan struct{}) {
	if rpcLib {
		return rpclib.GetServiceNodeIDs(service)
	}
	return getServiceNodeIDs(service)
}

// GetPartition returns the partition for the current node
func GetPartition() int32 {
	if rpcLib {
		return rpclib.GetPartition()
	}
	return getPartition()
}

// GetPartitions returns the sorted list of partitions in use and a channel to be notified of changes
func GetPartitions() ([]int32, <-chan struct{}) {
	if rpcLib {
		return rpclib.GetPartitions()
	}
	return getPartitions()
}

// GetSessionNodeId returns the node responsible for the specified session if defined or "" if not
func GetSessionNodeID(ctx context.Context, session Session) (string, error) {
	if rpcLib {
		return rpclib.GetSessionNodeID(ctx, session)
	}
	return getSessionNodeID(ctx, session)
}

// DelSession forgets the node id responsible for the specified session
func DelSession(ctx context.Context, session Session) error {
	if rpcLib {
		return rpclib.DelSession(ctx, session)
	}
	return delSession(ctx, session)
}

// CreateTopic attempts to create the specified topic using the given parameters
func CreateTopic(conf *Config, topic string, parameters string) error {
	if rpcLib {
		return rpclib.CreateTopic(conf, topic, parameters)
	}
	return createTopic(conf, topic, parameters)
}

// DeleteTopic attempts to delete the specified topic
func DeleteTopic(conf *Config, topic string) error {
	if rpcLib {
		return rpclib.DeleteTopic(conf, topic)
	}
	return deleteTopic(conf, topic)
}

// NewPublisher returns a new event publisher
func NewPublisher(conf *Config) (Publisher, error) {
	if rpcLib {
		return rpclib.NewPublisher(conf)
	}
	return newPublisher(conf)
}

// Subscribe to a topic
func Subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, target Target, method string, transform Transformer) (<-chan struct{}, error) {
	if rpcLib {
		return rpclib.Subscribe(ctx, conf, topic, group, oldest, target, method, transform)
	}
	return subscribe(ctx, conf, topic, group, oldest, target, method, transform)
}

func ChoosePartition() int32 {
	if rpcLib {
		return 0
	}
	ps, _ := GetPartitions()
	return ps[rand.Int31n(int32(len(ps)))]
}
