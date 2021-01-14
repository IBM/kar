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

package org.apache.camel.e2e.kafka;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.builder.component.ComponentsBuilderFactory;
import org.apache.camel.component.slack.SlackComponent;
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
        String exchangeBody = exchange.getIn().getBody(String.class);
        LOG.info("Received message from KAR Kafka with body: {}", exchangeBody);

        // Deserialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        CloudEvent event = format.deserialize(exchangeBody.getBytes());

        // Get the message from the user:
        String stockNameAndPrice = event.getType() + " : " + new String(event.getData());

        exchange.getIn().setHeader("redirectToSlack", "true");
        String outputSlackWebhook = System.getenv("SLACK_KAR_OUTPUT_WEBHOOK");
        if (outputSlackWebhook == null) {
            exchange.getIn().setHeader("redirectToSlack", "false");
        }

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(stockNameAndPrice);
    }
}

public final class MessageConsumerClient {

    private static final Logger LOG = LoggerFactory.getLogger(MessageConsumerClient.class);

    private MessageConsumerClient() {
    }

    public static void main(String[] args) throws Exception {

        LOG.info("About to run Kafka-camel integration...");

        CamelContext camelContext = new DefaultCamelContext();
        final SlackComponent slackComponent = (SlackComponent) camelContext.getComponent("slack");
        slackComponent.setWebhookUrl(System.getenv("SLACK_KAR_OUTPUT_WEBHOOK"));

        // Add route to send messages to Kafka
        camelContext.addRoutes(new RouteBuilder() {
            public void configure() {
                camelContext.getPropertiesComponent().setLocation("classpath:application.properties");

                // setup kafka component with the brokers
                ComponentsBuilderFactory.kafka()
                    .brokers("{{kafka.host}}:{{kafka.port}}")
                    .register(camelContext, "kafka");
                
                from("kafka:StockEvent")
                    .routeId("EventFromKafka")
                    .process(new TransformCloudEventToMessage())
                    .choice()
                        .when(simple("${header.redirectToSlack} == 'true'"))
                            .to("slack:#kar-output")
                        .otherwise()
                            .log("${body}");
            }
        });
        camelContext.start();

        // Run for 5 minutes.
        Thread.sleep(5 * 60 * 1000);

        camelContext.stop();
    }

}
