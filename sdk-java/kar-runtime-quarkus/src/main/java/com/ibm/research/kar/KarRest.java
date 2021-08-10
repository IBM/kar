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

import java.util.concurrent.CompletionStage;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.runtime.KarSidecar;

import io.smallrye.mutiny.Uni;
import io.vertx.mutiny.core.MultiMap;

public class KarRest implements KarSidecar {

    private final static String KAR_API_CONTEXT_ROOT = "/kar/v1";
    public final static String KAR_QUERYPARAM_SESSION_NAME = "session";

    private static KarHttpClient karClient = KarHttpClient.getClient();

    @Override
    public void close() throws Exception {
        // TODO Auto-generated method stub

    }

    @Override
    public Response tellDelete(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callDelete(path, KarRest.getAsyncServiceHeaders());
        uni.subscribe();

        return Response.ok().build();
    }

    @Override
    public Response tellPatch(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPatch(path, params, KarRest.getAsyncServiceHeaders());
   
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response tellPost(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPost(path, params, KarRest.getAsyncServiceHeaders(), null);
  
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response tellPut(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPut(path, params, KarRest.getAsyncServiceHeaders());
   
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response callDelete(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callDelete(path, KarRest.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response callGet(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callGet(path, null, KarRest.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response callHead(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callHead(path, KarRest.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response callOptions(String service, String path, JsonValue params) {
        // TBD Options not supported
        return null;
    }

    @Override
    public Response callPatch(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPatch(path, params, KarRest.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response callPost(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPost(path, params, KarRest.getStandardServiceHeaders(), null);

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response callPut(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPut(path, params, KarRest.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public CompletionStage<Response> callAsyncDelete(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callDelete(path, KarRest.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public CompletionStage<Response> callAsyncGet(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callGet(path, null, KarRest.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public CompletionStage<Response> callAsyncHead(String service, String path) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callHead(path, KarRest.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public CompletionStage<Response> callAsyncOptions(String service, String path, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public CompletionStage<Response> callAsyncPatch(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPatch(path, params, KarRest.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public CompletionStage<Response> callAsyncPost(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPost(path, params, KarRest.getStandardServiceHeaders(), null);

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public CompletionStage<Response> callAsyncPut(String service, String path, JsonValue params) {
        path = KarRest.getServicePath(service, path);
        Uni<Response> uni = karClient.callPut(path, params, KarRest.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public Response actorTell(String type, String id, String path, JsonArray args) {
        path = KarRest.getActorPath(type, id, path);
        Uni<Response> uni = karClient.callPost(path, args, KarRest.getAsyncActorHeaders(), null);
        
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public Response actorCall(String type, String id, String path, String session, JsonArray args) {
        path = KarRest.getActorPath(type, id, path);
        Uni<Response> uni = karClient.callPost(path, args, KarRest.getStandardActorHeaders(), session);

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    @Override
    public CompletionStage<Response> actorCallAsync(String type, String id, String path, String session,
            JsonArray args) {

        path = KarRest.getActorPath(type, id, path);
        Uni<Response> uni = karClient.callPost(path, args, KarRest.getStandardActorHeaders(), session);
        
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    @Override
    public Response actorCancelReminders(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorCancelReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetReminders(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorScheduleReminder(String type, String id, String reminderId, JsonObject params) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorHeadWithSubkeyState(String type, String id, String key, String subkey) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorSetWithSubkeyState(String type, String id, String key, String subkey, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorDeleteWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetState(String type, String id, String key, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorHeadState(String type, String id, String key) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorSetState(String type, String id, String key, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorSubmapOp(String type, String id, String key, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorDeleteState(String type, String id, String key, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetAllState(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorUpdate(String type, String id, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorDeleteAllState(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorDelete(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetAllSubscriptions(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorCancelAllSubscriptions(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorGetSubscription(String type, String id, String subscriptionId) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorCancelSubscription(String type, String id, String subscriptionId) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response actorSubscribe(String type, String id, String subscriptionId, JsonValue data) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response eventCreateTopic(String topic, JsonValue configuration) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response eventDeleteTopic(String topic) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response eventPublish(String topic, JsonValue event) {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response shutdown() {
        // TODO Auto-generated method stub
        return null;
    }

    @Override
    public Response systemInformation(String component) {
        // TODO Auto-generated method stub
        return null;
    }

    /*
     * Utility calls
     */

    /**
     * Construct sidecar URI from service name and path for service calls
     * 
     * @param service
     * @param path
     * @return path component of sidecar REST call
     */
    private static String getServicePath(String service, String path) {
        return KarRest.KAR_API_CONTEXT_ROOT + "/service/" + service + "/call/" + path;
    }

    private static String getActorPath(String type, String id, String path) {
        path = KarRest.KAR_API_CONTEXT_ROOT + "/actor/" + type + "/" + "id" + "/call/" + path;
        return path;
    }

    /**
     * Return headers used by KAR calls for service calls
     * 
     * @return headers
     */
    private static MultiMap getStandardServiceHeaders() {
        MultiMap headers = MultiMap.caseInsensitiveMultiMap();
        headers.add("Content-type", "application/json; charset=utf-8");
        // headers.add("Accept", "application/json; charset=utf-8");
        return headers;
    }

    private static MultiMap getAsyncServiceHeaders() {
        MultiMap headers = KarRest.getStandardServiceHeaders();
        headers.add("PRAGMA", "async");

        return headers;
    }

        /**
     * Return headers used by KAR calls for Actor calls
     * 
     * @return headers
     */
    private static MultiMap getStandardActorHeaders() {
        MultiMap headers = MultiMap.caseInsensitiveMultiMap();
        headers.add("Content-type", Kar.KAR_ACTOR_JSON);
        return headers;
    }

    private static MultiMap getAsyncActorHeaders() {
        MultiMap headers = KarRest.getStandardActorHeaders();
        headers.add("PRAGMA", "async");

        return headers;
    }

}
