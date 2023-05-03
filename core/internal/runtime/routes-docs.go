//
// Copyright IBM Corporation 2020,2023
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

// This file contains structs and comments that are only
// used to generate the swagger documentation for KAR.
//
// As much as possible, we keep the swagger: comments with
// the main code they are documenting, but go-swagger uses
// a collection of additional structs to generate the documentation
// of request parameters and request/response bodies for endpoint IDs.
// Since these structs are not otherwise used by the KAR runtime,
// we define them as non-exported types in this file.
//
// For documentation of the comment format for go-swagger
// see https://goswagger.io/use/spec.html
//
// Whitespace between comment blocks is
// semantically significant for go-swagger.  Be careful to
// preserve it when updating comments in this file and in kar-api-*.go.

// Package classification KAR
//
// This document describes the RESTful APIs provided or assumed by the
// Kubernetes Application Runtime (KAR). It consists of two sets of APIs:
// + The **sidecar** APIs are implemented by the KAR sidecar.
// + The **application component** APIs are meant to be implemented by application components.
//
// The **sidecar** APIs are divided into logical groups:
// + **Actors**: APIs to invoke actor methods.
// + **Services**: APIs to invoke service endpoints.
// + **Callbacks**: APIs to await the response to an asynchronous actor or service invocation.
// + **Events**: APIs to publish to event sinks or subscribe actors to event sources.
// + **Reminders**: APIs to schedule future actor invocations.
// + **State**: APIs to manage the persistent state of actors.
// + **System**: APIs for controlling the KAR runtime mesh.
//
// The **application component** APIs are divided into logical groups:
// + **Actor runtime**: APIs invoked by the sidecar to manage actor instances
// hosted by the application component.
//
//     Schemes: http
//     BasePath: /kar
//     Version: v1
//     Extensions:
//     x-tagGroups:
//     - name: sidecar
//       tags:
//       - actors
//       - callbacks
//       - events
//       - reminders
//       - services
//       - state
//       - system
//     - name: application component
//       tags:
//       - actor-runtime
//
// swagger:meta
package runtime

/*******************************************************************
 * Swagger specification for language-level actor runtime implementation
 *******************************************************************/

// swagger:route HEAD /impl/v1/actor/{actorType} actor-runtime idImplActorTypeGet
//
// actor type validation
//
// ### Test to see if the actor type is provided by the targeted runtime.
//
// Used to validate that the language level actor runtime knows how
// to instantiate an actor type.
//
//     Schemes: http
//     Responses:
//       200: response200
//       404: response404
//       500: response500
//
func dummy0() {}

// swagger:route GET /impl/v1/actor/{actorType}/{actorId} actor-runtime idImplActorGet
//
// actor allocation
//
// ### Allocate the language-level state for the specified actor instance
//
// Causes the language-level actor runtime to allocate storage for the
// actor instance and to invoke its initialization/activation method
// if one is provided.  This operation must be successfully completed
// before any POST operations on this actor instance are performed.
//
//     Schemes: http
//     Responses:
//       200: response200
//       404: response404
//       500: response500
//
func dummy1() {}

// swagger:route DELETE /impl/v1/actor/{actorType}/{actorId} actor-runtime idImplActorDelete
//
// actor deallocation
//
// ### Deallocate the language-level state for the specified actor instance
//
// The optional passivation/deactivation method for the actor type will
// be invoked.  After it completes, the language-level storate for the actor
// instance will be released. After this operation is invoked, no more POST
// operations on this actor instance may be performed unless a GET is first
// performed to re-initialize the language-level state of the actor instance.
//
//     Schemes: http
//     Responses:
//       200: response200
//       404: response404
//       500: response500
//
func dummy2() {}

// swagger:route POST /impl/v1/actor/{actorType}/{actorId}/{methodName} actor-runtime idImplActorPost
//
// actor invocation
//
// ### Invoke an actor method of the specified actor instance
//
// Invokes the actor method on the actor instance within the session specified in the path.
// The body of the request will contain the actual paramters on which to invoke the method.
//
//     Schemes: http
//     Consumes:
//     - application/kar+json
//     Produces:
//     - application/kar+json
//     Responses:
//       200: response200CallActorResult
//       202: response202
//       204: response204ActorNoContentResult
//       404: response404
//       500: response500
//
func dummy3() {}

/*******************************************************************
 * Request parameter and body documentation
 *******************************************************************/

// swagger:parameters idActorCall
// swagger:parameters idActorReminderGet
// swagger:parameters idActorReminderGetAll
// swagger:parameters idActorReminderSchedule
// swagger:parameters idActorReminderCancel
// swagger:parameters idActorReminderCancelAll
// swagger:parameters idActorSubscribe
// swagger:parameters idActorSubscriptionGet
// swagger:parameters idActorSubscriptionGetAll
// swagger:parameters idActorSubscriptionSchedule
// swagger:parameters idActorSubscriptionCancel
// swagger:parameters idActorSubscriptionCancelAll
// swagger:parameters idActorStateDelete
// swagger:parameters idActorStateExists
// swagger:parameters idActorStateGet
// swagger:parameters idActorStateSet
// swagger:parameters idActorStateUpdate
// swagger:parameters idActorStateSubmapOps
// swagger:parameters idActorStateSubkeyDelete
// swagger:parameters idActorStateSubkeyExists
// swagger:parameters idActorStateSubkeyGet
// swagger:parameters idActorStateSubkeySet
// swagger:parameters idActorStateGetAll
// swagger:parameters idActorStateDeleteAll
// swagger:parameters idImplActorGet
// swagger:parameters idImplActorDelete
// swagger:parameters idImplActorPost
type actorParam struct {
	// The actor type
	// in:path
	ActorType string `json:"actorType"`
	// The actor instance id
	// in:path
	ActorID string `json:"actorId"`
}

// swagger:parameters idImplActorTypeGet
type actorTypeOnlyParam struct {
	// The actor type
	// in:path
	ActorType string `json:"actorType"`
}

// swagger:parameters idActorStateDelete
// swagger:parameters idActorStateExists
// swagger:parameters idActorStateGet
// swagger:parameters idActorStateSet
// swagger:parameters idActorStateSubmapOps
// swagger:parameters idActorStateSubkeyDelete
// swagger:parameters idActorStateSubkeyExists
// swagger:parameters idActorStateSubkeyGet
// swagger:parameters idActorStateSubkeySet
type keyParam struct {
	// The key
	// in:path
	Key string `json:"key"`
}

// swagger:parameters idActorStateSubkeyDelete
// swagger:parameters idActorStateSubkeyExists
// swagger:parameters idActorStateSubkeyGet
// swagger:parameters idActorStateSubkeySet
type subkeyParam struct {
	// The subkey
	// in:path
	Subkey string `json:"subkey"`
}

// swagger:parameters idServiceDelete
// swagger:parameters idServiceGet
// swagger:parameters idServiceHead
// swagger:parameters idServiceOptions
// swagger:parameters idServicePatch
// swagger:parameters idServicePost
// swagger:parameters idServicePut
type serviceParam struct {
	// The service name
	// in:path
	Service string `json:"service"`
}

// swagger:parameters idActorCall
// swagger:parameters idServiceDelete
// swagger:parameters idServiceGet
// swagger:parameters idServiceHead
// swagger:parameters idServiceOptions
// swagger:parameters idServicePatch
// swagger:parameters idServicePost
// swagger:parameters idServicePut
type asyncParam struct {
	// Optionally specify the `async` pragma to make a non-blocking call.
	// Optionally specify the `promise` pragma to make a non-blocking call and
	// obtain a request id to query later.
	// in:header
	// required:false
	Pragma string `json:"Pragma"`
}

// swagger:parameters idEventPublish
type topicParam struct {
	// The topic name
	// in:path
	Topic string `json:"topic"`
}

// swagger:parameters idServiceDelete
// swagger:parameters idServiceGet
// swagger:parameters idServiceHead
// swagger:parameters idServiceOptions
// swagger:parameters idServicePatch
// swagger:parameters idServicePost
// swagger:parameters idServicePut
type pathParam struct {
	// The target endpoint to be invoked by the operation
	// in:path
	// Example: an/arbitrary/valid/pathSegment
	Path string `json:"path"`
}

// swagger:parameters idActorCall
// swagger:parameters idImplActorPost
type methodParam struct {
	// The actor method to be invoked
	// in:path
	// Example: computeMyResult
	MethodName string `json:"methodName"`
}

// swagger:parameters idActorCall
// swagger:parameters idImplActorGet
// swagger:parameters idImplActorPost
type sessionParam struct {
	// The session is an opaque string used by the KAR runtime to enable reentrancy
	// for nested actor calls and to track caller-callee relationships for failure recovery
	// in:query
	// required:false
	Session string `json:"session"`
}

// swagger:parameters idActorReminderSchedule
// swagger:parameters idActorReminderGet
// swagger:parameters idActorReminderCancel
type reminderIDParam struct {
	// The id of the specific reminder being targeted
	// in:path
	ReminderID string `json:"reminderId"`
}

// swagger:parameters idActorStateUpdate
type updateParamWrapper struct {
	// The request body describes the multi-element update operation to be performed
	// in:body
	Body stateUpdateOp
}

// swagger:parameters idActorStateSubmapOps
type mapopsParamWrapper struct {
	// The request body describes the submap operation to be performed
	// in:body
	Body submapOp
}

// swagger:parameters idActorReminderSchedule
type reminderScheduleParamWrapper struct {
	// The request body describes the reminder to be scheduled
	// in:body
	Body scheduleReminderPayload
}

// swagger:parameters idActorSubscribe
// swagger:parameters idActorSubscriptionGet
// swagger:parameters idActorSubscriptionCancel
type subscriptionIDParam struct {
	// The id of the specific subscription being targeted
	// in:path
	SubscriptionID string `json:"subscriptionID"`
}

// swagger:parameters idActorSubscribe
type subscriptionParamWrapper struct {
	// The request body describes the subscription
	// in:body
	Body EventSubscribeOptions
}

// swagger:parameters idTopicCreate
type topicCreateParamWrapper struct {
	// The request body describes the topic to be created
	// in:body
	Body topicCreateOptions
}

// swagger:parameters idAwait
type awaitParameter struct {
	// The request id
	// in:body
	Body string
}

// swagger:parameters idServiceOptions
// swagger:parameters idServicePatch
// swagger:parameters idServicePost
// swagger:parameters idServicePut
type endpointRequestBody struct {
	// An arbitrary request body to be passed through unchanged to the target endpoint
	// in:body
	TargetRequestBody interface{}
}

// swagger:parameters idActorCall
// swagger:parameters idImplActorPost
type actorCallRequestBody struct {
	// A possibly empty array containing the arguments with which to invoke the target actor method.
	// example: [3, 'hello', { msg: 'Greetings' }]
	// in:body
	ActorMethodArguments []interface{}
}

// swagger:parameters idActorStateGet
// swagger:parameters idActorReminderCancel
// swagger:parameters idActorReminderGet
// swagger:parameters idActorSubscriptionCancel
// swagger:parameters idActorSubscriptionGet
type actorStateGetParamWrapper struct {
	// Replace a REST-style `404` response with a `200` and nil response body when the requested key is not found.
	// in:query
	// required: false
	ErrorOnAbsent bool `json:"nilOnAbsent"`
}

// swagger:parameters idEventPublish
type eventPublishRequestBody struct {
	// An arbitrary request body to publish unchanged to the topic
	// in:body
	Event interface{}
}

// swagger:parameters idActorStateSetMultiple
type actorStateSetMultipleWrapper struct {
	// A map containing the state updates to perform
	// in:body
	Body map[string]interface{}
}

/*******************************************************************
 * Response documentation
 *******************************************************************/

// A success message.
// swagger:response response200
type success200 struct {
	// A success message
	// Example: OK
	Body string `json:"body"`
}

// The response body returned by the invoked endpoint
// swagger:response response200CallResult
type response200CallResult struct {
	// The response body returned by the invoked endpoint
	Body interface{} `json:"body"`
}

// The result of invoking the actor method
// swagger:response response200CallActorResult
type response200CallActorResult struct {
	// The result returned by the actor method
	Body actorCallResult
}

// The result of performing an update operation on an actor's state
// swagger:response response200StateUpdate
type response200StateUpdateOp struct {
	// The number of entries removed by the operation
	Removed int `json:"removed"`
	// The number of entires added by the operation
	Added int `json:"added"`
}

// The result of performing an operation on an submap of actor's state
// swagger:response response200StateSubmapOps
type response200StateSubmapOps struct {
	// The result of performing the supmap operation
	Body interface{} `json:"body"`
}

// swagger:response response200ReminderCancelResult
type response200ReminderCancelResult struct {
	// Returns 1 if a reminder was cancelled, 0 if not found and `nilOnError` was true
	NumberCancelled int
}

// swagger:response response200ReminderCancelAllResult
type response200ReminderCancelAllResult struct {
	// The number of reminders that were actually cancelled
	// Example: 3
	NumberCancelled int
}

// swagger:response response200ReminderGetResult
type response200ReminderGetResult struct {
	// The reminder
	// Example: { Actor: { Type: 'Foo', ID: '22' }, id: 'ticker', path: '/echo', targetTime: '2020-04-14T14:17:51.073Z', period: 5000000000, encodedData: '{"msg":"hello"}' }
	Body Reminder
}

// swagger:response response200ReminderGetAllResult
type response200ReminderGetAllResult struct {
	// An array containing all matching reminders
	// Example: [{ Actor: { Type: 'Foo', ID: '22' }, id: 'ticker', path: '/echo', targetTime: '2020-04-14T14:17:51.073Z', period: 5000000000, encodedData: '{"msg":"hello"}' }, { Actor: { Type: 'Foo', ID: '22' }, id: 'once', path: '/echo', targetTime: '2020-04-14T14:20:00Z', encodedData: '{"msg":"carpe diem"}' }]
	Body []Reminder
}

// swagger:response response200SubscriptionCancelResult
type response200SubscriptionCancelResult struct {
	// Returns 1 if a subscription was cancelled, 0 if not found and `nilOnError` was true
	NumberCancelled int
}

// swagger:response response200SubscriptionCancelAllResult
type response200SubscriptionCancelAllResult struct {
	// The number of subscriptions that were actually cancelled
	// Example: 3
	NumberCancelled int
}

// swagger:response response200SubscriptionGetResult
type response200SubscriptionGetResult struct {
	// The subscription
	Body source
}

// swagger:response response200SubscriptionGetAllResult
type response200SubscriptionGetAllResult struct {
	// An array containing all matching subscriptions
	Body []source
}

// swagger:response response200StateExistsResult
type response200StateExistsResult struct {
}

// swagger:response response200StateGetResult
type response200StateGetResult struct {
	// The requested value
	Response interface{}
}

// swagger:response response200StateGetAllResult
type response200StateGetAllResult struct {
	// A map containing the requested state
	Response map[string]interface{}
}

// swagger:response response200StateDeleteResult
type response200StateDeleteResult struct {
	// The number of key-value pairs that were deleted
	// Example: 3
	// Example: 0
	NumberDeleted int
}

// swagger:response response200StateSetResult
type response200StateSetResult struct {
	// Returns 0 if an existing entry was updated and 1 if a new entry was created
	NumberCreated int
}

// swagger:response response200StateSetMultipleResult
type response200StateSetMultipleResult struct {
	// Returns the number of new entries created by the operation
	NumberCreated int
}

// swagger:response response200SystemInfoResult
type response200SystemInfoResult struct {
	// Returns information about a system component
	ComponentInfo interface{}
}

// swagger:response response201
type success201 struct {
}

// An asynchronous operation has been accepted for eventual execution
// swagger:response response202
type success202 struct {
}

// swagger:response response204
type success204 struct {
}

// Indicates that the result of an actor method is void or undefined.
// swagger:response response204ActorNoContentResult
type response204ActorNoContentResult struct {
}

// Response indicating a bad request
// swagger:response response400
type error400 struct {
	// A message describing the problem with the request
	Body string `json:"body"`
}

// Response indicating requested resource is not found
// swagger:response response404
type error404 struct {
	// Requested resource is not found
	// Example: Not Found
	Body string `json:"body"`
}

// A message describing the error
// swagger:response response500
type error500 struct {
	// A message describing the error
	// Example: Internal Server Error
	Body string `json:"body"`
}

// A message describing the error
// swagger:response response503
type error503 struct {
	// A message describing the error
	// Example: Service Unavailable
	Body string `json:"body"`
}

// An error response returned by the invoked endpoint
// swagger:response responseGenericEndpointError
type responseGenericEndpointError struct {
	// The result body returned by the invoked endpoint
	Body interface{}
}
