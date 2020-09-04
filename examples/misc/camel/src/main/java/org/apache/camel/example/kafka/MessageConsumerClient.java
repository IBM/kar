package org.apache.camel.example.kafka;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.builder.component.ComponentsBuilderFactory;
import org.apache.camel.impl.DefaultCamelContext;
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
        LOG.info("====== Start Cloud Event Processing ======");
        String exchangeBody = exchange.getIn().getBody(String.class);
        LOG.info("Received message from KAR Kafka with body: {}", exchangeBody);

        // Deserialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        CloudEvent event = format.deserialize(exchangeBody.getBytes());

        // Get the message from the user:
        String userMessage = new String(event.getData());

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(userMessage);
        LOG.info("====== End Cloud Event Processing ======");
    }
}

public final class MessageConsumerClient {

    private static final Logger LOG = LoggerFactory.getLogger(MessageConsumerClient.class);

    private MessageConsumerClient() {
    }

    public static void main(String[] args) throws Exception {

        LOG.info("About to run Kafka-camel integration...");

        CamelContext camelContext = new DefaultCamelContext();

        // Add route to send messages to Kafka
        camelContext.addRoutes(new RouteBuilder() {
            public void configure() {
                camelContext.getPropertiesComponent().setLocation("classpath:application.properties");

                log.info("About to start route: Kafka Server -> Log ");

                // setup kafka component with the brokers
                ComponentsBuilderFactory.kafka()
                        .brokers("{{kafka.host}}:{{kafka.port}}")
                        .register(camelContext, "kafka");

                from("kafka:{{consumer.topic}}")
                    .routeId("FromKafka")
                    .log("${body}");
                
                from("kafka:HelloEvent")
                    .routeId("EventFromKafka")
                    .process(new TransformCloudEventToMessage())
                    .log("${body}");
            }
        });
        camelContext.start();

        // let it run for 5 minutes before shutting down
        Thread.sleep(5 * 60 * 1000);

        camelContext.stop();
    }

}
