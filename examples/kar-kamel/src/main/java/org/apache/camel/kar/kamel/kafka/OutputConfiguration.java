package org.apache.camel.kar.kamel.kafka;

import org.apache.camel.BindToRegistry;
import org.apache.camel.Exchange;
import org.apache.camel.Processor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.cloudevents.CloudEvent;
import io.cloudevents.core.format.EventFormat;
import io.cloudevents.core.provider.EventFormatProvider;

class TransformCloudEventToMessage implements Processor {
    private static final Logger LOG = LoggerFactory.getLogger(TransformCloudEventToMessage.class);

    public void process(Exchange exchange) throws Exception {
        String exchangeBody = exchange.getIn().getBody(String.class);
        LOG.info("Received message from KAR Kafka with body: {}", exchangeBody);

        // Deserialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        CloudEvent event = format.deserialize(exchangeBody.getBytes());

        exchange.getIn().setHeader("redirectToSlack", "true");
        String outputSlackWebhook = System.getenv("SLACK_KAR_OUTPUT_WEBHOOK");
        if (outputSlackWebhook == null) {
            exchange.getIn().setHeader("redirectToSlack", "false");
        }

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(event.getData());
    }
}

public class OutputConfiguration {
    @BindToRegistry
    public TransformCloudEventToMessage transformCloudEventToMessage() {
        return new TransformCloudEventToMessage();
    }
}