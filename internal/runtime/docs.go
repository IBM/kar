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
//     Schemes: https,http
//     BasePath: /kar
//     Version: 1.0.0
//     Schemes: http, https
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//
// swagger:meta
package runtime

// swagger:parameters idCancelReminder
// swagger:parameters idGetReminder
type reminderFilterParamWrapper struct {
	// The request body is an optional filter
	// used to select a subset of an actor's reminders.
	// in:body
	Body reminderFilter
}

// swagger:parameters idScheduleReminder
type remninderScheduleParamWrapper struct {
	// The request body describes the reminder to be scheduled
	// in:body
	Body scheduleReminderPayload
}
