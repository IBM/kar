package runtime

/*
 * This file contains the implementation of the portion of the
 * KAR REST API related to invoking actor methods and service endpoints.
 */

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/core/internal/pubsub"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

func tellHelper(w http.ResponseWriter, r *http.Request, ps httprouter.Params, direct bool) {
	var err error
	if ps.ByName("service") != "" {
		var m []byte
		m, err = json.Marshal(r.Header)
		if err != nil {
			logger.Error("failed to marshal header: %v", err)
		}
		err = TellService(ctx, ps.ByName("service"), ps.ByName("path"), ReadAll(r), string(m), r.Method, direct)
	} else {
		err = TellActor(ctx, Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), ReadAll(r), direct)
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
		request, err = CallPromiseService(ctx, ps.ByName("service"), ps.ByName("path"), ReadAll(r), string(m), r.Method, direct)
	} else {
		request, err = CallPromiseActor(ctx, Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), ReadAll(r), direct)
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
func routeImplAwaitPromise(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	reply, err := AwaitPromise(ctx, ReadAll(r))
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
//       202: response202
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
//       202: response202
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
//       202: response202
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
//       201: response201
//       204: response204
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
//       202: response202
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
//       202: response202
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
//       202: response202
//       404: response404
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route POST /v1/actor/{actorType}/{actorId}/call/{methodName} actors idActorCall
//
// call
//
// ### Invoke an actor method
//
// Call invokes the `methodName` method of the
// actor instance indicated by `actorType` and `actorId`.
// The request body must be a (possibly zero-length) JSON array whose elements
// are used as the actual parameters of the actor method.
// The result of the call is the result of invoking the target actor method
// unless the `async` or `promise` pragma header is specified.  If the actor
// method returns `void` or `undefined`, then a 204 - No Content reponse is returned.
//
//     Consumes:
//     - application/kar+json
//     Produces:
//     - application/kar+json
//     Schemes: http
//     Responses:
//       200: response200CallActorResult
//       202: response202
//       204: response204ActorNoContentResult
//       404: response404
//       500: response500
//       503: response503
//
func routeImplCall(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	direct := false
	for _, pragma := range r.Header[textproto.CanonicalMIMEHeaderKey("Pragma")] {
		if strings.ToLower(pragma) == "http" {
			direct = true
			break
		}
	}
	for _, pragma := range r.Header[textproto.CanonicalMIMEHeaderKey("Pragma")] {
		if strings.ToLower(pragma) == "async" {
			tellHelper(w, r, ps, direct)
			return
		} else if strings.ToLower(pragma) == "promise" {
			callPromise(w, r, ps, direct)
			return
		}
	}
	var reply *Reply
	var err error
	if ps.ByName("service") != "" {
		var m []byte
		m, err = json.Marshal(r.Header)
		if err != nil {
			logger.Error("failed to marshal header: %v", err)
		}
		reply, err = CallService(ctx, ps.ByName("service"), ps.ByName("path"), ReadAll(r), string(m), r.Method, direct)
	} else {
		session := r.FormValue("session")
		reply, err = CallActor(ctx, Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), ReadAll(r), session, direct)
	}
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else if err == pubsub.ErrRouteToActorTimeout {
			http.Error(w, fmt.Sprintf("timeout waiting for Actor type %v to be defined", ps.ByName("type")), http.StatusRequestTimeout)
		} else if err == pubsub.ErrRouteToServiceTimeout {
			http.Error(w, fmt.Sprintf("timeout waiting for Service %v to be defined", ps.ByName("type")), http.StatusRequestTimeout)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		if reply.StatusCode == http.StatusNoContent {
			w.WriteHeader(reply.StatusCode)
		} else {
			w.Header().Add("Content-Type", reply.ContentType)
			w.WriteHeader(reply.StatusCode)
			fmt.Fprint(w, reply.Payload)
		}
	}
}

// swagger:route DELETE /v1/actor/{actorType}/{actorId} actors idActorDelete
//
// actor
//
// ### Completely remove an actor instance
//
// All user-level and runtime state of the actor instance indicated by
// `actorType` and `actorId` will be asynchronously deleted.
//
//     Schemes: http
//     Responses:
//       202: response202
//       404: response404
//       500: response500
//
func routeImplDelActor(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := DeleteActor(ctx, Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, false)
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
