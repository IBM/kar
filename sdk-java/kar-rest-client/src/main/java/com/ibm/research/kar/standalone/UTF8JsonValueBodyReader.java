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

import java.io.IOException;
import java.io.InputStream;
import java.lang.annotation.Annotation;
import java.lang.reflect.Type;
import java.nio.charset.Charset;

import javax.json.Json;
import javax.json.JsonReader;
import javax.json.JsonReaderFactory;
import javax.json.JsonValue;
import javax.ws.rs.WebApplicationException;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import org.glassfish.json.jaxrs.JsonValueBodyReader;

/**
 * This serializer works around a limitation of the org.glassfish.json.jaxrs
 * JsonValueBodyReader, which can cause an exception when parsing 1 character
 * input.  KAR char encoding is always UTF-8
 */
public class UTF8JsonValueBodyReader extends JsonValueBodyReader {

    private final JsonReaderFactory rf = Json.createReaderFactory(null);
	@Override
    public JsonValue readFrom(Class<JsonValue> jsonValueClass,
            Type type, Annotation[] annotations, MediaType mediaType,
            MultivaluedMap<String, String> stringStringMultivaluedMap,
            InputStream inputStream) throws IOException, WebApplicationException {
        try (JsonReader reader = rf.createReader(inputStream, Charset.forName("UTF-8"))) {
            return reader.readValue();
        }
    }
}
