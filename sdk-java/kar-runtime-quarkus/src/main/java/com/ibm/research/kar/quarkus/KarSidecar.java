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

import java.util.Map;

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.runtime.KarResponse;

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
import io.vertx.mutiny.ext.web.client.predicate.ErrorConverter;
import io.vertx.mutiny.ext.web.client.predicate.ResponsePredicate;

public class KarSidecar {

    private static final Logger LOG = Logger.getLogger(KarSidecar.class);

    private final static String KAR_API_CONTEXT_ROOT = "/kar/v1";
    private final static String KAR_QUERYPARAM_SESSION_NAME = "session";

    private static KarHttpClient karClient = new KarHttpClient();

    private static String CONTENT_JSON = "application/json; charset=utf-8";

    public Uni<HttpResponse<Buffer>> tellDelete(String service, String path) {
        path = buildServicePath(service, path);
        return karClient.callDelete(path, headers(true));
    }

    public Uni<HttpResponse<Buffer>> tellPatch(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        return karClient.callPatch(path, params, headers(CONTENT_JSON, true));
    }

    public Uni<HttpResponse<Buffer>> tellPost(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        return karClient.callPost(path, params, headers(CONTENT_JSON, true));
    }

    public Uni<HttpResponse<Buffer>> tellPut(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        return karClient.callPut(path, params, headers(CONTENT_JSON, true));
    }

    public Uni<HttpResponse<Buffer>> callDelete(String service, String path) {
        path = buildServicePath(service, path);
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> callGet(String service, String path) {
        path = buildServicePath(service, path);
        return karClient.callGet(path, null, headers(false));
    }

    public Uni<HttpResponse<Buffer>> callHead(String service, String path) {
        path = buildServicePath(service, path);
        return karClient.callHead(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> callOptions(String service, String path, JsonValue params) {
        // TODO: Should be able to do the low-level hhtpCall with the OPTIONS method
        return Uni.createFrom().failure(new UnsupportedOperationException());
    }

    public Uni<HttpResponse<Buffer>> callPatch(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        return karClient.callPatch(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> callPost(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        return karClient.callPost(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> callPut(String service, String path, JsonValue params) {
        path = buildServicePath(service, path);
        return karClient.callPut(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> actorTell(String type, String id, String path, JsonArray args) {
        path = buildActorPath(type, id, "call/"+path);
        return karClient.callPost(path, args, headers(KarResponse.KAR_ACTOR_JSON, true));
    }

    public Uni<HttpResponse<Buffer>> actorCall(String type, String id, String path, String session, JsonArray args) {
        path = buildActorPath(type, id, "call/"+path);
        return karClient.callPost(path, args, headers(KarResponse.KAR_ACTOR_JSON, false), session);
    }

    public Uni<HttpResponse<Buffer>> actorCancelReminders(String type, String id) {
        String path = buildActorPath(type, id, "reminders");
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorCancelReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "reminders/"+reminderId);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        return karClient.callDelete(path, queryParamMap, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorGetReminders(String type, String id) {
        String path = buildActorPath(type, id, "reminders");
        return karClient.callGet(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorGetReminder(String type, String id, String reminderId, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "reminders/"+reminderId);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        return karClient.callGet(path, queryParamMap, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorScheduleReminder(String type, String id, String reminderId, JsonObject params) {
        String path = buildActorPath(type, id, "reminders/"+reminderId);
        return karClient.callPut(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> actorGetWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        return karClient.callGet(path, queryParamMap, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorHeadWithSubkeyState(String type, String id, String key, String subkey) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        return karClient.callHead(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorSetWithSubkeyState(String type, String id, String key, String subkey, JsonValue params) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        return karClient.callPut(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> actorDeleteWithSubkeyState(String type, String id, String key, String subkey, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key + "/" + subkey);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        return karClient.callDelete(path, queryParamMap, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorGetState(String type, String id, String key, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        return karClient.callGet(path, queryParamMap, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorHeadState(String type, String id, String key) {
        String path = buildActorPath(type, id, "state/" + key);
        return karClient.callHead(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorSetState(String type, String id, String key, JsonValue params) {
        String path = buildActorPath(type, id, "state/" + key);
        return karClient.callPut(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> actorSubmapOp(String type, String id, String key, JsonValue params) {
        String path = buildActorPath(type, id, "state/" + key);
        return karClient.callPost(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> actorDeleteState(String type, String id, String key, boolean nilOnAbsent) {
        String path = buildActorPath(type, id, "state/" + key);
        Map<String, String> queryParamMap = Map.of("nilOnAbsent", Boolean.toString(nilOnAbsent));
        return karClient.callDelete(path, queryParamMap, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorGetAllState(String type, String id) {
        String path = buildActorPath(type, id, "state");
        return karClient.callGet(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorUpdate(String type, String id, JsonValue params) {
        String path = buildActorPath(type, id, "state");
        return karClient.callPost(path, params, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> actorDeleteAllState(String type, String id) {
        String path = buildActorPath(type, id, "state");
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorDelete(String type, String id) {
        String path = buildActorPath(type, id);
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorGetAllSubscriptions(String type, String id) {
        String path = buildActorPath(type, id, "events");
        return karClient.callGet(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorCancelAllSubscriptions(String type, String id) {
        String path = buildActorPath(type, id, "events");
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorGetSubscription(String type, String id, String subscriptionId) {
        String path = buildActorPath(type, id, "events/"+subscriptionId);
        return karClient.callGet(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorCancelSubscription(String type, String id, String subscriptionId) {
        String path = buildActorPath(type, id, "events/"+subscriptionId);
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> actorSubscribe(String type, String id, String subscriptionId, JsonValue data) {
        String path = buildActorPath(type, id, "events/"+subscriptionId);
        return karClient.callPut(path, data, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> eventCreateTopic(String topic, JsonValue configuration) {
        String path = buildEventTopicPath(topic);
        return karClient.callPut(path, configuration, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> eventDeleteTopic(String topic) {
        String path = buildEventTopicPath(topic);
        return karClient.callDelete(path, headers(false));
    }

    public Uni<HttpResponse<Buffer>> eventPublish(String topic, JsonValue event) {
        String path = buildEventPublishPath(topic);
        return karClient.callPost(path, event, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> shutdown() {
        String path = getSystemShutdownPath();
        return karClient.callPost(path, headers(CONTENT_JSON, false));
    }

    public Uni<HttpResponse<Buffer>> systemInformation(String component) {
        String path = getSystemInformationPath(component);
        return karClient.callGet(path, headers(false));
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

            String useHttp1 = System.getProperty("kar.http.http1");
            LOG.info("Property useHttp1 = " + useHttp1);
            if ((useHttp1 != null) && (useHttp1.equalsIgnoreCase("true"))) {
                LOG.info("Using HTTP/1 with max pool of 32");
                options.setMaxPoolSize(32); //  bigger than the default of 5, but still a bottleneck
            } else {
                LOG.info("Configuring for HTTP/2");
                options.setProtocolVersion(HttpVersion.HTTP_2).setUseAlpn(true).setHttp2ClearTextUpgrade(false);
            }

            return WebClient.create(vertx, options);
        }

        public static class KarSidecarError extends Throwable {
            public final int statusCode;

            public KarSidecarError(HttpResponse<Buffer> response) {
                super(extractMessage(response));
                this.statusCode = response.statusCode();
            }

            private static String extractMessage(HttpResponse<Buffer> response) {
                String msg = response.statusMessage();
                String body = response.bodyAsString();
                if (body != null && !body.isBlank()) {
                    msg = msg + ": "+ body;
                }
                return msg;
            }
        }

        /*
         *
         * HTTP REST methods
         *
         */

        public Uni<HttpResponse<Buffer>> callDelete(String uri, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_DELETE, uri, headers);
            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callDelete(String uri, Map<String, String> qparams, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_DELETE, uri, headers);

            if ((qparams != null) && (qparams.size() != 0)) {
                for (Map.Entry<String, String> entry : qparams.entrySet()) {
                    request.addQueryParam(entry.getKey(), entry.getValue());
                }
            }

            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callGet(String uri, Map<String, String> qparams, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_GET, uri, headers);

            if ((qparams != null) && (qparams.size() != 0)) {
                for (Map.Entry<String, String> entry : qparams.entrySet()) {
                    request.addQueryParam(entry.getKey(), entry.getValue());
                }
            }

            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callGet(String uri, MultiMap headers) {
            return callGet(uri, null, headers);
        }

        public Uni<HttpResponse<Buffer>> callHead(String uri, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_HEAD, uri, headers);
            return request.send();
        }

        public Uni<HttpResponse<Buffer>> callPatch(String uri, Object body, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_PATCH, uri, headers);
            return body == null ? request.send() : request.sendBuffer(Buffer.buffer(body.toString()));
        }

        public Uni<HttpResponse<Buffer>> callPost(String uri, Object body, MultiMap headers, String session) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_POST, uri, headers);
            if (session != null) {
                request.addQueryParam(KAR_QUERYPARAM_SESSION_NAME, session);
            }
            return body == null ? request.send() : request.sendBuffer(Buffer.buffer(body.toString()));
        }

        public Uni<HttpResponse<Buffer>> callPost(String uri, Object body, MultiMap headers) {
            return callPost(uri, body, headers, null);
        }

        public Uni<HttpResponse<Buffer>> callPost(String uri, MultiMap headers, String session) {
            return callPost(uri, null, headers, session);
        }

        public Uni<HttpResponse<Buffer>> callPost(String uri, MultiMap headers) {
            return callPost(uri, null, headers, null);
        }

        public Uni<HttpResponse<Buffer>> callPut(String uri, Object body, MultiMap headers) {
            HttpRequest<Buffer> request = httpCall(KarHttpClient.HTTP_PUT, uri, headers);
            return body == null ? request.send() : request.sendBuffer(Buffer.buffer(body.toString()));
        }

        public HttpRequest<Buffer> httpCall(String method, String uri, MultiMap headers) throws UnsupportedOperationException {
            HttpRequest<Buffer> request = null;

            switch (method.toLowerCase()) {
                case KarHttpClient.HTTP_DELETE:
                    request = this.client.delete(uri);
                    break;
                case KarHttpClient.HTTP_GET:
                    request = this.client.get(uri);
                    break;
                case KarHttpClient.HTTP_HEAD:
                    request = this.client.head(uri);
                    break;
                case KarHttpClient.HTTP_PATCH:
                    request = this.client.patch(uri);
                    break;
                case KarHttpClient.HTTP_POST:
                    request = this.client.post(uri);
                    break;
                case KarHttpClient.HTTP_PUT:
                    request = this.client.put(uri);
                    break;
                default:
                    throw new UnsupportedOperationException("Unknown method type " + method + "in http call request");
            }

            request.putHeaders(headers);

            ErrorConverter ec = ErrorConverter.create(result -> new KarSidecarError(result.response()));
            request.expect(ResponsePredicate.create(ResponsePredicate.SC_SUCCESS, ec));

            return request;
        }
    }
}
