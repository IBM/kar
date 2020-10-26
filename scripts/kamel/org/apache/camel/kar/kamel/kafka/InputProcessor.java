// camel-k: dependency=github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626

package org.apache.camel.kar.kamel.kafka;

import org.apache.camel.BindToRegistry;
import org.apache.camel.Exchange;
import org.apache.camel.Processor;
import org.apache.camel.builder.RouteBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.cloudevents.CloudEvent;
import io.cloudevents.core.builder.CloudEventBuilder;
import io.cloudevents.core.format.EventFormat;
import io.cloudevents.core.provider.EventFormatProvider;

import java.net.URI;

class TransformMessageToCloudEvent implements Processor {
    private static final Logger LOG = LoggerFactory.getLogger(TransformMessageToCloudEvent.class);

    public void process(Exchange exchange) throws Exception {
        String body = exchange.getIn().getBody(String.class);
        LOG.info("Received message with body: {}", body);

        String eventType = exchange.getProperty("cloudevent.type", String.class);
        if (eventType == null) {
            eventType = "defaultEventType";
        }

        String eventSource = exchange.getProperty("cloudevent.source", String.class);
        if (eventSource == null) {
            eventSource = "defaultEventSource";
        }

        // Create a Cloud Event:
        CloudEvent event = CloudEventBuilder.v1()
                .withId(exchange.getExchangeId())
                .withType(eventType)
                .withSource(URI.create(eventSource))
                .withData("text/plain", body.getBytes())
                .build();
        LOG.info("Message packaged as CloudEvent: {}", event.toString());

        // Serialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        String eventAsString = new String(format.serialize(event));

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(eventAsString);
    }
}

public class InputProcessor extends RouteBuilder {
    @Override
    public void configure() throws Exception {
    }

    @BindToRegistry
    public TransformMessageToCloudEvent transformMessageToCloudEvent() {
        return new TransformMessageToCloudEvent();
    }
}
