package org.apache.camel.kar.kamel.kafka;

import org.apache.camel.BindToRegistry;
import org.apache.camel.Exchange;
import org.apache.camel.Processor;
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
        String body = exchange.getIn().getBody().toString();
        LOG.info("Received message from console with body: {}", body);

        // Create a Cloud Event:
        // - The event type of the event is the stock name.
        // - The event data is the stock price.
        CloudEvent event = CloudEventBuilder.v1()
                .withId("stockID")
                .withType("stock.input")
                .withSource(URI.create("http://financialmodelingprep.com"))
                .withData("text/plain", body.getBytes())
                .build();
        LOG.info("User generated message packaged as CloudEvent: {}", event.toString());

        // Serialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        String eventAsString = new String(format.serialize(event));

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(eventAsString);
    }
}

public class InputConfiguration {
    @BindToRegistry
    public TransformMessageToCloudEvent transformMessageToCloudEvent() {
        return new TransformMessageToCloudEvent();
    }
}
