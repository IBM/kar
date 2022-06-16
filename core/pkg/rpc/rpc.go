//
// Copyright IBM Corporation 2020,2022
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
	"fmt"
	"time"
)

// Config specifies the Kafka configuration
type Config struct {
	Version            string   // Kafka version
	Brokers            []string // Kafka brokers
	User               string   // Kafka SASL user
	Password           string   // Kafka SASL password
	EnableTLS          bool
	TLSSkipVerify      bool
	TopicConfig        map[string]*string
	SessionBusyTimeout time.Duration
	Cancellation       bool
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
	Name           string
	ID             string
	Flow           string
	DeferredLockID string
}

// Node implements Target
type Node struct {
	ID string
}

func (s Service) target() {}
func (s Session) target() {}
func (s Node) target()    {}

type Destination struct {
	Target Target
	Method string
}

// Handler for method
type ServiceHandler func(context.Context, Service, []byte) ([]byte, error)
type SessionHandler func(context.Context, Session, *SessionInstance, string, []byte) (*Destination, []byte, error)
type NodeHandler func(context.Context, Node, []byte) ([]byte, error)

// Data transformer applied to convert external events to Tell payloads
type Transformer func(context.Context, []byte) ([]byte, error)

// Result of async call
type Result struct {
	Value []byte
	Err   error
}

// An instance of a Session
type SessionInstance struct {
	Name       string
	ID         string
	ActiveFlow string
	Activated  bool
	lastAccess time.Time
	next       chan struct{} // coordination of queued tasks
	lock       chan struct{} // entry lock, never held for long, no need to watch ctx.Done()
	valid      bool          // false iff entry has been removed from table
}

func (a SessionInstance) String() string {
	return fmt.Sprintf("{Name: %v, ID: %v, ActiveFlow: %v, Activated: %v}", a.Name, a.ID, a.ActiveFlow, a.Activated)
}

// Register method handler
func RegisterService(method string, handler ServiceHandler) {
	registerService(method, handler)
}

// Register method handler
func RegisterSession(method string, handler SessionHandler) {
	registerSession(method, handler)
}

// Register method handler
func RegisterNode(method string, handler NodeHandler) {
	registerNode(method, handler)
}

// Connect to Kafka
func Connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	return connect(ctx, topic, conf, services...)
}

// Call method and wait for result
func Call(ctx context.Context, dest Destination, deadline time.Time, parentID string, value []byte) ([]byte, error) {
	return call(ctx, dest, deadline, parentID, value)
}

// Call method and return immediately (result will be discarded)
func Tell(ctx context.Context, dest Destination, deadline time.Time, parentID string, value []byte) error {
	return tell(ctx, dest, deadline, parentID, value)
}

// Call method and return a request id and a result channel
func Async(ctx context.Context, dest Destination, deadline time.Time, value []byte) (string, <-chan Result, error) {
	return async(ctx, dest, deadline, "", value)
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

// GetLocalActivatedSessions returns a map from Session names IDs with local Activated SessionInstances
// If name is not "", only SessionInstances with the given Name are returned.
func GetLocalActivatedSessions(ctx context.Context, name string) map[string][]string {
	return getLocalActivatedSessions(ctx, name)
}

// CollectInactiveSessions deactives SessionInstances that have not been used since the given time
func CollectInactiveSessions(ctx context.Context, time time.Time, callback func(context.Context, *SessionInstance)) {
	collectInactiveSessions(ctx, time, callback)
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

// BindingPartition
func BindingPartition() int32 {
	return 0
}

// GetPartitions returns the sorted list of partitions in use and a channel to be notified of changes
// TODO: fix hack, for now returns the set {0} as this method is only used to reload bindings
func GetPartitions() ([]int32, <-chan struct{}) {
	return getPartitions()
}

// GetSessionNodeId returns the node responsible for the specified session if defined or "" if not
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
type Publisher interface {
	Publish(topic string, value []byte) error
	Close() error
}

// NewPublisher returns a new event publisher
func NewPublisher(conf *Config) (Publisher, error) {
	return newPublisher(conf)
}

// Subscribe to a topic
func Subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, dest Destination, transform Transformer) (<-chan struct{}, error) {
	return subscribe(ctx, conf, topic, group, oldest, dest, transform)
}
