package rpc

import (
	"context"
	"time"
)

// Config specifies the Kafka configuration
type Config struct {
	Version       string   // Kafka version
	Brokers       []string // Kafka brokers
	User          string   // Kafka SASL user
	Password      string   // Kafka SASL password
	EnableTLS     bool
	TLSSkipVerify bool
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

func (s Service) target() {}
func (s Session) target() {}
func (s Node) target()    {}

// Handler for method
type Handler func(context.Context, Target, []byte) ([]byte, error)

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
func Connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	return connect(ctx, topic, conf, services...)
}

// Call method and wait for result
func Call(ctx context.Context, target Target, method string, deadline time.Time, value []byte) ([]byte, error) {
	return call(ctx, target, method, deadline, value)
}

// Call method and return immediately (result will be discarded)
func Tell(ctx context.Context, target Target, method string, deadline time.Time, value []byte) error {
	return tell(ctx, target, method, deadline, value)
}

// Call method and return a request id and a result channel
func Async(ctx context.Context, target Target, method string, deadline time.Time, value []byte) (string, <-chan Result, error) {
	return async(ctx, target, method, deadline, value)
}

// Reclaim resources associated with async request id
func Reclaim(requestID string) {
	reclaim(requestID)
}

// GetTopology returns a map from node ids to services
func GetTopology() (map[string][]string, <-chan struct{}) {
	return getTopology()
}

// GetServices returns the sorted list of services currently available
func GetServices() ([]string, <-chan struct{}) {
	return getServices()
}

// GetAllSessions returns a map from Session names to all known IDs for each name
func GetAllSessions(ctx context.Context, sessionPrefixFilter string) (map[string][]string, error) {
	return getAllSessions(ctx, sessionPrefixFilter)
}

// GetNodeID returns the node id for the current node
func GetNodeID() string {
	return getNodeID()
}

// GetNodeIDs returns the sorted list of live node ids and a channel to be notified of changes
func GetNodeIDs() ([]string, <-chan struct{}) {
	return getNodeIDs()
}

// GetServiceNodeIDs returns the sorted list of live node ids for a given service
func GetServiceNodeIDs(service string) ([]string, <-chan struct{}) {
	return getServiceNodeIDs(service)
}

// GetPartition returns the partition for the current node
func GetPartition() int32 {
	return getPartition()
}

// GetPartitions returns the sorted list of partitions in use and a channel to be notified of changes
func GetPartitions() ([]int32, <-chan struct{}) {
	return getPartitions()
}

// GetSession returns the node responsible for the specified session if defined or "" if not
func GetSessionNodeID(ctx context.Context, session Session) (string, error) {
	return getSessionNodeID(ctx, session)
}

// DelSession forgets the node id responsible for the specified session
func DelSession(ctx context.Context, session Session) error {
	return delSession(ctx, session)
}

// CreateTopic attempts to create the specified topic using the given parameters
func CreateTopic(conf *Config, topic string, parameters string) error {
	return createTopic(conf, topic, parameters)
}

// DeleteTopic attempts to delete the specified topic
func DeleteTopic(conf *Config, topic string) error {
	return deleteTopic(conf, topic)
}

// A Publisher makes it possible to publish events to Kafka
type Publisher struct {
	publisher publisher
}

// NewPublisher returns a new event publisher
func NewPublisher(conf *Config) (*Publisher, error) {
	return newPublisher(conf)
}

// Publish publishes a value to a topic
func (p *Publisher) Publish(topic string, value []byte) error {
	return p.publish(topic, value)
}

// Close publisher
func (p *Publisher) Close() error {
	return p.close()
}

// Subscribe to a topic
func Subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, handler func(ctx context.Context, value []byte, markAsDone func())) error {
	return subscribe(ctx, conf, topic, group, oldest, handler)
}
