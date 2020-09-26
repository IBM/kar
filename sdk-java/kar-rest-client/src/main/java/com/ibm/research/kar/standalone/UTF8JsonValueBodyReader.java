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
