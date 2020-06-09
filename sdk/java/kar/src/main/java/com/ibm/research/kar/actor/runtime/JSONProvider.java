package com.ibm.research.kar.actor.runtime;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.lang.annotation.Annotation;
import java.lang.reflect.Type;

import javax.annotation.PostConstruct;
import javax.json.Json;
import javax.json.JsonReader;
import javax.json.JsonReaderFactory;
import javax.json.JsonValue;
import javax.json.JsonWriter;
import javax.json.JsonWriterFactory;
import javax.ws.rs.WebApplicationException;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import com.ibm.research.kar.KarRest;

@javax.ws.rs.Consumes(KarRest.KAR_ACTOR_JSON)
@javax.ws.rs.Produces(KarRest.KAR_ACTOR_JSON)
@javax.ws.rs.ext.Provider
public class JSONProvider
    implements javax.ws.rs.ext.MessageBodyReader<JsonValue>, javax.ws.rs.ext.MessageBodyWriter<JsonValue> {

  JsonReaderFactory readerFactory;
  JsonWriterFactory writerFactory;

  @PostConstruct
  public void initialize() {
    this.readerFactory = Json.createReaderFactory(null);
    this.writerFactory = Json.createWriterFactory(null);
  }

  @Override
  public boolean isReadable(Class<?> type, Type genericType, Annotation[] annotations, MediaType mediaType) {
    return JsonValue.class.isAssignableFrom(type);
  }

  @Override
  public JsonValue readFrom(Class<JsonValue> type, Type genericType, Annotation[] annotations, MediaType mediaType,
      MultivaluedMap<String, String> httpHeaders, InputStream entityStream)
      throws IOException, WebApplicationException {
    JsonReader reader = readerFactory.createReader(entityStream);
    JsonValue retValue = reader.readValue();
    return retValue;
  }

  @Override
  public boolean isWriteable(Class<?> type, Type genericType, Annotation[] annotations, MediaType mediaType) {
    return JsonValue.class.isAssignableFrom(type);
  }

  @Override
  public void writeTo(JsonValue t, Class<?> type, Type genericType, Annotation[] annotations, MediaType mediaType,
      MultivaluedMap<String, Object> httpHeaders, OutputStream entityStream)
      throws IOException, WebApplicationException {
    JsonWriter writer = writerFactory.createWriter(entityStream);
    writer.write(t);
    writer.close();
  }
}
