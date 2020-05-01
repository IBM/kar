// This file contains structs and comments that are only
// used to generate the swagger documentation for the KAR.
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
// preserve it when updating comments in this file and in kar.go.

// Package classification KAR
//
// This document describes the RESTful API provided by the
// Kubernetes Application Runtime (KAR). It consists of
// four logical sets of sub-APIs:
// + **Actors**: APIs to invoke actor methods, access actor state, and schedule reminders.
// + **Events**: APIs to subscribe and unsubscribe from event sources and to publish to event sinks
// + **Services**: APIs to invoke service endpoints
// + **System**: APIs for controlling the KAR runtime mesh
//
// All operations are scoped to a single instance of an application.
//
//     Schemes: http
//     BasePath: /kar/v1
//     Version: v1
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//
// swagger:meta
package runtime

import "time"

/*******************************************************************
 * Request parameter and body documentation
 *******************************************************************/

// swagger:parameters idActorCall
// swagger:parameters idActorMigrate
// swagger:parameters idActorReminderGet
// swagger:parameters idActorReminderGetAll
// swagger:parameters idActorReminderSchedule
// swagger:parameters idActorReminderCancel
// swagger:parameters idActorReminderCancelAll
// swagger:parameters idActorStateDelete
// swagger:parameters idActorStateGet
// swagger:parameters idActorStateSet
// swagger:parameters idActorStateGetAll
// swagger:parameters idActorStateDeleteAll
// swagger:parameters idActorTell
type actorParam struct {
	// The actor type
	// in:path
	ActorType string `json:"actorType"`
	// The actor instance id
	// in:path
	ActorID string `json:"actorId"`
}

// swagger:parameters idServiceCall
// swagger:parameters idServiceTell
type serviceParam struct {
	// The service name
	// in:path
	Service string `json:"service"`
}

// swagger:parameters idEventPublish
// swagger:parameters idEventSubscribe
// swagger:parameters idEventUnsubscribe
type topicParam struct {
	// The topic name
	// in:path
	Topic string `json:"topic"`
}

// swagger:parameters idActorCall
// swagger:parameters idActorTell
// swagger:parameters idServiceCall
// swagger:parameters idServiceTell
// swagger:parameters idEventSubscribe
// swagger:parameters idSystemBroadcast
type pathParam struct {
	// The target endpoint to be invoked by the operation
	// in:path
	// Example: an/arbitrary/valid/pathSegment
	Path string `json:"path"`
}

// swagger:parameters idActorCall
type sessionParam struct {
	// Optionally specific the session to use when performing the call.  Enables re-entrancy for nested actor calls.
	// in:query
	// required:false
	// swagger:strfmt uuid
	Session string `json:"session"`
}

// swagger:parameters idActorReminderGet
// swagger:parameters idActorReminderCancel
type reminderIDParam struct {
	// The id of the specific reminder being targeted
	// in:path
	ReminderID string `json:"reminderId"`
}

// swagger:parameters idActorReminderSchedule
type reminderScheduleParamWrapper struct {
	// The request body describes the reminder to be scheduled
	// in:body
	Body scheduleReminderPayload
}

// swagger:parameters idActorCall
// swagger:parameters idActorTell
// swagger:parameters idServiceCall
// swagger:parameters idServiceTell
// swagger:parameters idSystemBroadcast
type endpointRequestBody struct {
	// An arbitrary JSON value to be passed through unchanged to the target endpoint
	// in:body
	TargetRequestBody interface{}
}

// swagger:parameters idActorStateGet
type actorStateGetParamWrapper struct {
	// Replace a REST-style `404` response with a `200` and nil response body when the requested key is not found.
	// in:query
	// required: false
	ErrorOnAbsent bool `json:"nilOnAbsent"`
}

type cloudeventWrapper struct {
	// An event identifier
	// required:true
	ID string `json:"id"`
	// A URI identifying the event source
	// required:true
	// swagger:strfmt uri
	Source string `json:"source"`
	// The version of the CloudEvent spec being used.
	// required:true
	// example: 1.0
	SpecVersion string `json:"specversion"`
	// The type of the event
	// required:true
	// example: com.github.pull.create
	Type string `json:"type"`
	// RFC-2046 encoding of data type
	// required:false
	// example: application/json
	DataContentType string `json:"datacontenttype"`
	// URI identifying the schema that `data` adheres to
	// required: false
	// swagger:strfmt uri
	DataSchema string `json:"dataschema"`
	// Describes the subject of the event in the context of the event producer
	// required: false
	Subject string `json:"subject"`
	// Time when the event occurred
	// required:false
	Time time.Time `json:"time"`
	// The event payload
	Data interface{} `json:"data"`
}

// swagger:parameters idEventPublish
type eventPublishRequestWrapper struct {
	// A JSON value conforming to the CloudEvent specification
	// in:body
	Event cloudeventWrapper
}

// swagger:parameters idEventSubscribe
type eventSubscribeRequestWrapper struct {
	// in:body
	Body eventSubscribeRequestBody
}
type eventSubscribeRequestBody struct {
	// A optional unique id to use for this subscrition.
	// If not id is provided, the `topic` will be used as the id.
	// required:false
	ID string `json:"id"`
	// The subscribing actor type
	// required:false
	ActorType string `json:"actorType"`
	// The subscribing actor instance id
	// required:false
	ActorID string `json:"actorId"`
	// The subscribing service name
	// required:false
	Service string `json:"service"`
	// The target endpoint to which events will be delivered
	// Example: an/arbitrary/valid/pathSegment
	// required:true
	Path string `json:"path"`
	// Should the subscription start with the oldest available event or
	// only include events published after the subscription operation?
	// required:false
	Oldest bool `json:"oldest"`
}

// swagger:parameters idEventUnsubscribe
type eventUnsubscribeRequestWrapper struct {
	// in:body
	Body eventUnsubscribeRequestBody
}
type eventUnsubscribeRequestBody struct {
	// The id of the subscription to be removed.
	// If not id is provided, the `topic` will be used as the id.
	// required: false
	ID string `json:"id"`
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

// swagger:response response200ReminderCancelResult
type response200ReminderCancelResult struct {
	// The number of reminders that were actually cancelled
	// Example: 3
	NumberCancelled int
}

// swagger:response response200ReminderGetResult
type response200ReminderGetResult struct {
	// An array containing all matching reminders
	// Example: [{ Actor: { Type: 'Foo', ID: '22' }, id: 'ticker', path: '/echo', deadline: '2020-04-14T14:17:51.073Z', period: 5000000000, encodedData: '{"msg":"hello"}' }, { Actor: { Type: 'Foo', ID: '22' }, id: 'once', path: '/echo', deadline: '2020-04-14T14:20:00Z', encodedData: '{"msg":"carpe diem"}' }]
	Body []Reminder
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
