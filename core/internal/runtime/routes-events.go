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

package runtime

/*
 * This file contains the implementation of the portion of the
 * KAR REST API related to events.
 */

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/rpc"
	"github.com/Shopify/sarama"
	"github.com/julienschmidt/httprouter"
)

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/events events idActorSubscriptionCancelAll
//
// subscriptions
//
// ### Cancel all subscriptions
//
// This operation cancels all subscriptions for the actor instance specified in the path.
// The number of subscriptions cancelled is returned as the result of the operation.
//
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200SubscriptionCancelAllResult
//       500: response500
//       503: response503
//

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/events/{subscriptionId} events idActorSubscriptionCancel
//
// subscriptions/id
//
// ### Cancel a subscription
//
// This operation cancels the subscription for the actor instance specified in the path.
// If the subscription is successfully cancelled a `200` response with a body of `1` will be returned.
// If the subscription is not found, a `404` response will be returned unless
// the boolean query parameter `nilOnAbsent` is set to `true`. If `nilOnAbsent`
// is sent to true the `404` response will instead be a `200` with a body containing `0`.
//
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200SubscriptionCancelResult
//       404: response404
//       500: response500
//       503: response503
//

// swagger:route GET /v1/actor/{actorType}/{actorId}/events events idActorSubscriptionGetAll
//
// subscriptions
//
// ### Get all subscriptions
//
// This operation returns all subscriptions for the actor instance specified in the path.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200SubscriptionGetAllResult
//       500: response500
//       503: response503
//

// swagger:route GET /v1/actor/{actorType}/{actorId}/events/{subscriptionId} events idActorSubscriptionGet
//
// subscriptions/id
//
// ### Get a subscription
//
// This operation returns the subscription for the actor instance specified in the path.
// If there is no subscription with the id `subscriptionId` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`.
// If `nilOnAbsent` is true the `404` response will be replaced with
// a `200` response with a `nil` response body.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200SubscriptionGetResult
//       404: response404
//       500: response500
//       503: response503
//

// swagger:route PUT /v1/actor/{actorType}/{actorId}/events/{subscriptionId} events idActorSubscribe
//
// subscriptions/id
//
// ### Subscribe to a topic
//
// Subscribe the actor instance using the subscriptionId specified in the path
// as described by the data provided in the request body.
// If there is already a subscription for the target actor instance with the same subscriptionId,
// that existing subscription will be updated based on the request body.
// The operation will not return until after the actor instance is subscribed.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       201: response201
//       204: response204
//       500: response500
//       503: response503
//
func routeImplSubscription(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var action string
	body := ""
	noa := "false"
	switch r.Method {
	case "GET":
		action = "get"
		noa = r.FormValue("nilOnAbsent")
	case "PUT":
		action = "set"
		body = ReadAll(r)
	case "DELETE":
		action = "del"
		noa = r.FormValue("nilOnAbsent")
	default:
		http.Error(w, fmt.Sprintf("Unsupported method %v", r.Method), http.StatusMethodNotAllowed)
		return
	}
	reply, err := Bindings(ctx, "subscriptions", Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("subscriptionId"), noa, action, body, r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		w.Header().Add("Content-Type", reply.ContentType)
		w.WriteHeader(reply.StatusCode)
		fmt.Fprint(w, reply.Payload)
	}
}

// swagger:route POST /v1/event/{topic}/publish events idEventPublish
//
// publish
//
// ### Publish an event to a topic
//
// The event provided as the request body will be published on `topic`.
// When the operation returns successfully, the event is guaranteed to
// eventually be published to the targeted topic.
//
//     Schemes: http
//     Consumes:
//     - application/*
//     Responses:
//       200: response200
//       400: response400
//
func routeImplPublish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	buf, _ := ioutil.ReadAll(r.Body)
	err := karPublisher.Publish(ps.ByName("topic"), buf)
	if err != nil {
		http.Error(w, fmt.Sprintf("publish error: %v", err), http.StatusBadRequest)
	} else {
		fmt.Fprint(w, "OK")
	}
}

// swagger:route PUT /v1/event/{topic} events idTopicCreate
//
// topic
//
// ### Creates or updates a given topic
//
// Parameters are specified in the body of the post, as stringified JSON.
// No body passed causes a default creation.
//
//     Schemes: http
//     Consumes:
//     - application/json
//     Responses:
//       201: response201
//       204: response204
//       500: response500
//
func routeImplCreateTopic(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	topic := ps.ByName("topic")
	params := ReadAll(r)
	err := rpc.CreateTopic(&config.KafkaConfig, topic, params)
	if err != nil {
		if e, ok := err.(*sarama.TopicError); ok && e.Err == sarama.ErrTopicAlreadyExists {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Already existed")
		} else {
			http.Error(w, fmt.Sprintf("Failed to create topic %v: %v", topic, err), http.StatusInternalServerError)
		}
	} else {
		w.Header().Set("Location", fmt.Sprintf("/kar/v1/event/%v", topic))
		w.WriteHeader(http.StatusCreated)
	}
}

// swagger:route DELETE /v1/event/{topic} events idTopicDelete
//
// topic
//
// ### Deletes given topic
//
// Deletes topic specified in route.
//
//     Schemes: http
//     Consumes:
//     - application/json
//     Responses:
//       200: response200
//       500: response500
//
func routeImplDeleteTopic(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := rpc.DeleteTopic(&config.KafkaConfig, ps.ByName("topic"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete topic %v: %v", ps.ByName("topic"), err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}
