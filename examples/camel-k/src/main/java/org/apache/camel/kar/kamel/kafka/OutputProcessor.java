// camel-k: dependency=github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626

package org.apache.camel.kar.kamel.kafka;

import org.apache.camel.BindToRegistry;
import org.apache.camel.Exchange;
import org.apache.camel.Processor;
import org.apache.camel.builder.RouteBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.cloudevents.CloudEvent;
import io.cloudevents.core.format.EventFormat;
import io.cloudevents.core.provider.EventFormatProvider;

class TransformCloudEventToMessage implements Processor {
    private static final Logger LOG = LoggerFactory.getLogger(TransformCloudEventToMessage.class);

    public void process(Exchange exchange) throws Exception {
        String exchangeBody = exchange.getIn().getBody(String.class);
        LOG.info("Received message with body: {}", exchangeBody);

        // Deserialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        CloudEvent event = format.deserialize(exchangeBody.getBytes());

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(event.getData());
    }
}

public class OutputProcessor extends RouteBuilder {
    @Override
    public void configure() throws Exception {
    }

    @BindToRegistry
    public TransformCloudEventToMessage transformCloudEventToMessage() {
        return new TransformCloudEventToMessage();
    }
}
