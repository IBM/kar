/**
 * This version of the KAR rest client is for applications that are running outside of
 * and application server.  It uses the Apache CXF implememtation of the Microprofile
 * REST client and pulls in org.glassfish.json.jaxrs JsonValue serializers normally provided
 * by the application server.
 */
package com.ibm.research.kar.standalone;

import java.net.URI;
import java.util.concurrent.TimeUnit;

import com.ibm.research.kar.KarConfig;
import com.ibm.research.kar.KarRest;
import com.ibm.research.kar.actor.exceptions.ActorExceptionMapper;

import org.eclipse.microprofile.rest.client.RestClientBuilder;
import org.glassfish.json.jaxrs.JsonValueBodyWriter;

public class Kar extends com.ibm.research.kar.Kar {

	public static void init() {
		RestClientBuilder builder = RestClientBuilder.newBuilder().baseUri(Kar.getUri());
		// If running in standalone mode, add JsonValue serializers by hand
		if (!Kar.isRunningEmbedded()) {
			builder.register(UTF8JsonValueBodyReader.class).register(JsonValueBodyWriter.class);
		}

		Kar.setRestClient(
				builder.register(ActorExceptionMapper.class)
				.readTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.connectTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS).build(KarRest.class)
				);

	}

	private static boolean isRunningEmbedded() {
		return (System.getProperty("wlp.server.name") != null);
	}

}
