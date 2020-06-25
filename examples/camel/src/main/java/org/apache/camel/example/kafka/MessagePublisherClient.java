package org.apache.camel.example.kafka;

import java.util.Calendar;
import java.util.HashMap;
import java.util.Map;
import java.net.URI;

import org.apache.camel.CamelContext;
import org.apache.camel.ProducerTemplate;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.builder.component.ComponentsBuilderFactory;
import org.apache.camel.component.kafka.KafkaConstants;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.camel.Exchange;
import org.apache.camel.Processor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.cloudevents.CloudEvent;
import io.cloudevents.core.builder.CloudEventBuilder;

import io.cloudevents.core.format.EventFormat;
import io.cloudevents.core.provider.EventFormatProvider;


class TransformMessageToCloudEvent implements Processor {
    private static final Logger LOG = LoggerFactory.getLogger(TransformMessageToCloudEvent.class);

    public void process(Exchange exchange) throws Exception {
        LOG.info("====== Start Cloud Event Processing ======");
        String exchangeBody = exchange.getIn().getBody(String.class);
        LOG.info("Received message from console with body: {}", exchangeBody);

        CloudEvent event = CloudEventBuilder.v1()
            .withId("message")
            .withType("user.generated")
            .withSource(URI.create("http://localhost"))
            .withData("text/plain", exchangeBody.getBytes())
            .build();
        LOG.info("User generated message packaged as CloudEvent: {}", event.toString());

        // Serialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        String eventAsString = new String(format.serialize(event));

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(eventAsString);
        LOG.info("====== End Cloud Event Processing ======");
    }
}

public final class MessagePublisherClient {
    private static final Logger LOG = LoggerFactory.getLogger(MessagePublisherClient.class);
    private MessagePublisherClient() { }

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Kafka-Camel integration...");
        String testKafkaMessage = "Test Message from  MessagePublisherClient " + Calendar.getInstance().getTime();
        CamelContext camelContext = new DefaultCamelContext();

        // Add route to send messages to Kafka.
        camelContext.addRoutes(new RouteBuilder() {
            public void configure() {
                camelContext.getPropertiesComponent().setLocation("classpath:application.properties");

                // Setup kafka component with the brokers.
                ComponentsBuilderFactory.kafka()
                        .brokers("{{kafka.host}}:{{kafka.port}}")
                        .register(camelContext, "kafka");

                // Send regular events.
                from("direct:kafkaStart")
                    .routeId("DirectToKafka")
                    .to("kafka:{{producer.topic}}")
                    .log("${headers}");

                // Send Cloud Event events.
                from("direct:kafkaStartEvent")
                    .routeId("kafkaStartEvent")
                    .to("kafka:HelloEvent")
                    .log("${headers}");

                // Takes input from the command line and send it as a Cloud Event.
                from("stream:in")
                    .process(new TransformMessageToCloudEvent())
                    .to("direct:kafkaStartEvent");
            }
        });

        // Camel producer template instantiation.
        ProducerTemplate producerTemplate = camelContext.createProducerTemplate();
        camelContext.start();

        // Setup the headers and send test message as a regular event through KAR Kafka.
        Map<String, Object> headers = new HashMap<>();
        headers.put(KafkaConstants.PARTITION_KEY, 0);
        headers.put(KafkaConstants.KEY, "1");
        producerTemplate.sendBodyAndHeaders("direct:kafkaStart", testKafkaMessage, headers);

        LOG.info("Successfully published regular event to KAR Kafka.");
        LOG.info("Prepare to send Cloud Event");

        // Build an event with a simple message:
        String dataContent = "Hello sent by Camel through KAR Kafka!";

        CloudEvent event = CloudEventBuilder.v1()
            .withId("hello")
            .withType("example.kafka")
            .withSource(URI.create("http://localhost"))
            .withData("text/plain", dataContent.getBytes())
            .build();
        LOG.info("Created the following CloudEvent:");
        LOG.info(event.toString());

        // Use the Cloud Events SDK serializer.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        producerTemplate.sendBody("direct:kafkaStartEvent", new String(format.serialize(event)));

        LOG.info("Successfully published test Cloud Event event to Kafka topic 'HelloEvent'.");
        LOG.info("Ready to send mesages as Cloud Events.");

        System.out.println("Enter text on the line below to be sent as Cloud Event: [Press Ctrl-C to exit.] ");

        Thread.sleep(5 * 60 * 1000);

        camelContext.stop();
    }

}
