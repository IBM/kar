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
package com.ibm.research.kar.quarkus;

import org.jboss.logging.Logger;
import java.util.concurrent.CompletionStage;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;
import javax.ws.rs.core.Response;
import java.util.Map;
import io.smallrye.mutiny.Uni;
import io.vertx.core.http.HttpVersion;
import io.vertx.ext.web.client.WebClientOptions;
import io.vertx.mutiny.core.MultiMap;
import io.vertx.mutiny.core.Vertx;
import io.vertx.mutiny.core.buffer.Buffer;
import io.vertx.mutiny.ext.web.client.HttpRequest;
import io.vertx.mutiny.ext.web.client.HttpResponse;
import io.vertx.mutiny.ext.web.client.WebClient;

public class KarSidecar {

    private static final Logger LOG = Logger.getLogger(KarSidecar.class);

    private final static String KAR_API_CONTEXT_ROOT = "/kar/v1";
    public final static String KAR_QUERYPARAM_SESSION_NAME = "session";

    private static KarHttpClient karClient = new KarHttpClient();

    public void close() throws Exception {
        // TODO Auto-generated method stub

    }

    public Response tellDelete(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callDelete(path, KarSidecar.getAsyncServiceHeaders());
        uni.subscribe();

        return Response.ok().build();
    }

    
    public Response tellPatch(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPatch(path, params, KarSidecar.getAsyncServiceHeaders());
   
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response tellPost(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPost(path, params, KarSidecar.getAsyncServiceHeaders(), null);
  
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response tellPut(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPut(path, params, KarSidecar.getAsyncServiceHeaders());
   
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response callDelete(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callDelete(path, KarSidecar.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response callGet(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callGet(path, null, KarSidecar.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response callHead(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callHead(path, KarSidecar.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response callOptions(String service, String path, JsonValue params) {
        throw new UnsupportedOperationException();
    }

    
    public Response callPatch(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPatch(path, params, KarSidecar.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response callPost(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPost(path, params, KarSidecar.getStandardServiceHeaders(), null);

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response callPut(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPut(path, params, KarSidecar.getStandardServiceHeaders());

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public CompletionStage<Response> callAsyncDelete(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callDelete(path, KarSidecar.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public CompletionStage<Response> callAsyncGet(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callGet(path, null, KarSidecar.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public CompletionStage<Response> callAsyncHead(String service, String path) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callHead(path, KarSidecar.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public CompletionStage<Response> callAsyncOptions(String service, String path, JsonValue params) {
        throw new UnsupportedOperationException();
    }

    
    public CompletionStage<Response> callAsyncPatch(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPatch(path, params, KarSidecar.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public CompletionStage<Response> callAsyncPost(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPost(path, params, KarSidecar.getStandardServiceHeaders(), null);

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public CompletionStage<Response> callAsyncPut(String service, String path, JsonValue params) {
        path = KarSidecar.getServicePath(service, path);
        Uni<Response> uni = karClient.callPut(path, params, KarSidecar.getStandardServiceHeaders());

        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public Response actorTell(String type, String id, String path, JsonArray args) {
        path = KarSidecar.getActorPath(type, id, path);
        Uni<Response> uni = karClient.callPost(path, args, KarSidecar.getAsyncActorHeaders(), null);
        
        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public Response actorCall(String type, String id, String path, String session, JsonArray args) {
        path = KarSidecar.getActorPath(type, id, path);
        Uni<Response> uni = karClient.callPost(path, args, KarSidecar.getStandardActorHeaders(), session);

        return (Response) uni.subscribeAsCompletionStage().join();
    }

    
    public CompletionStage<Response> actorCallAsync(String type, String id, String path, String session,
            JsonArray args) {

        path = KarSidecar.getActorPath(type, id, path);
        Uni<Response> uni = karClient.callPost(path, args, KarSidecar.getStandardActorHeaders(), session);
        
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    
    public Response actorCancelReminders(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorCancelReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetReminders(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorScheduleReminder(String type, String id, String reminderId, JsonObject params) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorHeadWithSubkeyState(String type, String id, String key, String subkey) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorSetWithSubkeyState(String type, String id, String key, String subkey, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorDeleteWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetState(String type, String id, String key, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorHeadState(String type, String id, String key) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorSetState(String type, String id, String key, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorSubmapOp(String type, String id, String key, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorDeleteState(String type, String id, String key, boolean nilOnAbsent) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetAllState(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorUpdate(String type, String id, JsonValue params) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorDeleteAllState(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorDelete(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetAllSubscriptions(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorCancelAllSubscriptions(String type, String id) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorGetSubscription(String type, String id, String subscriptionId) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorCancelSubscription(String type, String id, String subscriptionId) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response actorSubscribe(String type, String id, String subscriptionId, JsonValue data) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response eventCreateTopic(String topic, JsonValue configuration) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response eventDeleteTopic(String topic) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response eventPublish(String topic, JsonValue event) {
        // TODO Auto-generated method stub
        return null;
    }

    
    public Response shutdown() {
        // TODO Auto-generated method stub
        return null;
    }

    
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
        return KarSidecar.KAR_API_CONTEXT_ROOT + "/service/" + service + "/call/" + path;
    }

    private static String getActorPath(String type, String id, String path) {
        path = KarSidecar.KAR_API_CONTEXT_ROOT + "/actor/" + type + "/" + "id" + "/call/" + path;
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
        MultiMap headers = KarSidecar.getStandardServiceHeaders();
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
        MultiMap headers = KarSidecar.getStandardActorHeaders();
        headers.add("PRAGMA", "async");

        return headers;
    }

    public static class KarHttpClient {

        public static final int KAR_SERVICE_REQUEST = 0;
        public static final int KAR_ACTOR_REQUEST = 1;
        public static final int KAR_ACTOR_REMINDER_REQUEST = 2;
        public static final int KAR_ACTOR_STATE_REQUEST = 3;
        public static final int KAR_ACTOR_EVENTS_REQUEST = 4;
        public static final int KAR_SYSTEM_REQUEST = 5;
    
        public static final String HTTP_DELETE = "delete";
        public static final String HTTP_GET = "get";
        public static final String HTTP_HEAD = "head";
        public static final String HTTP_OPTION = "option";
        public static final String HTTP_PATCH = "path";
        public static final String HTTP_POST = "post";
        public static final String HTTP_PUT = "put";
    
        private final static String KAR_DEFAULT_SIDECAR_HOST = "127.0.0.1";
        private final static int KAR_DEFAULT_SIDECAR_PORT = 3000;
    
        Vertx vertx = Vertx.vertx();
    
        private WebClient client = instantiateClient();
    
        private WebClient instantiateClient() {
    
            // read KAR port from env
            int karPort = KarHttpClient.KAR_DEFAULT_SIDECAR_PORT;
            String karPortStr = System.getenv("KAR_RUNTIME_PORT");
            if (karPortStr != null) {
                try {
                    karPort = Integer.parseInt(karPortStr);
                } catch (NumberFormatException ex) {
                    LOG.debug("Warning: value " + karPortStr
                            + "from env variable KAR_RUNTIME_PORT is not an int, using default value "
                            + KarHttpClient.KAR_DEFAULT_SIDECAR_PORT);
                    ex.printStackTrace();
                }
            }
    
            LOG.info("Using KAR port " + karPort);
            // configure client with sidecar and port coordinates
            WebClientOptions options = new WebClientOptions().setDefaultHost(KarHttpClient.KAR_DEFAULT_SIDECAR_HOST)
                    .setDefaultPort(karPort);
    
            String useHttp2 = System.getProperty("kar.http.http2");
            LOG.info("Property useHttp2 = " + useHttp2);
            if ((useHttp2 != null) && (useHttp2.equalsIgnoreCase("true"))) {
                LOG.info("Configuring for HTTP/2");
                options.setProtocolVersion(HttpVersion.HTTP_2).setUseAlpn(true).setHttp2ClearTextUpgrade(false);
            } else {
                LOG.info("Using HTTP/1");
            }
    
            return WebClient.create(vertx, options);
        }
    
        /**
         * 
         * HTTP REST methods
         * 
         */
    
        /**
         * Service GET call
         * 
         * @param service name of service
         * @param path    path to call
         * @return
         */
        public Uni<Response> callDelete(String path, MultiMap headers) {
    
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_GET, path, headers);
            return request.send().onItem().transform(resp -> {
                return convertResponse(resp);
            });
        }
    
        /**
         * Service GET call
         * 
         * @param service name of service
         * @param path    path to call
         * @param params  JSON params
         * @return
         */
        public Uni<Response> callGet(String path, MultiMap params, MultiMap headers) {
    
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_GET, path, headers);
    
            // add queryparams
            if (params != null) {
                for (Map.Entry<String, String> entry : params.entries()) {
                    request.addQueryParam(entry.getKey(), entry.getValue());
                }
            }
    
            return request.send().onItem().transform(resp -> {
                return convertResponse(resp);
            });
        }
    
        /**
         * Service HEAD call
         * 
         * @param service name of service
         * @param path    path to call
         * @param params  JSON Params
         * @return
         */
        public Uni<Response> callHead(String path, MultiMap headers) {
    
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_HEAD, path, headers);
    
            return request.send().onItem().transform(resp -> {
                return convertResponse(resp);
            });
        }
    
        /**
         * Service POST call
         * 
         * @param service name of service
         * @param path    path to call
         * @param params  JSON Params
         * @return
         */
        public Uni<Response> callPatch(String path, Object params, MultiMap headers) {
    
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_PATCH, path, headers);
    
            if (params != null) {
                return request.sendBuffer(Buffer.buffer(params.toString())).onItem().transform(resp -> {
                    return convertResponse(resp);
                });
            } else {
                return request.send().onItem().transform(resp -> {
                    return convertResponse(resp);
                });
            }
        }
    
        /**
         * Service POST call
         * 
         * @param service name of service
         * @param path    path to call
         * @param params  JSON Params
         * @return
         */
        public Uni<Response> callPost(String path, Object params, MultiMap headers, String session) {
    
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_POST, path, headers);
    
            if (session != null) {
                request.addQueryParam(KarSidecar.KAR_QUERYPARAM_SESSION_NAME, session);
            }
    
            if (params != null) {
                return request.sendBuffer(Buffer.buffer(params.toString())).onItem().transform(resp -> {
                    return convertResponse(resp);
                });
            } else {
                return request.send().onItem().transform(resp -> {
                    return convertResponse(resp);
                });
            }
        }
    
        /**
         * Service PUT call
         * 
         * @param service name of service
         * @param path    path to call
         * @param params  JSON Params
         * @return
         */
        public Uni<Response> callPut(String path, Object params, MultiMap headers) {
    
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_PUT, path, headers);
            
            if (params != null) {
                return request.sendBuffer(Buffer.buffer(params.toString())).onItem().transform(resp -> {
                    return convertResponse(resp);
                });
            } else {
                return request.send().onItem().transform(resp -> {
                    return convertResponse(resp);
                });
            }
        }
    
        public HttpRequest<Buffer> httpCall(String method, String path, MultiMap headers)
                throws UnsupportedOperationException {
    
            HttpRequest<Buffer> request = null;
    
            switch (method.toLowerCase()) {
                case KarHttpClient.HTTP_DELETE:
                    request = this.client.delete(path);
                    break;
                case KarHttpClient.HTTP_GET:
                    request = this.client.get(path);
                    break;
                case KarHttpClient.HTTP_HEAD:
                    request = this.client.head(path);
                    break;
                case KarHttpClient.HTTP_PATCH:
                    request = this.client.patch(path);
                    break;
                case KarHttpClient.HTTP_POST:
                    request = this.client.post(path);
                    break;
                case KarHttpClient.HTTP_PUT:
                    request = this.client.put(path);
                    break;
                default:
                    throw new UnsupportedOperationException("Unknown method type " + method + "in http call request");
            }
    
            request.putHeaders(headers);
    
            return request;
    
        }
    
        /**
         * Convert vert.x response to javax.rs Response
         * 
         * @param response vert.x Response object
         * @return javax rs response object
         */
        private Response convertResponse(HttpResponse<Buffer> response) {
    
            return Response.status(response.statusCode()).entity(response.body()).build();
    
        }
    }
}
