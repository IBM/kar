// Type definitions for [KAR] [0.1.0]
// Project:[KAR:Kubernetes Application Runtime]

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

import { Server } from "spdy";
import { Application, Router } from "express";

export interface ActorImpl {
  /** The type of this Actor instance */
  type: string;
  /** The id of this Actor instance */
  id: string;
  /** The session of the active invocation */
  session?: string;
}

/**
 * An Actor instance
 */
export interface Actor {
  kar: ActorImpl
}

/**
 * The body of a multi-element actor state update operation
 */
export interface ActorStateUpdate {
  /** Keys to remove from the actor state */
  removals: Array<string>
  /** A mapping from submap names to the keys to remove from each submap */
  submapremovals: Map<string, Array<string>>
  /** The updates to perform to the actors state */
  updates: Map<string, any>
  /** A mapping from submap names to the updates to perform on each submap */
  submapupdates: Map<string, Map<String, any>>
}

/**
 * The result of a multi-element actor state update
 */
export interface ActorStateUpdateResult {
  added: number
  removed: number
}

/**
 * A Reminder
 */
export interface Reminder {
  /** The actor to be reminded */
  actor: ActorImpl;
  /** The id of this reminder */
  id: string;
  /** The time at which the reminder is eligible for delivery */
  targetTime: Date;
  /** The actor method to be invoked */
  path: string;
  /** An array of arguments with which to invoke the target method */
  data?: any[];
  /** Period at which the reminder should recur in nanoseconds. A value of 0 indicates a non-recurring reminder */
  period: number;
}

export interface ScheduleReminderOptions {
  /** The id of the reminder being scheduled */
  id: string;
  /** The earliest time at which the reminder should be delivered */
  targetTime: Date;
  /**  For periodic reminders, a string encoding a Duration representing the desired gap between successive reminders */
  period?: string;
}

/*
 * Events
 */

export interface SubscribeOptions {
  /** The id of the subscription being created */
  id?: string;
  /** The expected MIME content type of events from this subscription.  Defaults to application/json+cloudevent */
  contentType?: string;
}

export interface TopicCreationOptions {
  /** Kafka topic creation config */
  configEntries?: Map<string, string>;
  /** The number of Kafka paritions to create */
  numPartitions?: number;
  /** The replication factor for this topic */
  replicationFactor?: number;
}

/**
 * Asynchronous service invocation; returns "OK" immediately
 * @param service The service to invoke.
 * @param path The service endpoint to invoke.
 * @param body The request body with which to invoke the service endpoint.
 */
export function tell (service: string, path: string, body: any): Promise<any>;

/**
 * Synchronous service invocation; returns invocation result
 * @param service The service to invoke.
 * @param path The service endpoint to invoke.
 * @param body The request body with which to invoke the service endpoint.
 * @returns The result returned by the target service.
 */
export function call (service: string, path: string, body: any): Promise<any>;

/**
 * Actor operations
 */
export namespace actor {

  /**
   * Construct a proxy object that represents an Actor instance.
   * @param type The type of the Actor instance
   * @param id The instance id of the Actor instance
   * @returns A proxy object representing the Actor instance.
   */
  export function proxy (type: string, id: string): Actor;

  /**
   * Asynchronous actor invocation; returns "OK" immediately
   * @param actor The target actor.
   * @param path The actor method to invoke.
   * @param args The arguments with which to invoke the actor method.
   */
  export function tell (callee: Actor, path: string, ...args: any[]): Promise<any>;

  /**
   * Synchronous actor invocation propagating current session; returns the result of the invoked Actor method.
   * @param from The actor making the call
   * @param callee The target actor.
   * @param path The actor method to invoke.
   * @param args The arguments with which to invoke the actor method.
   */
  export function call (from: Actor, callee: Actor, path: string, ...args: any[]): Promise<any>;

  /**
   * Synchronous actor invocation creating a new session; returns the result of the invoked Actor method.
   * @param callee The target Actor.
   * @param path The actor method to invoke.
   * @param args The arguments with which to invoke the actor method.
   */
  export function call (callee: Actor, path: string, ...args: any[]): Promise<any>;

  /**
   * Asynchronously remove all user-level and runtime state of an Actor.
   *
   * @param target The Actor instance.
   */
  export function remove (target: Actor): Promise<any>;

  /**
   * Construct a result object that encodes a tail call to another actor method
   * @param callee The target Actor.
   * @param path The actor method to invoke.
   * @param args The arguments with which to invoke the actor method.
   */
  export function tailCall(callee: Actor, path: string, ...args: any[]): any;

  namespace reminders {
    /**
     * Cancel matching reminders for an Actor instance.
     * @param actor The Actor instance.
     * @param reminderId The id of a specific reminder to cancel
     * @returns The number of reminders that were cancelled.
     */
    export function cancel (actor: Actor, reminderId?: string): Promise<number>;

    /**
     * Get matching reminders for an Actor instance.
     * @param actor The Actor instance.
     * @param reminderId The id of a specific reminder to get
     * @returns An array of matching reminders
     */
    export function get (actor: Actor, reminderId?: string): Promise<Reminder | Array<Reminder>>;

    /**
     * Schedule a reminder for an Actor instance.
     * @param actor The Actor instance.
     * @param path The actor method to invoke when the reminder fires.
     * @param options.id The id of the reminder being scheduled
     * @param options.targetTime The earliest time at which the reminder should be delivered
     * @param options.period For periodic reminders, a string encoding a Duration representing the desired gap between successive reminders
     * @param args The arguments with which to invoke the actor method.
     */
    export function schedule (actor: Actor, path: string, options: ScheduleReminderOptions, ...args: any[]): Promise<any>;
  }

  namespace state {
    /**
     * Get one value from an Actor's state
     * @param actor The Actor instance.
     * @param key The key to get from the instance's state
     * @returns The value associated with `key`
     */
    export function get (actor: Actor, key: string): Promise<any>;

    /**
     * Get all of an Actor's state
     * @param actor The Actor instance.
     * @returns A map representing the Actor's state
     */
    export function getAll (actor: Actor): Promise<Map<string, any>>;

    /**
     * Check to see if an Actor's state contains an entry
     * @param actor The Actor instance.
     * @param key The key to check for in the instance's state
     * @returns `true` if the actor has a state entry for `key` and `false` if it does not
     */
    export function contains (actor: Actor, key: string): Promise<boolean>;

    /**
     * Store one value to an Actor's state
     * @param actor The Actor instance.
     * @param key The key to update in the instance's state
     * @param value The value to store
     */
    export function set (actor: Actor, key: string, value: any): Promise<void>;

    /**
     * Store multiple values to an Actor's state
     * @param actor The Actor instance.
     * @param updates The updates to make
     */
    export function setMultiple (actor: Actor, updates: Map<string, any>): Promise<void>;

    /**
     * Remove a single value from an Actor's state.
     * @param actor The Actor instance.
     * @param key The key to delete
     */
    export function remove (actor: Actor, key: string): Promise<void>;

    /**
     * Remove some values from an Actor's state.
     * @param actor The Actor instance.
     * @param keys The keys to delete
     * @returns the number of removed entries
     */
    export function removeSome (actor: Actor, keys: Array<string>): Promise<number>;

    /**
     * Remove an Actor's state
     * @param actor The Actor instance.
     */
    export function removeAll (actor: Actor): Promise<void>;

    /**
     * Perform a multi-element update operation to an Actor's state
     * @param actor The Actor instance
     * @param changes The collection of removals and updates to perform
     */
    export function update (actor: Actor, changes:ActorStateUpdate): Promise<ActorStateUpdateResult>

    namespace submap {
      /**
       * Get one value from an Actor's state submap
       * @param actor The Actor instance.
       * @param submap The Actor submap to access
       * @param key The key to get from the submap
       * @returns The value associated with `key/subkey`
       */
      export function get (actor: Actor, submap:string, key: string): Promise<any>;

      /**
       * Get the contents of the `key` map of an Actor's state
       * @param actor The Actor instance.
       * @param submap The name of the submap to get
       * @returns The contents of submap
       */
      export function getAll (actor: Actor, submap: String): Promise<Map<string, any>>;

      /**
       * Check to see if an Actor's state contains an entry
       * @param actor The Actor instance.
       * @param submap The Actor submap to access
       * @param key The key to check for in the submap
       * @returns `true` if the actor has a state entry for `key/subkey` and `false` if it does not
       */
      export function contains (actor: Actor, submap:string, key: string): Promise<boolean>;

      /**
       * Store one value to an Actor's state
       * @param actor The Actor instance.
       * @param submap The Actor submap to access
       * @param key The key to update in the submap
       * @param value The value to store
       */
      export function set (actor: Actor, submap:string, key: string, value: any): Promise<void>;

      /**
       * Store multiple (subkey, value) pairs to the `submap` map of an Actor's state
       * @param actor The Actor instance.
       * @param submap The Actor submap to update
       * @param updates The updates to make
       */
      export function setMultiple (actor: Actor, submap: string, updates: Map<string, any>): Promise<void>;

      /**
       * Remove a single value from an Actor submap.
       * @param actor The Actor instance.
       * @param submap The Actor submap to access
       * @param key The key to delete
       */
      export function remove (actor: Actor, submap:string, key: string): Promise<void>;

      /**
       * Remove some values from an Actor submap
       * @param actor The Actor instance.
       * @param submap The Actor submap to access
       * @params keys The keys to remove from the submap
       * @returns the number of removed submap entries
       */
      export function removeSome (actor: Actor, submap: string, keys: Array<string>): Promise<number>;

      /**
       * Remove an Actor submap
       * @param actor The Actor instance.
       * @param submap The Actor submap to remove
       * @returns the number of removed submap entries
       */
      export function removeAll (actor: Actor, submap: string): Promise<number>;

      /**
       * Get the subkeys associated with the given key
       * @param actor The Actor instance
       * @param submap The Actor submap to access
       * @returns An array containing keys defined in the submap
       */
      export function keys (actor: Actor, submap: string): Promise<Array<string>>;

      /**
       * Get the size of a submap
       * @param actor The Actor instance
       * @param submap The Actor submap to access
       * @returns The size of `submap`
       */
      export function size (actor: Actor, submap: string): Promise<number>;
    }
  }
}

export namespace events {
  /**
    * Cancel matching subscriptions for an Actor instance.
    * @param actor The Actor instance.
    * @param subscriptionId The id of a specific subscription to cancel
    * @returns The number of subscriptions that were cancelled.
    */
  export function cancelSubscription (actor: Actor, subscriptionId?: string): Promise<number>;

  /**
   * Get matching subscription(s) for an Actor instance.
   * @param actor The Actor instance.
   * @param subscriptionId The id of a specific subscription to get
   * @returns The matching subscription(s)
   */
  export function getSubscription (actor: Actor, subscriptionId?: string): Promise<Reminder | Array<Reminder>>;

  /**
   * Subscribe an Actor instance method to a topic.
   * @param actor The Actor instance to subscribe
   * @param path The actor method to invoke on each event received on the topic
   * @param topic The topic to which to subscribe
   * @param options.contentType The expected MIME content type of events (defaults to application/json+cloudevents)
   * @param options.id The subscription id; defaults to the topic name.
   */
  export function subscribe (actor: Actor, path: string, topic: string, options: SubscribeOptions): Promise<any>

  /**
   * Create a topic
   * @param topic The name of the topic to create
   * @param options.configEntries A map of kafka topic configuration options
   * @param options.numPartitions The number of partitions for the topic
   * @param options.replicationFactor The replication factor for the topic
   */
  export function createTopic (topic: string, options: TopicCreationOptions): Promise<any>

  /**
   * Delete a topic
   * @param topic the name of the topic to delete
   */
  export function deleteTopic (topic: string): Promise<any>

  /**
   * Publish an event on a topic
   * @param topic
   * @param event
   */
  export function publish (topic: string, event: any): Promise<void>
}


/**
 * Application configuration and system operations
 */
export namespace sys {
  /**
   * Instantiate an actor runtime for this application process by
   * providing it a collection of classes that implement Actor types.
   * @param actors The actor types implemented by this application component.
   * @returns Router that will serve routes designated for the actor runtime
   */
  export function actorRuntime (actors: { [k: string]: () => Object }): Router;

  /**
   * Wrap an Express App in an http/2 server.
   * @param app An Express App
   * @returns a Server
   */
  export function h2c (app: Application): Server;

  /**
   * Query sidecar.
   * @param a query
   * @returns an answer
   */
  export function get (query: string): Promise<any>;

  /**
   * Error handling middleware
   * TODO: proper type & documentation
   */
  export const errorHandler: any;

  /**
   * Kill this sidecar
   */
  export function shutdown (): Promise<void>;
}
