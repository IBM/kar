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

import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletionStage;
import java.util.logging.Logger;

import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.Reminder;
import com.ibm.research.kar.actor.Subscription;
import com.ibm.research.kar.actor.exceptions.ActorMethodInvocationException;
import com.ibm.research.kar.actor.exceptions.ActorMethodNotFoundException;
import com.ibm.research.kar.actor.exceptions.ActorMethodTimeoutException;

public class Kar {
	public final static String KAR_ACTOR_JSON = "application/kar+json";

	private static final Logger logger = Logger.getLogger(Kar.class.getName());

	private Kar() {
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

		/*
		 * Eliding for now, as these are likely to be Quarkus Specific. Eventually, a
		 * sync and async version of each REST verb (get, put, delete, etc)
		 */

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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
			throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
			}

			/**
			 * Cancel a specific reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param reminderId The id of a specific reminder to cancel
			 * @return The number of reminders that were cancelled.
			 */
			public static int cancel(ActorRef actor, String reminderId) {
				throw new UnsupportedOperationException();
			}

			/**
			 * Get all reminders for an Actor instance.
			 *
			 * @param actor The Actor instance.
			 * @return An array of matching reminders
			 */
			public static Reminder[] getAll(ActorRef actor) {
				throw new UnsupportedOperationException();
			}

			/**
			 * Get a specific reminder for an Actor instance.
			 *
			 * @param actor      The Actor instance.
			 * @param reminderId The id of a specific reminder to cancel
			 * @return An array of matching reminders
			 */
			public static Reminder[] get(ActorRef actor, String reminderId) {
				throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
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
			}

			/**
			 * Get one value from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param key   The key to use to access the instance's state
			 * @return The value associated with `key`
			 */
			public static JsonValue get(ActorRef actor, String key) {
				throw new UnsupportedOperationException();
			}

			/**
			 * Get all of an Actor's state.
			 *
			 * @param actor The Actor instance.
			 * @return A map representing the Actor's state
			 */
			public static Map<String, JsonValue> getAll(ActorRef actor) {
				throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
			}

			/**
			 * Store multiple values to an Actor's state
			 *
			 * @param actor   The Actor instance.
			 * @param updates A map containing the state updates to perform
			 * @return The number of new state entries created by this operation
			 */
			public static int set(ActorRef actor, Map<String, JsonValue> updates) {
				throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
			}

			/**
			 * Remove multiple values from an Actor's state
			 *
			 * @param actor The Actor instance.
			 * @param keys  The keys to delete
			 * @return the number of entries actually removed
			 */
			public static int removeAll(ActorRef actor, List<String> keys) {
				throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
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
				throw new UnsupportedOperationException();
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
					throw new UnsupportedOperationException();
				}

				/**
				 * Get all key/value pairs of the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return An array containing the currently defined subkeys
				 */
				public static Map<String, JsonValue> getAll(ActorRef actor, String submap) {
					throw new UnsupportedOperationException();
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
					throw new UnsupportedOperationException();
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
					throw new UnsupportedOperationException();
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
					throw new UnsupportedOperationException();
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
					throw new UnsupportedOperationException();
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
					throw new UnsupportedOperationException();
				}

				/**
				 * Remove all values from a submap in the Actor's state.
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return The number of removed subkey entrys
				 */
				public static int removeAll(ActorRef actor, String submap) {
					throw new UnsupportedOperationException();
				}

				/**
				 * Get the keys of the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return An array containing the currently defined subkeys
				 */
				public static String[] keys(ActorRef actor, String submap) {
					throw new UnsupportedOperationException();
				}

				/**
				 * Get the number of keys in the given submap
				 *
				 * @param actor  The Actor instance
				 * @param submap The name of the submap
				 * @return The number of currently define keys in the submap
				 */
				public static int size(ActorRef actor, String submap) {
					throw new UnsupportedOperationException();
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
