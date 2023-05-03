/*
 * Copyright IBM Corporation 2020,2023
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

import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonValue;

import com.ibm.research.kar.runtime.KarHttpConstants;

import org.jboss.logging.Logger;

import io.smallrye.mutiny.Uni;
import io.vertx.core.http.HttpMethod;
import io.vertx.core.http.HttpVersion;
import io.vertx.ext.web.client.WebClientOptions;
import io.vertx.mutiny.core.Vertx;
import io.vertx.mutiny.core.buffer.Buffer;
import io.vertx.mutiny.ext.web.client.HttpRequest;
import io.vertx.mutiny.ext.web.client.HttpResponse;
import io.vertx.mutiny.ext.web.client.WebClient;

public class KarSidecar {

    private static final Logger LOG = Logger.getLogger(KarSidecar.class);

    private final static String KAR_API_CONTEXT_ROOT = "/kar/v1";

    private final static String KAR_QUERYPARAM_SESSION_NAME = "session";
    private final static String HEADER_CONTENT_TYPE = "Content-Type";
    private final static String HEADER_PRAGMA = "PRAGMA";
    private final static String HEADER_ASYNC = "async";
    private static String CONTENT_JSON = "application/json; charset=utf-8";

    private static KarHttpClient karClient = new KarHttpClient();

    public Uni<HttpResponse<Buffer>> tellDelete(String service, String path) {
        String uri = buildServiceUri(service, path);
        return karClient.delete(uri).putHeader(HEADER_PRAGMA, HEADER_ASYNC).send();
    }

    public Uni<HttpResponse<Buffer>> tellPatch(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.patch(uri)
            .putHeader(HEADER_PRAGMA, HEADER_ASYNC)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> tellPost(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.post(uri)
            .putHeader(HEADER_PRAGMA, HEADER_ASYNC)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> tellPut(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.put(uri)
            .putHeader(HEADER_PRAGMA, HEADER_ASYNC)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> callDelete(String service, String path) {
        String uri = buildServiceUri(service, path);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> callGet(String service, String path) {
        String uri = buildServiceUri(service, path);
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> callHead(String service, String path) {
        String uri = buildServiceUri(service, path);
        return karClient.head(uri).send();
    }

    public Uni<HttpResponse<Buffer>> callOptions(String service, String path) {
        String uri = buildServiceUri(service, path);
        return karClient.options(uri).send();
    }

    public Uni<HttpResponse<Buffer>> callOptions(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.options(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> callPatch(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.patch(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> callPost(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> callPut(String service, String path, JsonValue params) {
        String uri = buildServiceUri(service, path);
        return karClient.put(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorTell(String type, String id, String path, JsonArray args) {
        String uri = buildActorUri(type, id, "call/" + path);
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, KarHttpConstants.KAR_ACTOR_JSON)
            .putHeader(HEADER_PRAGMA, HEADER_ASYNC)
            .sendBuffer(Buffer.buffer(args.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorCall(String type, String id, String path, JsonArray args) {
        String uri = buildActorUri(type, id, "call/"+path);
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, KarHttpConstants.KAR_ACTOR_JSON)
            .sendBuffer(Buffer.buffer(args.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorCall(String type, String id, String path, String session, JsonArray args) {
        String uri = buildActorUri(type, id, "call/"+path);
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, KarHttpConstants.KAR_ACTOR_JSON)
            .addQueryParam(KAR_QUERYPARAM_SESSION_NAME, session)
            .sendBuffer(Buffer.buffer(args.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorCancelReminders(String type, String id) {
        String uri = buildActorUri(type, id, "reminders");
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorCancelReminder(String type, String id, String reminderId) {
        String uri = buildActorUri(type, id, "reminders/"+reminderId);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorGetReminders(String type, String id) {
        String uri = buildActorUri(type, id, "reminders");
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorGetReminder(String type, String id, String reminderId) {
        String uri = buildActorUri(type, id, "reminders/"+reminderId);
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorScheduleReminder(String type, String id, String reminderId, JsonObject params) {
        String uri = buildActorUri(type, id, "reminders/"+reminderId);
        return karClient.put(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorGetWithSubkeyState(String type, String id, String key, String subkey) {
        String uri = buildActorUri(type, id, "state/" + key + "/" + subkey);
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorHeadWithSubkeyState(String type, String id, String key, String subkey) {
        String uri = buildActorUri(type, id, "state/" + key + "/" + subkey);
        return karClient.head(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorSetWithSubkeyState(String type, String id, String key, String subkey, JsonValue params) {
        String uri = buildActorUri(type, id, "state/" + key + "/" + subkey);
        return karClient.put(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorDeleteWithSubkeyState(String type, String id, String key, String subkey) {
        String uri = buildActorUri(type, id, "state/" + key + "/" + subkey);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorGetState(String type, String id, String key) {
        String uri = buildActorUri(type, id, "state/" + key);
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorHeadState(String type, String id, String key) {
        String uri = buildActorUri(type, id, "state/" + key);
        return karClient.head(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorSetState(String type, String id, String key, JsonValue params) {
        String uri = buildActorUri(type, id, "state/" + key);
        return karClient.put(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorSubmapOp(String type, String id, String key, JsonValue params) {
        String uri = buildActorUri(type, id, "state/" + key);
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorDeleteState(String type, String id, String key) {
        String uri = buildActorUri(type, id, "state/" + key);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorGetAllState(String type, String id) {
        String uri = buildActorUri(type, id, "state");
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorUpdate(String type, String id, JsonValue params) {
        String uri = buildActorUri(type, id, "state");
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(params.toString()));
    }

    public Uni<HttpResponse<Buffer>> actorDeleteAllState(String type, String id) {
        String uri = buildActorUri(type, id, "state");
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorDelete(String type, String id) {
        String uri = buildActorUri(type, id);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorGetAllSubscriptions(String type, String id) {
        String uri = buildActorUri(type, id, "events");
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorCancelAllSubscriptions(String type, String id) {
        String uri = buildActorUri(type, id, "events");
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorGetSubscription(String type, String id, String subscriptionId) {
        String uri = buildActorUri(type, id, "events/"+subscriptionId);
        return karClient.get(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorCancelSubscription(String type, String id, String subscriptionId) {
        String uri = buildActorUri(type, id, "events/"+subscriptionId);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> actorSubscribe(String type, String id, String subscriptionId, JsonValue data) {
        String uri = buildActorUri(type, id, "events/"+subscriptionId);
        return karClient.put(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(data.toString()));
    }

    public Uni<HttpResponse<Buffer>> eventCreateTopic(String topic, JsonValue configuration) {
        String uri = buildEventTopicUri(topic);
        return karClient.put(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(configuration.toString()));
    }

    public Uni<HttpResponse<Buffer>> eventDeleteTopic(String topic) {
        String uri = buildEventTopicUri(topic);
        return karClient.delete(uri).send();
    }

    public Uni<HttpResponse<Buffer>> eventPublish(String topic, JsonValue event) {
        String uri = buildEventPublishUri(topic);
        return karClient.post(uri)
            .putHeader(HEADER_CONTENT_TYPE, CONTENT_JSON)
            .sendBuffer(Buffer.buffer(event.toString()));
    }

    public Uni<HttpResponse<Buffer>> shutdown() {
        String uri = buildSystemShutdownUri();
        return karClient.post(uri).send();
    }

    public Uni<HttpResponse<Buffer>> systemInformation(String component) {
        String uri = buildSystemInformationUri(component);
        return karClient.get(uri).send();
    }

    /*
     * Helpers to construct sidecar URIs
     */
    private static String buildServiceUri(String service, String path) {
        return KAR_API_CONTEXT_ROOT + "/service/" + service + "/call/" + path;
    }

    private static String buildActorUri(String type, String id, String suffix) {
        return buildActorUri(type, id) + "/" + suffix;
    }

    private static String buildActorUri(String type, String id) {
        return KAR_API_CONTEXT_ROOT + "/actor/" + type + "/" + id;
    }

    private static String buildEventTopicUri(String topic) {
        return KAR_API_CONTEXT_ROOT + "/event/" + topic;
    }

    private static String buildEventPublishUri(String topic) {
        return buildEventTopicUri(topic) + "/publish";
    }

    private static String buildSystemShutdownUri() {
        return KAR_API_CONTEXT_ROOT + "/system/shutdown";
    }

    private static String buildSystemInformationUri(String component) {
        return KAR_API_CONTEXT_ROOT + "/system/information/" + component;
    }

    static class KarHttpClient {
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

        HttpRequest<Buffer> delete(String uri) {
            return this.client.delete(uri);
        }

        HttpRequest<Buffer> get(String uri) {
            return this.client.get(uri);
        }

        HttpRequest<Buffer> head(String uri) {
            return this.client.head(uri);
        }

        HttpRequest<Buffer> options(String uri) {
            return this.client.request(HttpMethod.OPTIONS, uri);
        }

        HttpRequest<Buffer> patch(String uri) {
            return this.client.patch(uri);
        }

        HttpRequest<Buffer> post(String uri) {
            return this.client.post(uri);
        }

        HttpRequest<Buffer> put(String uri) {
            return this.client.put(uri);
        }
    }
}
