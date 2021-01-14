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

package com.ibm.research.kar.standalone;

import java.net.URI;
import java.util.concurrent.TimeUnit;

import com.ibm.research.kar.KarConfig;
import com.ibm.research.kar.KarRest;

import org.eclipse.microprofile.rest.client.RestClientBuilder;
import org.glassfish.json.jaxrs.JsonValueBodyWriter;

/**
 * This version of the KAR rest client is for applications that are running outside of
 * and application server.  It uses the Apache CXF implememtation of the Microprofile
 * REST client and pulls in org.glassfish.json.jaxrs JsonValue serializers normally provided
 * by the application server.
 */
public class Kar extends com.ibm.research.kar.Kar {

	public static void init() {
		RestClientBuilder builder = RestClientBuilder.newBuilder().baseUri(Kar.getUri());
		// If running in standalone mode, add JsonValue serializers by hand
		if (!Kar.isRunningEmbedded()) {
			builder.register(UTF8JsonValueBodyReader.class).register(JsonValueBodyWriter.class);
		}

		Kar.setRestClient(
				builder.readTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS)
				.connectTimeout(KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS, TimeUnit.MILLISECONDS).build(KarRest.class)
				);

	}

	private static boolean isRunningEmbedded() {
		return (System.getProperty("wlp.server.name") != null);
	}

}
