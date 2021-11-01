/*
 * Copyright IBM Corporation 2020,2021
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.ibm.research.kar;

import java.net.URI;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonBuilderFactory;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonValue;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.WebApplicationException;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.Reminder;
import com.ibm.research.kar.actor.Subscription;
import com.ibm.research.kar.actor.exceptions.ActorMethodInvocationException;
import com.ibm.research.kar.actor.exceptions.ActorMethodNotFoundException;
import com.ibm.research.kar.actor.exceptions.ActorMethodTimeoutException;
import com.ibm.research.kar.liberty.KarSidecar;
import com.ibm.research.kar.runtime.KarConfig;

import org.eclipse.microprofile.rest.client.RestClientBuilder;

public class Kar {
	public static final String KAR_ACTOR_JSON = "application/kar+json";
	public static final MediaType KAR_ACTOR_JSON_TYPE = new MediaType("application", "kar+json");

	private static final Logger logger = Logger.getLogger(Kar.class.getName());

	private static final KarSidecar sidecar = instantiateSidecar();

	private static final JsonBuilderFactory factory = Json.createBuilderFactory(Map.of());

	private Kar() {
	}

	private static KarSidecar instantiateSidecar() {

		String port = System.getenv("KAR_RUNTIME_PORT");
		if (port == null || port.trim().isEmpty()) {
			logger.severe("KAR_RUNTIME_PORT is not set. Fatal misconfiguration. Forcing immediate hard exit of JVM.");
			Runtime.getRuntime().halt(1);
		}

		String baseURIStr = "http://localhost:" + port + "/";
		logger.fine("KAR Sidecar base URI is " + baseURIStr);
		URI sidecarURI = URI.create(baseURIStr);
		RestClientBuilder builder = RestClientBuilder.newBuilder().baseUri(sidecarURI);
		KarSidecar sidecar = builder.readTimeout(KarConfig.SIDECAR_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.connectTimeout(KarConfig.SIDECAR_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS).build(KarSidecar.class);

		return sidecar;
	}

	private static JsonArray packArgs(JsonValue[] args) {
		JsonArrayBuilder ja = factory.createArrayBuilder();
		for (JsonValue a : args) {
			ja.add(a);
		}
		return ja.build();
	}

	private static Object toValue(Response response) {
		if (response.hasEntity()) {
			MediaType type = response.getMediaType();
			MediaType basicType = new MediaType(type.getType(), type.getSubtype());
			if (basicType.equals(MediaType.APPLICATION_JSON_TYPE) || basicType.equals(KAR_ACTOR_JSON_TYPE)) {
				return response.readEntity(JsonValue.class);
			} else if (basicType.equals(MediaType.TEXT_PLAIN_TYPE)) {
				return response.readEntity(String.class);
			} else {
				return JsonValue.NULL;
			}
		} else {
			return JsonValue.NULL;
		}
	}

	private static int toInt(Response response) {
		if (response.hasEntity()) {
			return response.readEntity(java.lang.Integer.TYPE);
		} else {
			return 0;
		}
	}

	private static String responseToString(Response response) {
		if (response.hasEntity()) {
			MediaType type = response.getMediaType();
			MediaType basicType = new MediaType(type.getType(), type.getSubtype());
			if (basicType.equals(MediaType.APPLICATION_JSON_TYPE) || basicType.equals(KAR_ACTOR_JSON_TYPE)) {
				return response.readEntity(JsonValue.class).toString();
			} else if (basicType.equals(MediaType.TEXT_PLAIN_TYPE)) {
				return response.readEntity(String.class);
			}
		}
		return null;
	}

	private static Reminder[] toReminderArray(Response response) {
		try {
			ArrayList<Reminder> res = new ArrayList<Reminder>();
			JsonArray ja = ((JsonValue) toValue(response)).asJsonArray();
			for (JsonValue jv : ja) {
				try {
					JsonObject jo = jv.asJsonObject();
					String actorType = jo.getJsonObject("Actor").getString("Type");
					String actorId = jo.getJsonObject("Actor").getString("ID");
					String id = jo.getString("id");
					String path = jo.getString("path");
					String targetTimeString = jo.getString("targetTime");
					Instant targetTime = Instant.parse(targetTimeString);
					Duration period = null;
					if (jo.get("period") != null) {
						long nanos = ((JsonNumber) jo.get("period")).longValueExact();
						period = Duration.ofNanos(nanos);
					}
					String encodedData = jo.getString("encodedData");
					Reminder r = new Reminder(Actors.ref(actorType, actorId), id, path, targetTime, period, encodedData);
					res.add(r);
				} catch (ClassCastException e) {
					logger.warning("toReminderArray: Dropping unexpected element " + jv);
				}
			}
			return res.toArray(new Reminder[res.size()]);
		} catch (ClassCastException e) {
			return new Reminder[0];
		}
	}

	private static Subscription[] toSubscriptionArray(Response response) {
		try {
			ArrayList<Subscription> res = new ArrayList<Subscription>();
			JsonArray ja = ((JsonValue) toValue(response)).asJsonArray();
			for (JsonValue jv : ja) {
				try {
					JsonObject jo = jv.asJsonObject();
					String actorType = jo.getJsonObject("Actor").getString("Type");
					String actorId = jo.getJsonObject("Actor").getString("ID");
					String id = jo.getString("id");
					String path = jo.getString("path");
					String topic = jo.getString("topic");
					Subscription s = new Subscription(Actors.ref(actorType, actorId), id, path, topic);
					res.add(s);
				} catch (ClassCastException e) {
					logger.warning("toReminderArray: Dropping unexpected element " + jv);
				}
			}
			return res.toArray(new Subscription[res.size()]);
		} catch (ClassCastException e) {
			return new Subscription[0];
		}
	}

	private static final class ActorRefImpl implements ActorRef {
		final String type;
		final String id;

		ActorRefImpl(String type, String id) {
			this.type = type;
			this.id = id;
		}

		@Override
		public String getType() {
			return type;
		}

		@Override
		public String getId() {
			return id;
		}
	}

	/******************
	 * KAR API
	 ******************/

	/**
	 * KAR API methods for Services
	 */
	public static class Services {
		/*
		 * Lower-level REST operations on a KAR Service
		 */

		/**
		 * Synchronous REST DELETE
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Response delete(String service, String path) {
			return sidecar.callDelete(service, path);
		}

		/**
		 * Asynchronous REST DELETE
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> deleteAsync(String service, String path) {
			return sidecar.callAsyncDelete(service, path);
		}

		/**
		 * Synchronous REST GET
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Response get(String service, String path) {
			return sidecar.callGet(service, path);
		}

		/**
		 * Asynchronous REST GET
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> getAsync(String service, String path) {
			return sidecar.callAsyncGet(service, path);
		}

		/**
		 * Synchronous REST HEAD
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Response head(String service, String path) {
			return sidecar.callHead(service, path);
		}

		/**
		 * Asynchronous REST HEAD
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> headAsync(String service, String path) {
			return sidecar.callAsyncHead(service, path);
		}

		/**
		 * Synchronous REST OPTIONS
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Response options(String service, String path) {
			return sidecar.callOptions(service, path, JsonValue.NULL);
		}

		/**
		 * Synchronous REST OPTIONS
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Response options(String service, String path, JsonValue body) {
			return sidecar.callOptions(service, path, body);
		}

		/**
		 * Asynchronous REST OPTIONS
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> optionsAsync(String service, String path) {
			return sidecar.callAsyncOptions(service, path, JsonValue.NULL);
		}

		/**
		 * Asynchronous REST OPTIONS
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> optionsAsync(String service, String path, JsonValue body) {
			return sidecar.callAsyncOptions(service, path, body);
		}

		/**
		 * Synchronous REST PATCH
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Response patch(String service, String path, JsonValue body) {
			return sidecar.callPatch(service, path, body);
		}

		/**
		 * Asynchronous REST PATCH
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> patchAsync(String service, String path, JsonValue body) {
			return sidecar.callAsyncPatch(service, path, body);
		}

		/**
		 * Synchronous REST POST
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Response post(String service, String path, JsonValue body) {
			return sidecar.callPost(service, path, body);
		}

		/**
		 * Asynchronous REST POST
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> postAsync(String service, String path, JsonValue body) {
			return sidecar.callAsyncPost(service, path, body);
		}

		/**
		 * Synchronous REST PUT
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Response put(String service, String path, JsonValue body) {
			return sidecar.callPut(service, path, body);
		}

		/**
		 * Asynchronous REST PUT
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static CompletionStage<Response> putAsync(String service, String path, JsonValue body) {
			return sidecar.callAsyncPut(service, path, body);
		}

		/*
		 * Higher-level Service call/tell operations that hide the REST layer
		 */

		/**
		 * Asynchronous service invocation; returns as soon as the invocation has been
		 * initiated.
		 *
		 * @param service The name of the service to invoke.
		 * @param path    The service endpoint to invoke.
		 * @param body    The request body with which to invoke the service endpoint.
		 */
		public static void tell(String service, String path, JsonValue body) {
			sidecar.tellPost(service, path, body);
		}

		/**
		 * Synchronous service invocation
		 *
		 * @param service The name of the service to invoke.
		 * @param path    The service endpoint to invoke.
		 * @param body    The request body with which to invoke the service endpoint.
		 * @return The result returned by the target service.
		 */
		public static Object call(String service, String path, JsonValue body) {
			Response resp = sidecar.callPost(service, path, body);
			return toValue(resp);
		}

		/**
		 * Aynchronous service invocation with eventual access to the result of the
		 * invocation
		 *
		 * @param service The name of the service to invoke.
		 * @param path    The service endpoint to invoke.
		 * @param body    The request body with which to invoke the service endpoint.
		 * @return A CompletionStage containing the result of invoking the target
		 *         service.
		 */
		public static CompletionStage<Object> callAsync(String service, String path, JsonValue body) {
			return sidecar.callAsyncPut(service, path, body).thenApply(response -> toValue(response));
		}
	}

	/**
	 * KAR API methods for Actors
	 */
	public static class Actors {

		/**
		 * Construct an ActorRef that represents a specific Actor instance.
		 *
		 * @param type The type of the Actor instance
		 * @param id   The instance id of the Actor instance
		 * @return An ActorRef representing the Actor instance.
		 */
		public static ActorRef ref(String type, String id) {
			return new ActorRefImpl(type, id);
		}

		/**
		 * Asynchronously remove all user-level and runtime state of an Actor.
		 *
		 * @param actor The Actor instance.
		 */
		public static void remove(ActorRef actor) {
			sidecar.actorDelete(actor.getType(), actor.getId());
		}

		/**
		 * Asynchronous actor invocation; returns as soon as the invocation has been
		 * initiated.
		 *
		 * @param actor The target actor.
		 * @param path  The actor method to invoke.
		 * @param args  The arguments with which to invoke the actor method.
		 */
		public static void tell(ActorRef actor, String path, JsonValue... args) {
			sidecar.actorTell(actor.getType(), actor.getId(), path, packArgs(args));
		}

		/**
		 * Synchronous actor invocation where the invoked method will execute as part of
		 * the current session.
		 *
		 * @param caller The calling actor.
		 * @param actor  The target actor.
		 * @param path   The actor method to invoke.
		 * @param args   The arguments with which to invoke the actor method.
		 * @return The result of the invoked actor method.
		 */
		public static JsonValue call(ActorInstance caller, ActorRef actor, String path, JsonValue... args)
				throws ActorMethodNotFoundException, ActorMethodInvocationException {
			try {
				Response response = sidecar.actorCall(actor.getType(), actor.getId(), path, caller.getSession(),
						packArgs(args));
				return callProcessResponse(response);
			} catch (WebApplicationException e) {
				if (e.getResponse() != null && e.getResponse().getStatus() == 404) {
					String msg = responseToString(e.getResponse());
					throw new ActorMethodNotFoundException(
							msg != null ? msg : "Not found: " + actor.getType() + "[" + actor.getId() + "]." + path, e);
				} else if (e.getResponse() != null && e.getResponse().getStatus() == 408) {
					throw new ActorMethodTimeoutException(
							"Method timeout: " + actor.getType() + "[" + actor.getId() + "]." + path);
				} else {
					throw e;
				}
			}
		}

		/**
		 * Synchronous actor invocation where the invoked method will execute as part of
		 * the specified session.
		 *
		 * @param session The session in which to execute the actor method
		 * @param actor   The target actor.
		 * @param path    The actor method to invoke.
		 * @param args    The arguments with which to invoke the actor method.
		 * @return The result of the invoked actor method.
		 */
		public static JsonValue call(String session, ActorRef actor, String path, JsonValue... args)
				throws ActorMethodNotFoundException, ActorMethodInvocationException, ActorMethodTimeoutException {
			try {
				Response response = sidecar.actorCall(actor.getType(), actor.getId(), path, session, packArgs(args));
				return callProcessResponse(response);
			} catch (WebApplicationException e) {
				if (e.getResponse() != null && e.getResponse().getStatus() == 404) {
					String msg = responseToString(e.getResponse());
					throw new ActorMethodNotFoundException(
							msg != null ? msg : "Not found: " + actor.getType() + "[" + actor.getId() + "]." + path, e);
				} else if (e.getResponse() != null && e.getResponse().getStatus() == 408) {
					throw new ActorMethodTimeoutException(
							"Method timeout: " + actor.getType() + "[" + actor.getId() + "]." + path);
				} else {
					throw e;
				}
			}
		}

		/**
		 * Synchronous actor invocation where the invoked method will execute in a new
		 * session.
		 *
		 * @param actor The target Actor.
		 * @param path  The actor method to invoke.
		 * @param args  The arguments with which to invoke the actor method.
		 * @return The result of the invoked actor method.
		 */
		public static JsonValue call(ActorRef actor, String path, JsonValue... args)
				throws ActorMethodNotFoundException, ActorMethodInvocationException {
			try {
				Response response = sidecar.actorCall(actor.getType(), actor.getId(), path, null, packArgs(args));
				return callProcessResponse(response);
			} catch (WebApplicationException e) {
				if (e.getResponse() != null && e.getResponse().getStatus() == 404) {
					String msg = responseToString(e.getResponse());
					throw new ActorMethodNotFoundException(
							msg != null ? msg : "Not found: " + actor.getType() + "[" + actor.getId() + "]." + path, e);
				} else if (e.getResponse() != null && e.getResponse().getStatus() == 408) {
					throw new ActorMethodTimeoutException(
							"Method timeout: " + actor.getType() + "[" + actor.getId() + "]." + path);
				} else {
					throw e;
				}
			}
		}

		/**
		 * Asynchronous actor invocation with eventual access to the result of the
		 * invocation.
		 *
		 * @param actor The target Actor.
		 * @param path  The actor method to invoke.
		 * @param args  The arguments with which to invoke the actor method.
		 * @return A CompletionStage containing the response returned from the actor
		 *         method invocation.
		 */
		public static CompletionStage<JsonValue> callAsync(ActorRef actor, String path, JsonValue... args) {
			CompletionStage<Response> cr = sidecar.actorCallAsync(actor.getType(), actor.getId(), path, null, packArgs(args));
			return cr.thenApply(r -> callProcessResponse(r));
		}

		// Internal helper to go from a Response to the JsonValue representing the
		// result of the method (or an exception)
		private static JsonValue callProcessResponse(Response response)
				throws ActorMethodNotFoundException, ActorMethodInvocationException {
			if (response.getStatus() == Status.OK.getStatusCode()) {
				JsonObject o = ((JsonValue) toValue(response)).asJsonObject();
				if (o.containsKey("error")) {
					String message = o.containsKey("message") ? o.getString("message") : "Unknown error";
					Throwable cause = o.containsKey("stack") ? new Throwable(o.getString("stack")) : null;
					cause.setStackTrace(new StackTraceElement[0]); // avoid duplicating the stack trace where we are creating this
																													// dummy exception...the real stack is in the msg.
					throw new ActorMethodInvocationException(message, cause);
				} else {
					return o.containsKey("value") ? o.get("value") : JsonValue.NULL;
				}
			} else if (response.getStatus() == Status.NOT_FOUND.getStatusCode()) {
				if (response.hasEntity()) {
					throw new ActorMethodNotFoundException(toValue(response).toString());
				} else {
					throw new ActorMethodNotFoundException();
				}
			} else if (response.getStatus() == Status.NO_CONTENT.getStatusCode()) {
				return null;
			} else {
				throw new ProcessingException(response.getStatus() + ": " + toValue(response));
			}
		}

		/**
		 * An actor method may return a ContinueResult to indicate that the "result"
		 * of the method is to schedule a subsequent invocation (either to itself or
		 * to another actor instance).
		 */
		public static final class ContinueResult {
			public final ActorRef actor;
			public final String path;
			public final JsonValue[] args;

			public ContinueResult(ActorRef actor, String path, JsonValue... args) {
				this.actor = actor;
				this.path = path;
				this.args = args;
			}
		}

		/**
		 * KAR API methods for Actor Reminders
		 */
		public static class Reminders {

			/**
			 * Cancel all reminders for an Actor instance.
			 *
			 * @param actor The Actor instance.
			 * @return The number of reminders that were cancelled.
			 */
			public static int cancelAll(ActorRef actor) {
				Response response = sidecar.actorCancelReminders(actor.getType(), actor.getId());
				return toInt(response);
			}

			/**
			 * Cancel a specific reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param reminderId The id of a specific reminder to cancel
			 * @return The number of reminders that were cancelled.
			 */
			public static int cancel(ActorRef actor, String reminderId) {
				Response response = sidecar.actorCancelReminder(actor.getType(), actor.getId(), reminderId, true);
				return toInt(response);
			}

			/**
			 * Get all reminders for an Actor instance.
			 *
			 * @param actor The Actor instance.
			 * @return An array of matching reminders
			 */
			public static Reminder[] getAll(ActorRef actor) {
				Response response = sidecar.actorGetReminders(actor.getType(), actor.getId());
				return toReminderArray(response);
			}

			/**
			 * Get a specific reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param reminderId The id of a specific reminder to cancel
			 * @return An array of matching reminders
			 */
			public static Reminder[] get(ActorRef actor, String reminderId) {
				Response response = sidecar.actorGetReminder(actor.getType(), actor.getId(), reminderId, true);
				return toReminderArray(response);
			}

			/**
			 * Schedule a reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param path       The actor method to invoke when the reminder fires.
			 * @param reminderId The id of the reminder being scheduled
			 * @param targetTime The earliest time at which the reminder should be delivered
			 * @param period     For periodic reminders, a String that is compatible with
			 *                   GoLang's Duration
			 * @param args       The arguments with which to invoke the actor method.
			 */
			public static void schedule(ActorRef actor, String path, String reminderId, Instant targetTime, Duration period,
					JsonValue... args) {
				JsonObjectBuilder builder = factory.createObjectBuilder();
				builder.add("path", "/" + path);
				builder.add("targetTime", targetTime.toString());

				if (period != null) {
					// Sigh. Encode in a way that GoLang will understand since it sadly doesn't
					// actually implement ISO-8601
					String goPeriod = "";
					if (period.toHours() > 0) {
						goPeriod += period.toHours() + "h";
						period.minusHours(period.toHours());
					}
					if (period.toMinutes() > 0) {
						goPeriod += period.toMinutes() + "m";
						period.minusMinutes(period.toMinutes());
					}
					if (period.toSeconds() > 0) {
						goPeriod += period.toSeconds() + "s";
						period.minusSeconds(period.toSeconds());
					}
					if (period.toMillis() > 0) {
						goPeriod += period.toMillis() + "ms";
						period.minusMillis(period.toMillis());
					}
					builder.add("period", goPeriod);
				}
				builder.add("data", packArgs(args));
				JsonObject requestBody = builder.build();

				sidecar.actorScheduleReminder(actor.getType(), actor.getId(), reminderId, requestBody);
			}
		}

		/**
		 * KAR API methods for Actor State
		 */
		public static class State {
			public static class ActorUpdateResult {
				public final int added;
				public final int removed;

				ActorUpdateResult(int added, int removed) {
					this.added = added;
					this.removed = removed;
				}
			};

			/**
			 * Get one value from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to use to access the instance's state
			 * @return The value associated with `key`
			 */
			public static JsonValue get(ActorRef actor, String key) {
				JsonValue value;
				try {
					Response resp = sidecar.actorGetState(actor.getType(), actor.getId(), key, true);
					return (JsonValue) toValue(resp);
				} catch (WebApplicationException e) {
					value = JsonValue.NULL;
				}
				return value;
			}

			/**
			 * Get all of an Actor's state.
			 *
			 * @param actor The Actor instance.
			 * @return A map representing the Actor's state
			 */
			public static Map<String, JsonValue> getAll(ActorRef actor) {
				Response response = sidecar.actorGetAllState(actor.getType(), actor.getId());
				try {
					return ((JsonValue) toValue(response)).asJsonObject();
				} catch (ClassCastException e) {
					return Collections.emptyMap();
				}
			}

			/**
			 * Check to see if an entry exists in an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to check against the instance's state
			 * @return `true` if the actor instance has a value defined for `key`, `false`
			 *         otherwise.
			 */
			public static boolean contains(ActorRef actor, String key) {
				Response resp;
				try {
					resp = sidecar.actorHeadState(actor.getType(), actor.getId(), key);
				} catch (WebApplicationException e) {
					resp = e.getResponse();
				}
				return resp != null && resp.getStatus() == Status.OK.getStatusCode();
			}

			/**
			 * Store one value to an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to use to access the instance's state
			 * @param value The value to store
			 * @return The number of new state entries created by this store (0 or 1)
			 */
			public static int set(ActorRef actor, String key, JsonValue value) {
				Response response = sidecar.actorSetState(actor.getType(), actor.getId(), key, value);
				return response.getStatus() == Status.CREATED.getStatusCode() ? 1 : 0;
			}

			/**
			 * Store multiple values to an Actor's state
			 *
			 * @param actor   The Actor instance.
			 * @param updates A map containing the state updates to perform
			 * @return The number of new state entries created by this operation
			 */
			public static int set(ActorRef actor, Map<String, JsonValue> updates) {
				if (updates.isEmpty())
					return 0;
				ActorUpdateResult result = update(actor, Collections.emptyList(), Collections.emptyMap(), updates,
						Collections.emptyMap());
				return result.added;
			}

			/**
			 * Remove one value from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to delete
			 * @return `1` if an entry was actually removed and `0` if there was no entry
			 *         for `key`.
			 */
			public static int remove(ActorRef actor, String key) {
				Response response = sidecar.actorDeleteState(actor.getType(), actor.getId(), key, true);
				return toInt(response);
			}

			/**
			 * Remove multiple values from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param keys  The keys to delete
			 * @return the number of entries actually removed
			 */
			public static int removeAll(ActorRef actor, List<String> keys) {
				if (keys.isEmpty())
					return 0;
				ActorUpdateResult res = update(actor, keys, Collections.emptyMap(), Collections.emptyMap(),
						Collections.emptyMap());
				return res.removed;
			}

			/**
			 * Remove all elements of an Actor's user level state. Unlike
			 * {@link Actors#remove} this method is synchronous and does not remove the
			 * KAR-level mapping of the instance to a specific runtime Process.
			 *
			 * @param actor The Actor instance.
			 * @return The number of removed key/value pairs
			 */
			public static int removeAll(ActorRef actor) {
				Response response = sidecar.actorDeleteAllState(actor.getType(), actor.getId());
				return toInt(response);
			}

			/**
			 * Perform a multi-element update operation on a Actor's state. This method is
			 * the most general form of Actor state update and enables both top-level keys
			 * and submap keys to be removed and updated in a single KAR operation.
			 *
			 * @param actor          The Actor instance.
			 * @param removals       The keys to remove from the actor state
			 * @param submapRemovals A mapping from submap names to the keys to remove from
			 *                       each submap
			 * @param updates        The updates to perform to the actors state
			 * @param submapUpdates  A mapping from submap names to the updates to perform
			 *                       on each submap
			 * @return An object containing the number of state entries removed and added by
			 *         the update.
			 */
			public static ActorUpdateResult update(ActorRef actor, List<String> removals,
					Map<String, List<String>> submapRemovals, Map<String, JsonValue> updates,
					Map<String, Map<String, JsonValue>> submapUpdates) {
				JsonObjectBuilder requestBuilder = factory.createObjectBuilder();

				if (!removals.isEmpty()) {
					JsonArrayBuilder jb = factory.createArrayBuilder();
					for (String k : removals) {
						jb.add(k);
					}
					requestBuilder.add("removals", jb.build());
				}

				if (!submapRemovals.isEmpty()) {
					JsonObjectBuilder smr = factory.createObjectBuilder();
					for (Entry<String, List<String>> e : submapRemovals.entrySet()) {
						JsonArrayBuilder jb = factory.createArrayBuilder();
						for (String k : e.getValue()) {
							jb.add(k);
						}
						smr.add(e.getKey(), jb.build());
					}
					requestBuilder.add("submapremovals", smr.build());
				}

				if (!updates.isEmpty()) {
					JsonObjectBuilder u = factory.createObjectBuilder();
					for (Entry<String, JsonValue> e : updates.entrySet()) {
						u.add(e.getKey(), e.getValue());
					}
					requestBuilder.add("updates", u.build());
				}

				if (!submapUpdates.isEmpty()) {
					JsonObjectBuilder smu = factory.createObjectBuilder();
					for (Entry<String, Map<String, JsonValue>> e : submapUpdates.entrySet()) {
						JsonObjectBuilder u = factory.createObjectBuilder();
						for (Entry<String, JsonValue> e2 : e.getValue().entrySet()) {
							u.add(e2.getKey(), e2.getValue());
						}
						smu.add(e.getKey(), u.build());
					}
					requestBuilder.add("submapupdates", smu.build());
				}

				JsonObject params = requestBuilder.build();
				Response response = sidecar.actorUpdate(actor.getType(), actor.getId(), params);
				JsonObject responseObject = ((JsonValue) toValue(response)).asJsonObject();
				int added = responseObject.getInt("added");
				int removed = responseObject.getInt("removed");

				return new ActorUpdateResult(added, removed);
			}

			/**
			 * KAR API methods for optimized operations for storing a map as a nested
			 * element of an Actor's state.
			 */
			public static class Submap {
				/**
				 * Get one value from a submap of an Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap
				 * @param key    The subkey to use to access the instance's state
				 * @return The value associated with `key/subkey`
				 */
				public static JsonValue get(ActorRef actor, String submap, String key) {
					JsonValue value;
					try {
						Response resp = sidecar.actorGetWithSubkeyState(actor.getType(), actor.getId(), submap, key, true);
						return (JsonValue) toValue(resp);
					} catch (WebApplicationException e) {
						value = JsonValue.NULL;
					}
					return value;
				}

				/**
				 * Get all key/value pairs of the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return An array containing the currently defined subkeys
				 */
				public static Map<String, JsonValue> getAll(ActorRef actor, String submap) {
					JsonObjectBuilder jb = factory.createObjectBuilder();
					jb.add("op", Json.createValue("get"));
					JsonObject params = jb.build();
					Response response = sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params);
					try {
						return ((JsonValue) toValue(response)).asJsonObject();
					} catch (ClassCastException e) {
						return Collections.emptyMap();
					}
				}

				/**
				 * Check to see if an entry exists in a submap in an Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap
				 * @param key    The key to check for in the given submap
				 * @return `true` if the actor instance has a value defined for `key/subkey`,
				 *         `false` otherwise.
				 */
				public static boolean contains(ActorRef actor, String submap, String key) {
					Response resp;
					try {
						resp = sidecar.actorHeadWithSubkeyState(actor.getType(), actor.getId(), submap, key);
					} catch (WebApplicationException e) {
						resp = e.getResponse();
					}
					return resp != null && resp.getStatus() == Status.OK.getStatusCode();
				}

				/**
				 * Store one value to a submap in an Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap to update
				 * @param key    The key in the submap to update
				 * @param value  The value to store at `key/subkey`
				 * @return The number of new state entries created by this store (0 or 1)
				 */
				public static int set(ActorRef actor, String submap, String key, JsonValue value) {
					Response response = sidecar.actorSetWithSubkeyState(actor.getType(), actor.getId(), submap, key, value);
					return response.getStatus() == Status.CREATED.getStatusCode() ? 1 : 0;
				}

				/**
				 * Store multiple values to an Actor sub-map with name `key`
				 *
				 * @param actor   The Actor instance.
				 * @param submap  The name of the submap to which the updates should be
				 *                performed
				 * @param updates A map containing the (subkey, value) pairs to store
				 * @return The number of new map entries created by this operation
				 */
				public static int set(ActorRef actor, String submap, Map<String, JsonValue> updates) {
					if (updates.isEmpty())
						return 0;
					Map<String, Map<String, JsonValue>> tmp = new HashMap<String, Map<String, JsonValue>>();
					tmp.put(submap, updates);
					ActorUpdateResult res = update(actor, Collections.emptyList(), Collections.emptyMap(), Collections.emptyMap(),
							tmp);
					return res.added;
				}

				/**
				 * Remove one value from a submap in the Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap from which to delete the key
				 * @param key    The key of the entry to delete from the submap
				 * @return `1` if an entry was actually removed and `0` if there was no entry
				 *         for `key`.
				 */
				public static int remove(ActorRef actor, String submap, String key) {
					Response response = sidecar.actorDeleteWithSubkeyState(actor.getType(), actor.getId(), submap, key, true);
					return toInt(response);
				}

				/**
				 * Remove multiple values from one submap of an Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap from which to delete the keys
				 * @param keys   The keys to delete
				 * @return the number of entries actually removed
				 */
				public static int removeAll(ActorRef actor, String submap, List<String> keys) {
					if (keys.isEmpty())
						return 0;

					Map<String, List<String>> tmp = new HashMap<String, List<String>>();
					tmp.put(submap, keys);
					ActorUpdateResult res = update(actor, Collections.emptyList(), tmp, Collections.emptyMap(),
							Collections.emptyMap());
					return res.removed;
				}

				/**
				 * Remove all values from a submap in the Actor's state.
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return The number of removed subkey entrys
				 */
				public static int removeAll(ActorRef actor, String submap) {
					JsonObjectBuilder jb = factory.createObjectBuilder();
					jb.add("op", Json.createValue("clear"));
					JsonObject params = jb.build();
					Response response = sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params);
					return toInt(response);
				}

				/**
				 * Get the keys of the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return An array containing the currently defined subkeys
				 */
				public static String[] keys(ActorRef actor, String submap) {
					JsonObjectBuilder jb = factory.createObjectBuilder();
					jb.add("op", Json.createValue("keys"));
					JsonObject params = jb.build();
					Response response = sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params);
					Object[] jstrings = ((JsonValue) toValue(response)).asJsonArray().toArray();
					String[] ans = new String[jstrings.length];
					for (int i = 0; i < jstrings.length; i++) {
						ans[i] = ((JsonValue) jstrings[i]).toString();
					}
					return ans;
				}

				/**
				 * Get the number of keys in the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return The number of currently define keys in the submap
				 */
				public static int size(ActorRef actor, String submap) {
					JsonObjectBuilder jb = Json.createObjectBuilder();
					jb.add("op", Json.createValue("size"));
					JsonObject params = jb.build();
					Response response = sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params);
					return toInt(response);
				}
			}
		}
	}

	/**
	 * KAR API methods for Eventing.
	 */
	public static class Events {

		/**
		 * Cancel all subscriptions for an Actor instance.
		 *
		 * @param actor The Actor instance.
		 * @return The number of subscriptions that were cancelled.
		 */
		public static int cancelAllSubscriptions(ActorRef actor) {
			Response response = sidecar.actorCancelAllSubscriptions(actor.getType(), actor.getId());
			return toInt(response);
		}

		/**
		 * Cancel a specific subscription for an Actor instance.
		 *
		 * @param actor          The Actor instance.
		 * @param subscriptionId The id of a specific subscription to cancel
		 * @return The number of subscriptions that were cancelled.
		 */
		public static int cancelSubscription(ActorRef actor, String subscriptionId) {
			Response response = sidecar.actorCancelSubscription(actor.getType(), actor.getId(), subscriptionId);
			return toInt(response);
		}

		/**
		 * Get all subscriptions for an Actor instance.
		 *
		 * @param actor The Actor instance.
		 * @return An array of subscriptions
		 */
		public static Subscription[] getSubscriptions(ActorRef actor) {
			Response response = sidecar.actorGetAllSubscriptions(actor.getType(), actor.getId());
			return toSubscriptionArray(response);
		}

		/**
		 * Get a specific subscription for an Actor instance.
		 *
		 * @param actor          The Actor instance.
		 * @param subscriptionId The id of a specific subscription to get
		 * @return An array of zero or one subscription
		 */
		public static Subscription[] getSubscription(ActorRef actor, String subscriptionId) {
			Response response = sidecar.actorGetSubscription(actor.getType(), actor.getId(), subscriptionId);
			return toSubscriptionArray(response);
		}

		/**
		 * Subscribe an Actor instance method to a topic.
		 *
		 * @param actor The Actor instance to subscribe
		 * @param path  The actor method to invoke on each event received on the topic
		 * @param topic The topic to which to subscribe
		 */
		public static void subscribe(ActorRef actor, String path, String topic) {
			subscribe(actor, path, topic, topic);
		}

		/**
		 * Subscribe an Actor instance method to a topic.
		 *
		 * @param actor          The Actor instance to subscribe
		 * @param path           The actor method to invoke on each event received on
		 *                       the topic
		 * @param topic          The topic to which to subscribe
		 * @param subscriptionId The subscriptionId to use for this subscription
		 */
		public static void subscribe(ActorRef actor, String path, String topic, String subscriptionId) {
			JsonObjectBuilder builder = factory.createObjectBuilder();
			builder.add("path", "/" + path);
			builder.add("topic", topic);
			JsonObject data = builder.build();
			sidecar.actorSubscribe(actor.getType(), actor.getId(), subscriptionId, data);
		}

		/**
		 * Create a topic using the default Kafka configuration options.
		 *
		 * @param topic The name of the topic to create
		 */
		public static void createTopic(String topic) {
			sidecar.eventCreateTopic(topic, JsonValue.EMPTY_JSON_OBJECT);
		}

		/**
		 * Delete a topic.
		 *
		 * @param topic the name of the topic to delete
		 */
		public static void deleteTopic(String topic) {
			sidecar.eventDeleteTopic(topic);
		}

		/**
		 * Publish an event on a topic.
		 *
		 * @param topic the name of the topic on which to publish
		 * @param event the event to publish
		 */
		public static void publish(String topic, JsonValue event) {
			sidecar.eventPublish(topic, event);
		}
	}

	/**
	 * KAR API methods for directly interacting with the KAR service mesh
	 */
	public static class Sys {
		/**
		 * Shutdown this sidecar. Does not return.
		 */
		public static void shutdown() {
			sidecar.shutdown();
		}

		/**
		 * Get information about a system component.
		 *
		 * @param component The component whose information is being requested
		 * @return information about the given component
		 */
		public static Object information(String component) {
			Response response = sidecar.systemInformation(component);
			return toValue(response);
		}
	}
}
