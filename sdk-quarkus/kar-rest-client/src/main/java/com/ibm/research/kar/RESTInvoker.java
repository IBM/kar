package com.ibm.research.kar;

import org.eclipse.microprofile.config.inject.ConfigProperty;
import org.jboss.logging.Logger;

import javax.annotation.PostConstruct;
import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;

import io.smallrye.mutiny.Uni;
import io.vertx.core.http.HttpVersion;
import io.vertx.ext.web.client.WebClientOptions;
import io.vertx.mutiny.core.Vertx;
import io.vertx.mutiny.core.buffer.Buffer;
import io.vertx.mutiny.ext.web.client.WebClient;

@ApplicationScoped
public class RESTInvoker {

    private static final Logger LOG = Logger.getLogger(RESTInvoker.class);

    private final static String KAR_API_CONTEXT_ROOT = "/kar/v1/service";
    private final static String KAR_DEFAULT_SIDECAR_HOST = "127.0.0.1";
    private final static int KAR_DEFAULT_SIDECAR_PORT = 3000;

    @Inject
    Vertx vertx;

    @ConfigProperty(name = "kar.http.http2", defaultValue = "true")
    boolean useHttp2;

    private WebClient client;

    @PostConstruct
    void initialize() {

        // read KAR port from env
        int karPort = RESTInvoker.KAR_DEFAULT_SIDECAR_PORT;
        String karPortStr = System.getenv("KAR_RUNTIME_PORT");
        if (karPortStr != null) {
            try {
                karPort = Integer.parseInt(karPortStr);
            } catch (NumberFormatException ex) {
                LOG.debug("Warning: value " + karPortStr + "from env variable KAR_RUNTIME_PORT is not an int, using default value " + RESTInvoker.KAR_DEFAULT_SIDECAR_PORT);
                ex.printStackTrace();
            } 
        }

        LOG.debug("Using KAR port " + karPort);
       // configure client with sidecar and port coordinates
        WebClientOptions options =  new WebClientOptions()
                .setDefaultHost(RESTInvoker.KAR_DEFAULT_SIDECAR_HOST)
                .setDefaultPort(karPort);

        // Add HTTP2 configuration that skips clear text upgrade
        if (useHttp2 == true) {
            LOG.info("Configuring for HTTP/2");
            options
                .setProtocolVersion(HttpVersion.HTTP_2)
                .setUseAlpn(true)
                .setHttp2ClearTextUpgrade(false);
        } else {
            LOG.info("Using HTTP/1");
        }

        this.client = WebClient.create(vertx, options);
    }

    /*
     * Invoke service that returns a string
     */
    public Uni<String> invokeKar(String path, String message) {
        return (Uni<String>) this.client.post(RESTInvoker.KAR_API_CONTEXT_ROOT + "/" + path)
                .putHeader("content-type", "text/plain;charset=UTF-8").sendBuffer(Buffer.buffer(message)).onItem()
                .transform(resp -> {
                    if (resp.statusCode() == 200) {
                        return resp.bodyAsString();
                    } else {
                        return "Error with status code " + resp.statusCode();
                    }
                });

    }

}
