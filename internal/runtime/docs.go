// This file contains structs and comments that are only
// used to generate the swagger documentation for the KAR.
//
// As much as possible, we keep the swagger: comments with
// the main code they are documenting, but go-swagger does
// require some additional structs to connect endpoint IDs to
// their request and response bodies that are not otherwise
// needed.  We stick those structs in here to make it clear
// they are not really used by KAR at runtime.

// Package classification KAR
//
// This document describes the RESTful API provided by the
// Kubernetes Application Runtime (KAR) runtime to application
// processes.
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

// swagger:parameters idActorCall
// swagger:parameters idActorMigrate
// swagger:parameters idActorReminderGet
// swagger:parameters idActorReminderSchedule
// swagger:parameters idActorReminderCancel
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
	Path string `json:"path"`
}

// swagger:parameters idActorCall
type sessionParam struct {
	// Optionally specific the session to use when performing the call.  Enables re-entrancy for nested actor calls.
	// in:query
	// required: false
	Session string `json:"session"`
}

// swagger:parameters idActorReminderCancel
// swagger:parameters idActorReminderGet
type reminderFilterParamWrapper struct {
	// The request body is an optional filter
	// used to select a subset of an actor's reminders.
	// in:body
	Body reminderFilter
}

// swagger:parameters idActorReminderSchedule
type reminderScheduleParamWrapper struct {
	// The request body describes the reminder to be scheduled
	// in:body
	Body scheduleReminderPayload
}

// swagger:parameters idActorStateGet
type actorStateGetParamWrapper struct {
	// Controls response when key is absent; if true an absent key will result in a `404` response, otherwise a `200` response with a nil value will be returned.
	// in:query
	// required: false
	ErrorOnAbsent bool `json:"errorOnAbsent"`
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

// A success message.
// swagger:response response200
type success200 struct {
	// A success message
	// Example: OK
	Body string `json:"body"`
}

// The response returned by the invoked endpoint
// swagger:response callPath200Response
type callPath200Response struct {
	// The response returned by the invoked endpoint
	Body interface{} `json:"body"`
}

// swagger:response cancelReminder200Response
type cancelReminder200Response struct {
	// The number of reminders that were actually cancelled
	// Example: 3
	Body int
}

// swagger:response getReminder200Response
type getReminder200Response struct {
	// An array containing all matching reminders
	// Example: [{ Actor: { Type: 'Foo', ID: '22' }, id: 'ticker', path: '/echo', deadline: '2020-04-14T14:17:51.073Z', period: 5000000000, encodedData: '{"msg":"hello"}' }]
	Body []Reminder
}
