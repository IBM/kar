//go:generate swagger generate spec

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/internal/runtime"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	// termination
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
	wg            = &sync.WaitGroup{}                        // wait for kafka consumer and http server to stop processing requests
	wg9           = &sync.WaitGroup{}                        // wait for signal handler
)

func tell(w http.ResponseWriter, r *http.Request, ps httprouter.Params, direct bool) {
	var err error
	if ps.ByName("service") != "" {
		var m []byte
		m, err = json.Marshal(r.Header)
		if err != nil {
			logger.Error("failed to marshal header: %v", err)
		}
		err = runtime.TellService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r), string(m), r.Method, direct)
	} else {
		err = runtime.TellActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"), r.Method, direct)
	}
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "OK")
	}
}

func callPromise(w http.ResponseWriter, r *http.Request, ps httprouter.Params, direct bool) {
	var request string
	var err error
	if ps.ByName("service") != "" {
		var m []byte
		m, err = json.Marshal(r.Header)
		if err != nil {
			logger.Error("failed to marshal header: %v", err)
		}
		request, err = runtime.CallPromiseService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r), string(m), r.Method, direct)
	} else {
		request, err = runtime.CallPromiseActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"), r.Header.Get("Accept"), r.Method, direct)
	}
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, request)
	}
}

// swagger:route POST /v1/await callbacks idAwait
//
// await
//
// ### Await the response to an actor or service call
//
// Await blocks until the response to an asynchronous call is received and
// returns this response.
//
//     Consumes:
//     - text/plain
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//
func awaitPromise(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	reply, err := runtime.AwaitPromise(ctx, runtime.ReadAll(r))
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to await promise: %v", err), http.StatusInternalServerError)
		}
	} else {
		w.Header().Add("Content-Type", reply.ContentType)
		w.WriteHeader(reply.StatusCode)
		fmt.Fprint(w, reply.Payload)
	}
}

// swagger:route POST /v1/service/{service}/call/{path} services idServicePost
//
// call
//
// ### Perform a POST on a service endpoint
//
// Execute a `POST` operation on the `path` endpoint of `service`.
// The request body is passed through to the target endpoint.
// The result of performing a POST on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route GET /v1/service/{service}/call/{path} services idServiceGet
//
// call
//
// ### Perform a GET on a service endpoint
//
// Execute a `GET` operation on the `path` endpoint of `service`.
// The result of performing a GET on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route HEAD /v1/service/{service}/call/{path} services idServiceHead
//
// call
//
// ### Perform a HEAD on a service endpoint
//
// Execute a `HEAD` operation on the `path` endpoint of `service`.
// The result of performing a HEAD on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route PUT /v1/service/{service}/call/{path} services idServicePut
//
// call
//
// ### Perfrom a PUT on a service endpoint
//
// Execute a `PUT` operation on the `path` endpoint of `service`.
// The request body is passed through to the target endpoint.
// The result of performing a PUT on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route PATCH /v1/service/{service}/call/{path} services idServicePatch
//
// call
//
// ### Perform a PATCH on a service endpoint
//
// Execute a `PATCH` operation on the `path` endpoint of `service`.
// The request body is passed through to the target endpoint.
// The result of performing a PATCH on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route DELETE /v1/service/{service}/call/{path} services idServiceDelete
//
// call
//
// ### Perform a DELETE on a service endpoint
//
// Execute a `DELETE` operation on the `path` endpoint of `service`.
// The result of performing a DELETE on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route OPTIONS /v1/service/{service}/call/{path} services idServiceOptions
//
// call
//
// ### Perform an OPTIONS on a service endpoint
//
// Execute an `OPTIONS` operation on the `path` endpoint of `service`.
// The request body is passed through to the target endpoint.
// The result of performing an OPTIONS on the target service endpoint
// is returned unless the `async` or `promise` pragma header is specified.
//
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route POST /v1/actor/{actorType}/{actorId}/call/{path} actors idActorCall
//
// call
//
// ### Invoke an actor method
//
// Call executes a `POST` to the `path` endpoint of the
// actor instance indicated by `actorType` and `actorId`.
// The request body must be a (possibly zero-length) JSON array whose elements
// are used as the actual parameters of the actor method.
// The result of the call is the result of invoking the target actor method
// unless the `async` or `promise` pragma header is specified.
//
//     Consumes:
//     - application/kar+json
//     Produces:
//     - application/kar+json
//     Schemes: http
//     Responses:
//       200: response200CallActorResult
//       202: response202CallResult
//       404: response404
//       500: response500
//       503: response503
//
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	direct := false
	for _, pragma := range r.Header[textproto.CanonicalMIMEHeaderKey("Pragma")] {
		if strings.ToLower(pragma) == "http" {
			direct = true
			break
		}
	}
	for _, pragma := range r.Header[textproto.CanonicalMIMEHeaderKey("Pragma")] {
		if strings.ToLower(pragma) == "async" {
			tell(w, r, ps, direct)
			return
		} else if strings.ToLower(pragma) == "promise" {
			callPromise(w, r, ps, direct)
			return
		}
	}
	var reply *runtime.Reply
	var err error
	if ps.ByName("service") != "" {
		var m []byte
		m, err = json.Marshal(r.Header)
		if err != nil {
			logger.Error("failed to marshal header: %v", err)
		}
		reply, err = runtime.CallService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r), string(m), r.Method, direct)
	} else {
		session := r.FormValue("session")
		reply, err = runtime.CallActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"), r.Header.Get("Accept"), r.Method, session, direct)
	}
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

// swagger:route POST /v1/actor/{actorType}/{actorId}/reminders/{reminderId} reminders idActorReminderSchedule
//
// reminders
//
// ### Schedule a reminder
//
// This operation schedules a reminder for the actor instance specified in the path
// as described by the data provided in the request body.
// If there is already a reminder for the target actor instance with the same reminderId,
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
func reminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var action string
	body := ""
	noa := "false"
	switch r.Method {
	case "GET":
		action = "get"
		noa = r.FormValue("nilOnAbsent")
	case "POST":
		action = "set"
		body = runtime.ReadAll(r)
	case "DELETE":
		action = "del"
		noa = r.FormValue("nilOnAbsent")
	default:
		http.Error(w, fmt.Sprintf("Unsupported method %v", r.Method), http.StatusMethodNotAllowed)
		return
	}
	reply, err := runtime.Bindings(ctx, "reminders", runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("reminderId"), noa, action, body, r.Header.Get("Content-Type"), r.Header.Get("Accept"))
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

// swagger:route POST /v1/actor/{actorType}/{actorId}/events/{subscriptionId} events idActorSubscribe
//
// subscriptions
//
// ### Subscribe to a topic
//
// This operation subscribes an actor instance to a topic.
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
//       200: response200
//       500: response500
//       503: response503
//
func subscription(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var action string
	body := ""
	noa := "false"
	switch r.Method {
	case "GET":
		action = "get"
		noa = r.FormValue("nilOnAbsent")
	case "POST":
		// FIXME: https://github.ibm.com/solsa/kar/issues/31
		//        Should return a 404 if topic doesn't exist.
		action = "set"
		body = runtime.ReadAll(r)
	case "DELETE":
		action = "del"
		noa = r.FormValue("nilOnAbsent")
	default:
		http.Error(w, fmt.Sprintf("Unsupported method %v", r.Method), http.StatusMethodNotAllowed)
		return
	}
	reply, err := runtime.Bindings(ctx, "subscriptions", runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("subscriptionId"), noa, action, body, r.Header.Get("Content-Type"), r.Header.Get("Accept"))
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

func stateKey(t, id string) string {
	return "main" + config.Separator + "state" + config.Separator + t + config.Separator + id
}

func flatEntryKey(key string) string {
	return key + config.Separator + config.Separator
}

func nestedEntryKey(key string, subkey string) string {
	return key + config.Separator + subkey
}

// swagger:route PUT /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateSet
//
// state/key
//
// ### Update a single entry of an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by setting `key` to contain the JSON request body.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if a new entry was created and `0` if an existing entry was updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200StateSetResult
//       500: response500
//

// swagger:route PUT /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeySet
//
// state/key/subkey
//
// ### Update a single entry of a sub-map of an actor's state
//
// The map state of the actor instance indicated by `actorType` and `actorId`
// will be updated by setting `key`/`subkey` to contain the JSON request body.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if a new entry was created and `0` if an existing entry was updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200StateSetResult
//       500: response500
//
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}

	if reply, err := store.HSet(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey, runtime.ReadAll(r)); err != nil {
		http.Error(w, fmt.Sprintf("HSET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateGet
//
// state/key
//
// ### Get a single entry of an actor's state
//
// The `key` entry of the state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
// If there is no entry for `key` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`,
// in which case a `200` reponse with a `nil` response body will be returned.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetResult
//       404: response404
//       500: response500
//

// swagger:route GET /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeyGet
//
// state/key/subkey
//
// ### Get a single entry of an actor's state
//
// The `key/subkey` entry of the state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
// If there is no entry for  `key/subkey` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`,
// in which case a `200` reponse with a `nil` response body will be returned.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetResult
//       404: response404
//       500: response500
//
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}

	if reply, err := store.HGet(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey); err == store.ErrNil {
		if noa := r.FormValue("nilOnAbsent"); noa == "true" {
			fmt.Fprint(w, reply)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("HGET failed: %v", err), http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, reply)
	}
}

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateDelete
//
// state/key
//
// ### Remove a single entry in an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by removing the entry for `key`.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if an entry was actually removed and
// `0` if there was no entry for `key`.
//
//     Schemes: http
//     Produces:
//     - text/plain
//     Responses:
//       200: response200StateDeleteResult
//       500: response500
//

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeyDelete
//
// state/key/subkey
//
// ### Remove a single entry in an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`, and `key`
// will be updated by removing the entry for `key/subkey`.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if an entry was actually removed and
// `0` if there was no entry for `key`.
//
//     Schemes: http
//     Produces:
//     - text/plain
//     Responses:
//       200: response200StateDeleteResult
//       500: response500
//
func del(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}
	if reply, err := store.HDel(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey); err != nil {
		http.Error(w, fmt.Sprintf("HDEL failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /v1/actor/{actorType}/{actorId}/state state idActorStateGetAll
//
// state
//
// ### Get an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetAllResult
//       500: response500
//
func getAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HGetAll(stateKey(ps.ByName("type"), ps.ByName("id"))); err != nil {
		http.Error(w, fmt.Sprintf("HGETALL failed: %v", err), http.StatusInternalServerError)
	} else {
		// reply has type map[string]string
		// we unmarshal the values then marshal the map
		m := map[string]interface{}{}
		for i, s := range reply {
			var v interface{}
			json.Unmarshal([]byte(s), &v)
			splitKeys := strings.SplitN(i, config.Separator, 2)
			key := splitKeys[0]
			subkey := splitKeys[1]
			if subkey == config.Separator {
				m[key] = v
			} else {
				// FIXME: This is where we have to play with submaps
				logger.Error("subkey get all not implemented; dropping value!!!")
			}

		}
		b, _ := json.Marshal(m)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, string(b))
	}
}

// swagger:route POST /v1/actor/{actorType}/{actorId}/state state idActorStateSetMultiple
//
// state
//
// ### Update multiple entries of an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by atomically updated by storing all key-value pairs
// in the request body.
// The operation will not return until the state has been updated.
// The result of the operation is the number of new entires that were created.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200StateSetMultipleResult
//       400: response400
//       500: response500
//
func setMultiple(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var updates map[string]interface{}
	if err := json.Unmarshal([]byte(runtime.ReadAll(r)), &updates); err != nil {
		http.Error(w, "Request body was not a map[string, interface{}]", http.StatusBadRequest)
		return
	}
	sk := stateKey(ps.ByName("type"), ps.ByName("id"))
	m := map[string]string{}
	for i, v := range updates {
		s, err := json.Marshal(v)
		if err != nil {
			logger.Error("setMultiple: %v[%v] = %v failed due to %v", sk, i, v, err)
			http.Error(w, fmt.Sprintf("Unable to re-serialize value %v", v), http.StatusInternalServerError)
			return
		}
		m[flatEntryKey(i)] = string(s)
	}
	if reply, err := store.HSetMultiple(sk, m); err != nil {
		http.Error(w, fmt.Sprintf("HSET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/state state idActorStateDeleteAll
//
// state
//
// ### Remove an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be deleted.
//
//     Schemes: http
//     Responses:
//       200: response200StateDeleteResult
//       404: response404
//       500: response500
//
func delAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Del(stateKey(ps.ByName("type"), ps.ByName("id"))); err == store.ErrNil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("DEL failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route POST /v1/system/shutdown system idSystemShutdown
//
// shutdown
//
// ### Shutdown a single KAR runtime
//
// Initiate an orderly shutdown of the target KAR runtime process.
//
//     Schemes: http
//     Responses:
//       200: response200
//
func shutdown(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
	logger.Info("Invoking cancel() in response to shutdown request")
	cancel()
}

// swagger:route GET /v1/system/health system isSystemHealth
//
// health
//
// ### Health-check endpoint
//
// Returns a `200` response to indicate that the KAR runtime processes is healthy.
//
//     Schemes: http
//     Responses:
//       200: response200
//
func health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
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
//     - application
//     Responses:
//       200: response200
//       500: response500
//
func publish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// FIXME: https://github.ibm.com/solsa/kar/issues/30
	//        Should return a 404 if topic doesn't exist.
	buf, _ := ioutil.ReadAll(r.Body)
	err := pubsub.Publish(ps.ByName("topic"), buf)
	if err != nil {
		http.Error(w, fmt.Sprintf("publish error: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

// post handles a direct http request from a peer sidecar
// TODO swagger
func post(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	value, _ := ioutil.ReadAll(r.Body)
	m := pubsub.Message{Value: value}
	process(m)
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, "OK")
}

// swagger:route POST /v1/event/{topic} events idTopicCreate
//
// createTopic
//
// ### Creates given topic
//
// Parameters are specified in the body of the post, as stringified JSON.
// No body passed causes a default creation.
//
//     Schemes: http
//     Consumes:
//     - application
//     Responses:
//       200: response200
//       500: response500
//
func createTopic(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	params := runtime.ReadAll(r)
	err := pubsub.CreateTopic(ps.ByName("topic"), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create topic %v: %v", ps.ByName("topic"), err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

// swagger:route DELETE /v1/event/{topic} events idTopicDelete
//
// deleteTopic
//
// ### Deletes given topic
//
// Deletes kafka topic specified in route.
//
//     Schemes: http
//     Consumes:
//     - application
//     Responses:
//       200: response200
//       500: response500
//
func deleteTopic(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := pubsub.DeleteTopic(ps.ByName("topic"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete topic %v: %v", ps.ByName("topic"), err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

func getSidecars(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	format := "text/plain"
	if r.Header.Get("Accept") == "application/json" {
		format = "application/json"
	}
	data, err := pubsub.GetSidecars(format)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to acquire information: %v", err), http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", format)
		fmt.Fprint(w, data)
	}
}

// server implements the HTTP server
func server(listener net.Listener) http.Server {
	base := "/kar/v1"
	router := httprouter.New()
	methods := [7]string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

	// service invocation - handles all common HTTP requests
	for _, method := range methods {
		router.Handle(method, base+"/service/:service/call/*path", call)
	}

	// callbacks
	router.POST(base+"/await", awaitPromise)

	// actor invocation
	router.POST(base+"/actor/:type/:id/call/*path", call)

	// reminders
	router.GET(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.GET(base+"/actor/:type/:id/reminders", reminder)
	router.POST(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.DELETE(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.DELETE(base+"/actor/:type/:id/reminders", reminder)

	// events
	router.GET(base+"/actor/:type/:id/events/:subscriptionId", subscription)
	router.GET(base+"/actor/:type/:id/events", subscription)
	router.POST(base+"/actor/:type/:id/events/:subscriptionId", subscription)
	router.DELETE(base+"/actor/:type/:id/events/:subscriptionId", subscription)
	router.DELETE(base+"/actor/:type/:id/events", subscription)

	// actor state
	router.GET(base+"/actor/:type/:id/state/:key/:subkey", get)
	router.PUT(base+"/actor/:type/:id/state/:key/:subkey", set)
	router.DELETE(base+"/actor/:type/:id/state/:key/:subkey", del)
	router.GET(base+"/actor/:type/:id/state/:key", get)
	router.PUT(base+"/actor/:type/:id/state/:key", set)
	router.DELETE(base+"/actor/:type/:id/state/:key", del)
	router.GET(base+"/actor/:type/:id/state", getAll)
	router.POST(base+"/actor/:type/:id/state", setMultiple)
	router.DELETE(base+"/actor/:type/:id/state", delAll)

	// kar system methods
	router.GET(base+"/system/health", health)
	router.POST(base+"/system/shutdown", shutdown)
	router.POST(base+"/system/post", post)
	router.GET(base+"/system/information/sidecars", getSidecars)

	// events
	router.POST(base+"/event/:topic/publish", publish)
	router.POST(base+"/event/:topic/", createTopic)
	router.DELETE(base+"/event/:topic/", deleteTopic)

	return http.Server{Handler: h2c.NewHandler(router, &http2.Server{MaxConcurrentStreams: 262144})}
}

// process incoming message asynchronously
// one goroutine, incr and decr WaitGroup
func process(m pubsub.Message) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Process(ctx, cancel, m)
	}()
}

func main() {
	logger.Warning("starting...")
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	wg9.Add(1)
	go func() {
		defer wg9.Done()
		select {
		case <-signals:
			logger.Info("Invoking cancel9() from signal handler")
			cancel9()
		case <-ctx9.Done():
		}
	}()

	var listenHost string
	if config.KubernetesMode {
		listenHost = fmt.Sprintf(":%d", config.RuntimePort)
	} else {
		listenHost = fmt.Sprintf("127.0.0.1:%d", config.RuntimePort)
	}
	listener, err := net.Listen("tcp", listenHost)
	if err != nil {
		logger.Fatal("listener failed: %v", err)
	}

	if store.Dial() != nil {
		logger.Fatal("failed to connect to Redis: %v", err)
	}
	defer store.Close()

	if pubsub.Dial() != nil {
		logger.Fatal("dial failed: %v", err)
	}
	defer pubsub.Close()

	if config.Purge {
		purge("*")
		return
	} else if config.Drain {
		purge("pubsub" + config.Separator + "*")
		return
	}

	// one goroutine, defer close(closed)
	closed, err := pubsub.Join(ctx, process, listener.Addr().(*net.TCPAddr).Port)
	if err != nil {
		logger.Fatal("join failed: %v", err)
	}

	srv := server(listener)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done() // wait
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown HTTP server: %v", err)
		}
		runtime.CloseIdleConnections()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Collect(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.ProcessReminders(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.ManageBindings(ctx)
	}()

	runtimePort := fmt.Sprintf("KAR_RUNTIME_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	appPort := fmt.Sprintf("KAR_APP_PORT=%d", config.AppPort)
	requestTimeout := fmt.Sprintf("KAR_REQUEST_TIMEOUT=%d", config.RequestTimeout.Milliseconds())
	logger.Info("%s %s", runtimePort, appPort)

	args := flag.Args()

	if config.Invoke {
		exitCode = runtime.Invoke(ctx9, args)
		cancel()
	} else if config.Get != "" {
		exitCode = runtime.GetInformation(ctx9, args)
		cancel()
	} else if len(args) > 0 {
		exitCode = runtime.Run(ctx9, args, append(os.Environ(), runtimePort, appPort, requestTimeout))
		cancel()
	}

	<-closed // wait for closed consumer first since process adds to WaitGroup
	wg.Wait()

	cancel9()

	wg9.Wait()

	logger.Warning("exiting...")
}

func purge(pattern string) {
	if err := pubsub.Purge(); err != nil {
		logger.Error("failed to delete Kafka topic: %v", err)
	}
	if err := store.Purge(pattern); err != nil {
		logger.Error("failed to delete Redis keys: %v", err)
	}
}
