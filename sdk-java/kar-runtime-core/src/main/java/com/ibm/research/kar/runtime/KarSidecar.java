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

package com.ibm.research.kar.runtime;

import java.util.concurrent.CompletionStage;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;
import javax.ws.rs.core.Response;

/**
 * KarClient defines the REST operations that the upper-layer of the
 * Kar runtime expect to be able to invoke on its sidecar via a middleware
 * provided REST Client instance.
 */
public interface KarSidecar extends AutoCloseable {

	/*
	 * Services
	 */

	// asynchronous service invocation, returns (202, "OK")
	public Response tellDelete(String service, String path);

	public Response tellPatch(String service, String path, JsonValue params);

	public Response tellPost(String service, String path, JsonValue params);

	public Response tellPut(String service, String path, JsonValue params);

	// synchronous service invocation, returns invocation result
	public Response callDelete(String service, String path);

	public Response callGet(String service, String path);

	public Response callHead(String service, String path);

	public Response callOptions(String service, String path, JsonValue params);

	public Response callPatch(String service, String path, JsonValue params);

	public Response callPost(String service, String path, JsonValue params);

	public Response callPut(String service, String path, JsonValue params);

	// asynchronous service invocation, returns CompletionStage that will contain
	// the eventual invocation result
	public CompletionStage<Response> callAsyncDelete(String service, String path);

	public CompletionStage<Response> callAsyncGet(String service, String path);

	public CompletionStage<Response> callAsyncHead(String service, String path);

	public CompletionStage<Response> callAsyncOptions(String service, String path, JsonValue params);

	public CompletionStage<Response> callAsyncPatch(String service, String path, JsonValue params);

	public CompletionStage<Response> callAsyncPost(String service, String path, JsonValue params);

	public CompletionStage<Response> callAsyncPut(String service, String path, JsonValue params);

	/*
	 * Actors
	 */

	// asynchronous actor invocation, returns (202, "OK")
	public Response actorTell(String type, String id, String path, JsonArray args);

	// synchronous actor invocation: returns invocation result
	public Response actorCall(String type, String id, String path, String session, JsonArray args);

	// asynchronous actor invocation: returns future of invocation result
	public CompletionStage<Response> actorCallAsync(String type, String id, String path, String session, JsonArray args);

	//
	// Actor Reminder operations
	//

	public Response actorCancelReminders(String type, String id);

	public Response actorCancelReminder(String type, String id, String reminderId, boolean nilOnAbsent);

	public Response actorGetReminders(String type, String id);

	public Response actorGetReminder(String type, String id, String reminderId, boolean nilOnAbsent);

	public Response actorScheduleReminder(String type, String id, String reminderId, JsonObject params);

	//
	// Actor state operations
	//

	public Response actorGetWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent);

	public Response actorHeadWithSubkeyState(String type, String id, String key, String subkey);

	public Response actorSetWithSubkeyState(String type, String id, String key, String subkey, JsonValue params);

	public Response actorDeleteWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent);

	public Response actorGetState(String type, String id, String key, boolean nilOnAbsent);

	public Response actorHeadState(String type, String id, String key);

	public Response actorSetState(String type, String id, String key, JsonValue params);

	public Response actorSubmapOp(String type, String id, String key, JsonValue params);

	public Response actorDeleteState(String type, String id, String key, boolean nilOnAbsent);

	public Response actorGetAllState(String type, String id);

	public Response actorUpdate(String type, String id, JsonValue params);

	public Response actorDeleteAllState(String type, String id);

	public Response actorDelete(String type, String id);

	/*
	 * Events
	 */

	public Response actorGetAllSubscriptions(String type, String id);

	public Response actorCancelAllSubscriptions(String type, String id);

	public Response actorGetSubscription(String type, String id, String subscriptionId);

	public Response actorCancelSubscription(String type, String id, String subscriptionId);

	public Response actorSubscribe(String type, String id, String subscriptionId, JsonValue data);

	public Response eventCreateTopic(String topic, JsonValue configuration);

	public Response eventDeleteTopic(String topic);

	public Response eventPublish(String topic, JsonValue event);

	/*
	 * System
	 */

	public Response shutdown();

	public Response systemInformation(String component);
}
