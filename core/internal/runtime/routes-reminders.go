package runtime

/*
 * This file contains the implementation of the portion of the
 * KAR REST API related to actor reminders.
 */

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/reminders reminders idActorReminderCancelAll
//
// reminders
//
// ### Cancel all reminders
//
// This operation cancels all reminders for the actor instance specified in the path.
// The number of reminders cancelled is returned as the result of the operation.
//
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200ReminderCancelAllResult
//       500: response500
//       503: response503
//

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/reminders/{reminderId} reminders idActorReminderCancel
//
// reminders/id
//
// ### Cancel a reminder
//
// This operation cancels the reminder for the actor instance specified in the path.
// If the reminder is successfully cancelled a `200` response with a body of `1` will be returned.
// If the reminder is not found, a `404` response will be returned unless
// the boolean query parameter `nilOnAbsent` is set to `true`. If `nilOnAbsent`
// is sent to true the `404` response will instead be a `200` with a body containing `0`.
//
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200ReminderCancelResult
//       404: response404
//       500: response500
//       503: response503
//

// swagger:route GET /v1/actor/{actorType}/{actorId}/reminders reminders idActorReminderGetAll
//
// reminders
//
// ### Get all reminders
//
// This operation returns all reminders for the actor instance specified in the path.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200ReminderGetAllResult
//       500: response500
//       503: response503
//

// swagger:route GET /v1/actor/{actorType}/{actorId}/reminders/{reminderId} reminders idActorReminderGet
//
// reminders/id
//
// ### Get a reminder
//
// This operation returns the reminder for the actor instance specified in the path.
// If there is no reminder with the id `reminderId` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`.
// If `nilOnAbsent` is true the `404` response will be replaced with
// a `200` response with a `nil` response body.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200ReminderGetResult
//       404: response404
//       500: response500
//       503: response503
//

// swagger:route PUT /v1/actor/{actorType}/{actorId}/reminders/{reminderId} reminders idActorReminderSchedule
//
// reminders/id
//
// ### Schedule a reminder
//
// Schedule the reminder for the actor instance and reminderId specified in the path
// as described by the data provided in the request body.
// If there is already a reminder for the target actor instance and reminderId,
// that existing reminder's schedule will be updated based on the request body.
// The operation will not return until after the reminder is scheduled.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200
//       500: response500
//       503: response503
//
func routeImplReminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	reply, err := Bindings(ctx, "reminders", Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("reminderId"), noa, action, body, r.Header.Get("Content-Type"), r.Header.Get("Accept"))
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
