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

import java.io.StringReader;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.logging.Logger;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonBuilderFactory;
import javax.json.JsonException;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonReaderFactory;
import javax.json.JsonValue;
import javax.json.stream.JsonParsingException;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.Reminder;
import com.ibm.research.kar.actor.Subscription;
import com.ibm.research.kar.actor.exceptions.ActorMethodInvocationException;
import com.ibm.research.kar.actor.exceptions.ActorMethodNotFoundException;
import com.ibm.research.kar.actor.exceptions.ActorMethodTimeoutException;
import com.ibm.research.kar.quarkus.KarSidecar;
import com.ibm.research.kar.quarkus.KarSidecar.KarHttpClient.KarSidecarError;
import com.ibm.research.kar.runtime.KarResponse;

import io.vertx.mutiny.core.buffer.Buffer;
import io.vertx.mutiny.ext.web.client.HttpResponse;
import io.smallrye.mutiny.Uni;

public class Kar {
	private static final Logger logger = Logger.getLogger(Kar.class.getName());

	private static KarSidecar sidecar = instantiateSidecar();
	private static final JsonBuilderFactory factory = Json.createBuilderFactory(Map.of());
	private static final JsonReaderFactory readerFactory = Json.createReaderFactory(Map.of());


	private static KarSidecar instantiateSidecar() {
		return new KarSidecar();
	}

	private static JsonArray packArgs(JsonValue[] args) {
		JsonArrayBuilder ja = factory.createArrayBuilder();
		for (JsonValue a : args) {
			ja.add(a);
		}
		return ja.build();
	}

	private static JsonValue toJsonValue(HttpResponse<Buffer> response) {
			return readerFactory.createReader(new StringReader(response.bodyAsString())).readValue();
	}

	private static int toInt(HttpResponse<Buffer> response) {
		try {
			return Integer.parseInt(response.bodyAsString());
		} catch (NumberFormatException e) {
			return 0;
		}
	}

	private static Reminder[] toReminderArray(HttpResponse<Buffer> response) {
		try {
			ArrayList<Reminder> res = new ArrayList<Reminder>();
			JsonArray ja = toJsonValue(response).asJsonArray();
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

	@SuppressWarnings("unused")
	private static Subscription[] toSubscriptionArray(HttpResponse<Buffer> response) {
		try {
			ArrayList<Subscription> res = new ArrayList<Subscription>();
			JsonArray ja = toJsonValue(response).asJsonArray();
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

		/**
		 * REST DELETE
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> delete(String service, String path) {
			return sidecar.callDelete(service, path);
		}

		/**
		 * REST GET
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> get(String service, String path) {
			return sidecar.callGet(service, path);
		}

		/**
		 * REST HEAD
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> head(String service, String path) {
			return sidecar.callHead(service, path);
		}

		/**
		 * REST OPTIONS
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> options(String service, String path) {
			return sidecar.callOptions(service, path, JsonValue.NULL);
		}

		/**
		 * REST OPTIONS
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> options(String service, String path, JsonValue body) {
			return sidecar.callOptions(service, path, body);
		}

		/**
		 * REST PATCH
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> patch(String service, String path, JsonValue body) {
			return sidecar.callPatch(service, path, body);
		}

		/**
		 * REST POST
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> post(String service, String path, JsonValue body) {
			return sidecar.callPost(service, path, body);
		}

		/**
		 * REST PUT
		 *
		 * @param service The name of the service.
		 * @param path    The service endpoint.
		 * @param body    The request body.
		 * @return The response returned by the target service.
		 */
		public static Uni<HttpResponse<Buffer>> put(String service, String path, JsonValue body) {
			return sidecar.callPut(service, path, body);
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
		public static Uni<Void> tell(String service, String path, JsonValue body) {
			return sidecar.tellPost(service, path, body).chain(() -> Uni.createFrom().nullItem());
		}

		/**
		 * Synchronous service invocation
		 *
		 * @param service The name of the service to invoke.
		 * @param path    The service endpoint to invoke.
		 * @param body    The request body with which to invoke the service endpoint.
		 * @return The result returned by the target service.
		 */
		public static Uni<Object> call(String service, String path, JsonValue body) {
			return sidecar.callPost(service, path, body).chain(resp -> {
				Object result = resp.bodyAsString();
				String contentType = resp.getHeader("Content-Type");
				if (contentType != null && !contentType.startsWith(KarResponse.TEXT_PLAIN)) {
					try {
						result = readerFactory.createReader(new StringReader(resp.bodyAsString())).readValue();
					} catch (JsonException e) {
						result = resp.bodyAsString();
					}
				}
				return Uni.createFrom().item(result);
			});
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
		public static Uni<Void> remove(ActorRef actor) {
			return sidecar.actorDelete(actor.getType(), actor.getId()).chain(()->Uni.createFrom().nullItem());
		}

		/**
		 * Asynchronous actor invocation; returns as soon as the invocation has been
		 * initiated.
		 *
		 * @param actor The target actor.
		 * @param path  The actor method to invoke.
		 * @param args  The arguments with which to invoke the actor method.
		 */
		public static Uni<Void> tell(ActorRef actor, String path, JsonValue... args) {
			return sidecar.actorTell(actor.getType(), actor.getId(), path, packArgs(args)).chain(()->Uni.createFrom().nullItem());
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
		public static Uni<JsonValue> call(ActorInstance caller, ActorRef actor, String path, JsonValue... args) {
			return sidecar.actorCall(actor.getType(), actor.getId(), path, caller.getSession(), packArgs(args))
				.chain(response -> callProcessResponse(response, actor, path));
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
		public static Uni<JsonValue> call(String session, ActorRef actor, String path, JsonValue... args) {
				return sidecar.actorCall(actor.getType(), actor.getId(), path, session, packArgs(args))
					.chain(response -> callProcessResponse(response, actor, path));
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
		public static Uni<JsonValue> call(ActorRef actor, String path, JsonValue... args) {
			return sidecar.actorCall(actor.getType(), actor.getId(), path, null, packArgs(args))
				.chain(response -> callProcessResponse(response, actor, path));
		}

		// Internal helper to go from a Response to the JsonValue representing the
		// result of the method (or a Uni with a failure that propagates the exception)
		private static Uni<JsonValue> callProcessResponse(HttpResponse<Buffer> response, ActorRef actor, String path) {
			if (response.statusCode() == KarResponse.OK) {
				JsonObject o = toJsonValue(response).asJsonObject();
				if (o.containsKey("error")) {
					String message = o.containsKey("message") ? o.getString("message") : "Unknown error";
					Throwable cause = o.containsKey("stack") ? new Throwable(o.getString("stack")) : null;

					cause.setStackTrace(new StackTraceElement[0]); // avoid duplicating the stack trace where we are creating this dummy exception...the real stack is in the msg.
					return Uni.createFrom().failure(new ActorMethodInvocationException(message, cause));
				} else {
					return Uni.createFrom().item(o.containsKey("value") ? (JsonValue)o.get("value") : JsonValue.NULL);
				}
			} else if (response.statusCode() == KarResponse.NOT_FOUND) {
				String msg = response.bodyAsString();
				if (msg != null) {
					return Uni.createFrom().failure(new ActorMethodNotFoundException(msg));
				} else {
					return Uni.createFrom().failure(new ActorMethodNotFoundException("Not found: " + actor.getType() + "[" + actor.getId() + "]." + path));
				}
			} else if (response.statusCode() == KarResponse.NO_CONTENT) {
				return Uni.createFrom().nullItem();
			} else if (response.statusCode() == KarResponse.REQUEST_TIMEOUT) {
				return Uni.createFrom().failure(new ActorMethodTimeoutException("Method timeout: " + actor.getType() + "[" + actor.getId() + "]." + path));
			} else {
				return Uni.createFrom().failure(new KarSidecarError(response));
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
			public static Uni<Integer> cancelAll(ActorRef actor) {
				return sidecar.actorCancelReminders(actor.getType(), actor.getId())
					.chain(response -> Uni.createFrom().item(toInt(response)));
			}

			/**
			 * Cancel a specific reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param reminderId The id of a specific reminder to cancel
			 * @return The number of reminders that were cancelled.
			 */
			public static Uni<Integer> cancel(ActorRef actor, String reminderId) {
				return sidecar.actorCancelReminder(actor.getType(), actor.getId(), reminderId, true)
					.chain(response -> Uni.createFrom().item(toInt(response)));
			}

			/**
			 * Get all reminders for an Actor instance.
			 *
			 * @param actor The Actor instance.
			 * @return An array of matching reminders
			 */
			public static Uni<Reminder[]> getAll(ActorRef actor) {
				return sidecar.actorGetReminders(actor.getType(), actor.getId())
					.chain(response -> Uni.createFrom().item(toReminderArray(response)));
			}

			/**
			 * Get a specific reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param reminderId The id of a specific reminder to cancel
			 * @return An array of matching reminders
			 */
			public static Uni<Reminder[]> get(ActorRef actor, String reminderId) {
				return sidecar.actorGetReminder(actor.getType(), actor.getId(), reminderId, true)
					.chain(response -> Uni.createFrom().item(toReminderArray(response)));
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
			public static Uni<Void> schedule(ActorRef actor, String path, String reminderId, Instant targetTime, Duration period, JsonValue... args) {
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

				return sidecar.actorScheduleReminder(actor.getType(), actor.getId(), reminderId, requestBody).chain(() -> Uni.createFrom().nullItem());
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
			public static Uni<JsonValue> get(ActorRef actor, String key) {
				return sidecar.actorGetState(actor.getType(), actor.getId(), key, true)
					.chain(response -> Uni.createFrom().item(toJsonValue(response)))
					.onFailure().recoverWithItem(JsonValue.NULL);
			}

			/**
			 * Get all of an Actor's state.
			 *
			 * @param actor The Actor instance.
			 * @return A map representing the Actor's state
			 */
			public static Uni<Map<String, JsonValue>> getAll(ActorRef actor) {
				return sidecar.actorGetAllState(actor.getType(), actor.getId())
					.chain(response -> Uni.createFrom().item((Map<String, JsonValue>)toJsonValue(response)))
					.onFailure().recoverWithItem(Collections.emptyMap());
			}

			/**
			 * Check to see if an entry exists in an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to check against the instance's state
			 * @return `true` if the actor instance has a value defined for `key`, `false`
			 *         otherwise.
			 */
			public static Uni<Boolean> contains(ActorRef actor, String key) {
				return sidecar.actorHeadState(actor.getType(), actor.getId(), key)
					.chain(response -> Uni.createFrom().item(response.statusCode() == KarResponse.OK));
			}

			/**
			 * Store one value to an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to use to access the instance's state
			 * @param value The value to store
			 * @return The number of new state entries created by this store (0 or 1)
			 */
			public static Uni<Integer> set(ActorRef actor, String key, JsonValue value) {
				return sidecar.actorSetState(actor.getType(), actor.getId(), key, value)
					.chain(response -> Uni.createFrom().item(toInt(response)));
			}

			/**
			 * Store one value to an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to use to access the instance's state
			 * @param value The value to store
			 */
			public static Uni<Void> setV(ActorRef actor, String key, JsonValue value) {
				return sidecar.actorSetState(actor.getType(), actor.getId(), key, value)
					.chain(() -> Uni.createFrom().nullItem());
			}

			/**
			 * Store multiple values to an Actor's state
			 *
			 * @param actor   The Actor instance.
			 * @param updates A map containing the state updates to perform
			 * @return The number of new state entries created by this store
			 */
			public static Uni<Integer> set(ActorRef actor, Map<String, JsonValue> updates) {
				if (updates.isEmpty()) {
					return Uni.createFrom().nullItem();
				}
				return update(actor, Collections.emptyList(), Collections.emptyMap(), updates, Collections.emptyMap())
					.chain(res -> Uni.createFrom().item(res.removed));
			}

			/**
			 * Store multiple values to an Actor's state
			 *
			 * @param actor   The Actor instance.
			 * @param updates A map containing the state updates to perform
			 */
			public static Uni<Void> setV(ActorRef actor, Map<String, JsonValue> updates) {
				if (updates.isEmpty()) {
					return Uni.createFrom().nullItem();
				}
				return update(actor, Collections.emptyList(), Collections.emptyMap(), updates, Collections.emptyMap())
					.chain(() -> Uni.createFrom().nullItem());
			}

			/**
			 * Remove one value from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to delete
			 * @return `1` if an entry was actually removed and `0` if there was no entry
			 *         for `key`.
			 */
			public static Uni<Integer> remove(ActorRef actor, String key) {
				return sidecar.actorDeleteState(actor.getType(), actor.getId(), key, true)
					.chain(response -> Uni.createFrom().item(toInt(response)));
			}

			/**
			 * Remove multiple values from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param keys  The keys to delete
			 * @return the number of entries actually removed
			 */
			public static Uni<Integer> removeAll(ActorRef actor, List<String> keys) {
				if (keys.isEmpty()) {
					return Uni.createFrom().item(0);
				}
				return update(actor, keys, Collections.emptyMap(), Collections.emptyMap(), Collections.emptyMap())
					.chain(res -> Uni.createFrom().item(res.removed));
			}

			/**
			 * Remove all elements of an Actor's user level state. Unlike
			 * {@link Actors#remove} this method is synchronous and does not remove the
			 * KAR-level mapping of the instance to a specific runtime Process.
			 *
			 * @param actor The Actor instance.
			 * @return The number of removed key/value pairs
			 */
			public static Uni<Integer> removeAll(ActorRef actor) {
				return sidecar.actorDeleteAllState(actor.getType(), actor.getId())
					.chain(response -> Uni.createFrom().item(toInt(response)));
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
			public static Uni<ActorUpdateResult> update(ActorRef actor, List<String> removals,
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
				return sidecar.actorUpdate(actor.getType(), actor.getId(), params)
					.chain(response -> {
						JsonObject responseObject = toJsonValue(response).asJsonObject();
						int added = responseObject.getInt("added");
						int removed = responseObject.getInt("removed");
						return Uni.createFrom().item(new ActorUpdateResult(added, removed));
					});
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
				public static Uni<JsonValue> get(ActorRef actor, String submap, String key) {
					return sidecar.actorGetWithSubkeyState(actor.getType(), actor.getId(), submap, key, true)
						.chain(response -> Uni.createFrom().item(toJsonValue(response)))
						.onFailure().recoverWithItem(JsonValue.NULL);
				}

				/**
				 * Get all key/value pairs of the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return An array containing the currently defined subkeys
				 */
				public static Uni<Map<String, JsonValue>> getAll(ActorRef actor, String submap) {
					JsonObjectBuilder jb = factory.createObjectBuilder();
					jb.add("op", Json.createValue("get"));
					JsonObject params = jb.build();
					return sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params)
						.chain(response -> Uni.createFrom().item((Map<String, JsonValue>)toJsonValue(response)))
						.onFailure().recoverWithItem(Collections.emptyMap());
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
				public static Uni<Boolean> contains(ActorRef actor, String submap, String key) {
					return sidecar.actorHeadWithSubkeyState(actor.getType(), actor.getId(), submap, key)
						.chain(response -> Uni.createFrom().item(response.statusCode() == KarResponse.OK));
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
				public static Uni<Integer> set(ActorRef actor, String submap, String key, JsonValue value) {
					return sidecar.actorSetWithSubkeyState(actor.getType(), actor.getId(), submap, key, value)
						.chain(response -> Uni.createFrom().item(toInt(response)));
				}

				/**
				 * Store one value to a submap in an Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap to update
				 * @param key    The key in the submap to update
				 * @param value  The value to store at `key/subkey`
				 */
				public static Uni<Void> setV(ActorRef actor, String submap, String key, JsonValue value) {
					return sidecar.actorSetWithSubkeyState(actor.getType(), actor.getId(), submap, key, value).chain(() -> Uni.createFrom().nullItem());
				}

				/**
				 * Store multiple values to an Actor sub-map with name `key`
				 *
				 * @param actor   The Actor instance.
				 * @param submap  The name of the submap to which the updates should be
				 *                performed
				 * @param updates A map containing the (subkey, value) pairs to store
				 * @return The number of new state entries created by this store (0 or 1)
				 */
				public static Uni<Integer> set(ActorRef actor, String submap, Map<String, JsonValue> updates) {
					if (updates.isEmpty()) {
						return Uni.createFrom().nullItem();
					}
					Map<String, Map<String, JsonValue>> tmp = new HashMap<String, Map<String, JsonValue>>();
					tmp.put(submap, updates);
					return update(actor, Collections.emptyList(), Collections.emptyMap(), Collections.emptyMap(), tmp)
						.chain(res -> Uni.createFrom().item(res.added));
				}

				/**
				 * Store multiple values to an Actor sub-map with name `key`
				 *
				 * @param actor   The Actor instance.
				 * @param submap  The name of the submap to which the updates should be
				 *                performed
				 * @param updates A map containing the (subkey, value) pairs to store
				 * @return The number of new state entries created by this store (0 or 1)
				 */
				public static Uni<Void> setV(ActorRef actor, String submap, Map<String, JsonValue> updates) {
					if (updates.isEmpty()) {
						return Uni.createFrom().nullItem();
					}
					Map<String, Map<String, JsonValue>> tmp = new HashMap<String, Map<String, JsonValue>>();
					tmp.put(submap, updates);
					return update(actor, Collections.emptyList(), Collections.emptyMap(), Collections.emptyMap(), tmp)
						.chain(() -> Uni.createFrom().nullItem());
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
				public static Uni<Integer> remove(ActorRef actor, String submap, String key) {
					return sidecar.actorDeleteWithSubkeyState(actor.getType(), actor.getId(), submap, key, true)
						.chain(response -> Uni.createFrom().item(toInt(response)));
				}

				/**
				 * Remove multiple values from one submap of an Actor's state
				 *
				 * @param actor  The Actor instance.
				 * @param submap The name of the submap from which to delete the keys
				 * @param keys   The keys to delete
				 * @return the number of entries actually removed
				 */
				public static Uni<Integer> removeAll(ActorRef actor, String submap, List<String> keys) {
					if (keys.isEmpty()) {
						return Uni.createFrom().item(0);
					}

					Map<String, List<String>> tmp = new HashMap<String, List<String>>();
					tmp.put(submap, keys);
					return update(actor, Collections.emptyList(), tmp, Collections.emptyMap(), Collections.emptyMap())
						.chain(res -> Uni.createFrom().item(res.removed));
				}

				/**
				 * Remove all values from a submap in the Actor's state.
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return The number of removed subkey entrys
				 */
				public static Uni<Integer> removeAll(ActorRef actor, String submap) {
					JsonObjectBuilder jb = factory.createObjectBuilder();
					jb.add("op", Json.createValue("clear"));
					JsonObject params = jb.build();
					return sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params)
						.chain(response -> Uni.createFrom().item(toInt(response)));
				}

				/**
				 * Get the keys of the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return An array containing the currently defined subkeys
				 */
				public static Uni<String[]> keys(ActorRef actor, String submap) {
					JsonObjectBuilder jb = factory.createObjectBuilder();
					jb.add("op", Json.createValue("keys"));
					JsonObject params = jb.build();
					return sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params)
						.chain(response -> {
							Object[] jstrings = toJsonValue(response).asJsonArray().toArray();
							String[] ans = new String[jstrings.length];
							for (int i = 0; i < jstrings.length; i++) {
								ans[i] = ((JsonValue) jstrings[i]).toString();
							}
							return Uni.createFrom().item(ans);

						});
				}

				/**
				 * Get the number of keys in the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return The number of currently define keys in the submap
				 */
				public static Uni<Integer> size(ActorRef actor, String submap) {
					JsonObjectBuilder jb = Json.createObjectBuilder();
					jb.add("op", Json.createValue("size"));
					JsonObject params = jb.build();
					return sidecar.actorSubmapOp(actor.getType(), actor.getId(), submap, params)
						.chain(response -> Uni.createFrom().item(toInt(response)));
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
			throw new UnsupportedOperationException();
		}

		/**
		 * Cancel a specific subscription for an Actor instance.
		 *
		 * @param actor          The Actor instance.
		 * @param subscriptionId The id of a specific subscription to cancel
		 * @return The number of subscriptions that were cancelled.
		 */
		public static int cancelSubscription(ActorRef actor, String subscriptionId) {
			throw new UnsupportedOperationException();
		}

		/**
		 * Get all subscriptions for an Actor instance.
		 *
		 * @param actor The Actor instance.
		 * @return An array of subscriptions
		 */
		public static Subscription[] getSubscriptions(ActorRef actor) {
			throw new UnsupportedOperationException();
		}

		/**
		 * Get a specific subscription for an Actor instance.
		 *
		 * @param actor          The Actor instance.
		 * @param subscriptionId The id of a specific subscription to get
		 * @return An array of zero or one subscription
		 */
		public static Subscription[] getSubscription(ActorRef actor, String subscriptionId) {
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
		}

		/**
		 * Create a topic using the default Kafka configuration options.
		 *
		 * @param topic The name of the topic to create
		 */
		public static void createTopic(String topic) {
			throw new UnsupportedOperationException();
		}

		/**
		 * Delete a topic.
		 *
		 * @param topic the name of the topic to delete
		 */
		public static void deleteTopic(String topic) {
			throw new UnsupportedOperationException();
		}

		/**
		 * Publish an event on a topic.
		 *
		 * @param topic the name of the topic on which to publish
		 * @param event the event to publish
		 */
		public static void publish(String topic, JsonValue event) {
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
		}

		/**
		 * Get information about a system component.
		 *
		 * @param component The component whose information is being requested
		 * @return information about the given component
		 */
		public static Object information(String component) {
			throw new UnsupportedOperationException();
		}
	}
}
