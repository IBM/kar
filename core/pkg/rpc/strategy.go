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
	"encoding/json"
	"sync"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
)

// Custom strategy to assign partitions to consumer group members
type strategy struct{}

func (s *strategy) Name() string { return "custom" }

// Assign partitions to group members
func (s *strategy) Plan(members map[string]sarama.ConsumerGroupMemberMetadata, topics map[string][]int32) (sarama.BalanceStrategyPlan, error) {
	logger.Info("enter plan")

	partitions := topics[appTopic]     // topic partitions
	node2member := map[string]string{} // a map from node id to sarama member id
	recovery = map[int32]bool{}

	// reset the routing tables
	service2nodes = map[string][]string{}
	// node2partition = map[string]int32{} // keep the info we already have
	session2NodeCache = new(sync.Map)
	liveNodes := map[string]struct{}{}

	// reset partition message counts
	newest = map[int32]int64{}

	// find partitions attached to live node
	// build node2member map
	// update node2partition map
	for member, meta := range members {
		var v info
		err := json.Unmarshal(meta.UserData, &v)
		if err != nil {
			return nil, err
		}
		node2member[v.Node] = member
		for _, s := range v.Services {
			service2nodes[s] = append(service2nodes[s], v.Node)
		}
		liveNodes[v.Node] = struct{}{}
		if v.Partition > 0 { // do not overwrite partition assignment with outdated metadata
			node2partition[v.Node] = v.Partition
		}
		if node2partition[v.Node] == 0 {
			offset0 = 0 // new node, revisit requests for unavailable services (TODO could we reliably check for new services instead?)
		}
	}

	for n, p := range node2partition {
		if _, ok := liveNodes[n]; !ok {
			delete(node2partition, n) // discard dead nodes
		} else if p > 0 {
			recovery[p] = true // partition connected to live node
		}
	}

	// get newest offsets for non-empty partitions
	clean := true
	for _, p := range partitions {
		min, err := consumerClient.GetOffset(appTopic, p, sarama.OffsetOldest)
		if err != nil {
			return nil, err
		}
		max, err := consumerClient.GetOffset(appTopic, p, sarama.OffsetNewest)
		if err != nil {
			return nil, err
		}
		if min < max {
			newest[p] = max
			if p != 0 && !recovery[p] {
				clean = false
				logger.Info("partition %d is not empty: %d < %d", p, min, max)
			} else if p == 0 && offset0 < max {
				max0 = max
				clean = false // only recover if there is a new node or new content in partition 0
				// TODO better "new" content detection
			}
		}
	}

	// find free partitions for new members
	next := 1
	for node := range node2member {
		for ; node2partition[node] == 0 && next < len(partitions); next++ {
			if !recovery[partitions[next]] && newest[partitions[next]] == 0 {
				node2partition[node] = partitions[next]
			}
		}
		if node2partition[node] == 0 {
			missing := int32(0)
			for node := range node2member {
				if node2partition[node] == 0 {
					missing++
				}
			}
			if err := admin.CreatePartitions(appTopic, int32(len(partitions))+missing, nil, false); err != nil {
				return nil, err
			}
			return nil, errTooFewPartitions
		}
	}

	// instantiate plan
	plan := make(sarama.BalanceStrategyPlan, len(members))

	if !clean {
		// entering recovery, assign non-empty partitions + 0 to leader
		for _, p := range partitions {
			if p == 0 || newest[p] > 0 {
				plan.Add(node2member[self.Node], appTopic, p)
			}
		}
	} else {
		// recovery not needed, assign partitions to group members
		for node, member := range node2member {
			plan.Add(member, appTopic, node2partition[node])
		}

		// reset map to signal recovery is not necessary
		recovery = nil
	}

	logger.Info("exit plan")

	return plan, nil
}

// We do not rely on sarama to persist the assignments between generations as it interferes with the other pieces of info we need to exchange
func (s *strategy) AssignmentData(memberID string, topics map[string][]int32, generationID int32) ([]byte, error) {
	return nil, nil
}

func updateRoutes() error {
	logger.Info("enter update routes")

	// retrieve consumer group description
	groups, err := admin.DescribeConsumerGroups([]string{appTopic})
	if err != nil {
		return err
	}
	members := groups[0].Members

	// reset routing tables
	service2nodes = map[string][]string{}
	node2partition = map[string]int32{}
	session2NodeCache = new(sync.Map)

	// rebuild tables
	for _, member := range members {
		// build service2nodes map from metadata
		meta, err1 := member.GetMemberMetadata()
		if err1 != nil {
			return err1
		}
		var data info
		if err1 = json.Unmarshal(meta.UserData, &data); err1 != nil {
			return err1
		}
		for _, s := range data.Services {
			service2nodes[s] = append(service2nodes[s], data.Node)
		}

		// build node2partition map from assignments
		// the partition info in the metadata cannot be used as it reflects the previous generation
		assignment, err1 := member.GetMemberAssignment()
		if err1 != nil {
			return err1
		}
		node2partition[data.Node] = assignment.Topics[appTopic][0]
	}

	logger.Info("exit update routes")

	return nil
}
