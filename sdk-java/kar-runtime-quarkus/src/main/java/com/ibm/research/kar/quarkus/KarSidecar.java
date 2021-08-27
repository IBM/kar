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

import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.util.Map;
import java.util.concurrent.CompletionStage;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar;

import org.jboss.logging.Logger;

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
    private final static String KAR_QUERYPARAM_SESSION_NAME = "session";

    private static KarHttpClient karClient = new KarHttpClient();

    private static String CONTENT_JSON = "application/json; charset=utf-8";

    public HttpResponse<Buffer> tellDelete(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(true));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> tellPatch(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPatch(path, params, headers(CONTENT_JSON, true));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> tellPost(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, params, headers(CONTENT_JSON, true));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> tellPut(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, params, headers(CONTENT_JSON, true));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> callDelete(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> callGet(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, null, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> callHead(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callHead(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> callOptions(String service, String path, JsonValue params) {
        throw new UnsupportedOperationException();
    }

    public HttpResponse<Buffer> callPatch(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPatch(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> callPost(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> callPut(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncDelete(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncGet(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, null, headers(false));
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncHead(String service, String path) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callHead(path, headers(false));
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncOptions(String service, String path, JsonValue params) {
        throw new UnsupportedOperationException();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncPatch(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPatch(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncPost(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public CompletionStage<HttpResponse<Buffer>> callAsyncPut(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public HttpResponse<Buffer> actorTell(String type, String id, String path, JsonArray args) {
        path = buildActorPath(type, id, "call/"+path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, args, headers(Kar.KAR_ACTOR_JSON, true));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorCall(String type, String id, String path, String session, JsonArray args) {
        path = buildActorPath(type, id, "call/"+path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, args, headers(Kar.KAR_ACTOR_JSON, false), session);
        return uni.subscribeAsCompletionStage().join();
    }

    public CompletionStage<HttpResponse<Buffer>> actorCallAsync(String type, String id, String path, String session, JsonArray args) {
        path = buildActorPath(type, id, "call/"+path);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, args, headers(Kar.KAR_ACTOR_JSON, false), session);
        return uni.subscribeAsCompletionStage().minimalCompletionStage();
    }

    public HttpResponse<Buffer> actorCancelReminders(String type, String id) {
        String path = buildActorPath(type, id, "reminders");
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorCancelReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "reminders/"+reminderId);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, queryParamMap, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetReminders(String type, String id) {
        String path = buildActorPath(type, id, "reminders");
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, null, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "reminders/"+reminderId);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, queryParamMap, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorScheduleReminder(String type, String id, String reminderId, JsonObject params) {
        String path = buildActorPath(type, id, "reminders/"+reminderId);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, queryParamMap, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorHeadWithSubkeyState(String type, String id, String key, String subkey) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        Uni<HttpResponse<Buffer>> uni = karClient.callHead(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorSetWithSubkeyState(String type, String id, String key, String subkey, JsonValue params) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorDeleteWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, queryParamMap, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetState(String type, String id, String key, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, queryParamMap, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorHeadState(String type, String id, String key) {
        String path = buildActorPath(type, id, "state/" + key);
        Uni<HttpResponse<Buffer>> uni = karClient.callHead(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorSetState(String type, String id, String key, JsonValue params) {
        String path = buildActorPath(type, id, "state/" + key);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorSubmapOp(String type, String id, String key, JsonValue params) {
        String path = buildActorPath(type, id, "state/" + key);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorDeleteState(String type, String id, String key, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, queryParamMap, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetAllState(String type, String id) {
        String path = buildActorPath(type, id, "state");
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorUpdate(String type, String id, JsonValue params) {
        String path = buildActorPath(type, id, "state");
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, params, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorDeleteAllState(String type, String id) {
        String path = buildActorPath(type, id, "state");
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorDelete(String type, String id) {
        String path = buildActorPath(type, id, null);
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetAllSubscriptions(String type, String id) {
        String path = buildActorPath(type, id, "events");
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorCancelAllSubscriptions(String type, String id) {
        String path = buildActorPath(type, id, "events");
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorGetSubscription(String type, String id, String subscriptionId) {
        String path = buildActorPath(type, id, "events/"+subscriptionId);
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorCancelSubscription(String type, String id, String subscriptionId) {
        String path = buildActorPath(type, id, "events/"+subscriptionId);
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> actorSubscribe(String type, String id, String subscriptionId, JsonValue data) {
        String path = buildActorPath(type, id, "events/"+subscriptionId);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, data, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> eventCreateTopic(String topic, JsonValue configuration) {
        String path = buildEventTopicPath(topic);
        Uni<HttpResponse<Buffer>> uni = karClient.callPut(path, configuration, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> eventDeleteTopic(String topic) {
        String path = buildEventTopicPath(topic);
        Uni<HttpResponse<Buffer>> uni = karClient.callDelete(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> eventPublish(String topic, JsonValue event) {
        String path = buildEventPublishPath(topic);
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, event, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> shutdown() {
        String path = getSystemShutdownPath();
        Uni<HttpResponse<Buffer>> uni = karClient.callPost(path, headers(CONTENT_JSON, false));
        return uni.subscribeAsCompletionStage().join();
    }

    public HttpResponse<Buffer> systemInformation(String component) {
        String path = getSystemInformationPath(component);
        Uni<HttpResponse<Buffer>> uni = karClient.callGet(path, headers(false));
        return uni.subscribeAsCompletionStage().join();
    }

    /*
     * Utility calls specific to Quarkus-based KarSidecar
     */

    /*
     * Helpers to construct sidecar URIs
     */
    private static String buildServicePath(String service, String path) {
        return KAR_API_CONTEXT_ROOT + "/service/" + service + "/call/" + path;
    }

    private static String buildActorPath(String type, String id, String suffix) {
        return buildActorPath(type, id) + "/" + suffix;
    }

    private static String buildActorPath(String type, String id) {
        try {
            id = URLEncoder.encode(id, "UTF-8");
        } catch (UnsupportedEncodingException ex) {
            ex.printStackTrace();
        }
        return KAR_API_CONTEXT_ROOT + "/actor/" + type + "/" + id;
    }

    private static String buildEventTopicPath(String topic) {
        return KAR_API_CONTEXT_ROOT + "/event/" + topic;
    }

    private static String buildEventPublishPath(String topic) {
        return buildEventTopicPath(topic) + "/publish";
    }

    private static String getSystemShutdownPath() {
        return KAR_API_CONTEXT_ROOT + "/system/shutdown";
    }

    private static String getSystemInformationPath(String component) {
        return KAR_API_CONTEXT_ROOT + "/system/information/" + component;
    }

    /*
     * Helpers to build request headers
     */

    private static MultiMap headers(boolean async) {
        MultiMap headers = MultiMap.caseInsensitiveMultiMap();
        if (async) {
            headers.add("PRAGMA", "async");
        }
        return headers;
    }

    private static MultiMap headers(String contentType, boolean async) {
        MultiMap headers = MultiMap.caseInsensitiveMultiMap();
        headers.add("Content-type", contentType);
        if (async) {
            headers.add("PRAGMA", "async");
        }
        return headers;
    }

    public static class KarHttpClient {
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

        /*
         *
         * HTTP REST methods
         *
         */

        public Uni<HttpResponse<Buffer>> callDelete(String path, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_DELETE, path, headers);
            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callDelete(String path, Map<String, String> qparams, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_DELETE, path, headers);

            if ((qparams != null) && (qparams.size() != 0)) {
                for (Map.Entry<String, String> entry : qparams.entrySet()) {
                    request.addQueryParam(entry.getKey(), entry.getValue());
                }
            }

            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callGet(String path, Map<String, String> qparams, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_GET, path, headers);

            if ((qparams != null) && (qparams.size() != 0)) {
                for (Map.Entry<String, String> entry : qparams.entrySet()) {
                    request.addQueryParam(entry.getKey(), entry.getValue());
                }
            }

            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callGet(String path, MultiMap headers) {
            return callGet(path, null, headers);
        }

        public Uni<HttpResponse<Buffer>> callHead(String path, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_HEAD, path, headers);
            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callPatch(String path, Object body, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_PATCH, path, headers);
            return body == null ? request.send() : request.sendBuffer(Buffer.buffer(body.toString()));
        }

        public Uni<HttpResponse<Buffer>> callPost(String path, Object body, MultiMap headers, String session) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_POST, path, headers);
            if (session != null) {
                request.addQueryParam(KAR_QUERYPARAM_SESSION_NAME, session);
            }
            return body == null ? request.send() : request.sendBuffer(Buffer.buffer(body.toString()));
        }

        public Uni<HttpResponse<Buffer>> callPost(String path, Object body, MultiMap headers) {
            return callPost(path, body, headers, null);
        }

        public Uni<HttpResponse<Buffer>> callPost(String path, MultiMap headers, String session) {
            return callPost(path, null, headers, session);
        }

        public Uni<HttpResponse<Buffer>> callPost(String path, MultiMap headers) {
            return callPost(path, null, headers, null);
        }

        public Uni<HttpResponse<Buffer>> callPut(String path, Object body, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_PUT, path, headers);
            return body == null ? request.send() : request.sendBuffer(Buffer.buffer(body.toString()));
        }

        public HttpRequest<Buffer> httpCall(String method, String path, MultiMap headers) throws UnsupportedOperationException {
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
    }
}
