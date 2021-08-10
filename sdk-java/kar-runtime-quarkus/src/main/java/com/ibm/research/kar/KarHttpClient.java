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

import org.jboss.logging.Logger;

import java.util.Map;

import javax.annotation.PostConstruct;
import javax.enterprise.context.ApplicationScoped;
import javax.ws.rs.core.Response;

import io.smallrye.mutiny.Uni;
import io.vertx.mutiny.core.MultiMap;
import io.vertx.core.http.HttpVersion;
import io.vertx.ext.web.client.WebClientOptions;
import io.vertx.mutiny.core.Vertx;
import io.vertx.mutiny.core.buffer.Buffer;
import io.vertx.mutiny.ext.web.client.HttpRequest;
import io.vertx.mutiny.ext.web.client.HttpResponse;
import io.vertx.mutiny.ext.web.client.WebClient;

@ApplicationScoped
public class KarHttpClient {

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

    private static final Logger LOG = Logger.getLogger(KarHttpClient.class);

    private final static String KAR_DEFAULT_SIDECAR_HOST = "127.0.0.1";
    private final static int KAR_DEFAULT_SIDECAR_PORT = 3000;

    Vertx vertx = Vertx.vertx();

    private WebClient client;

    public static KarHttpClient getClient() {
        KarHttpClient client = new KarHttpClient();
        client.initialize();
        return client;
    }

    @PostConstruct
    void initialize() {

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

        this.client = WebClient.create(vertx, options);
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
            request.addQueryParam(KarRest.KAR_QUERYPARAM_SESSION_NAME, session);
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
    private Response convertResponse(HttpResponse response) {

        return Response.status(response.statusCode()).entity(response.body()).build();

    }

}
