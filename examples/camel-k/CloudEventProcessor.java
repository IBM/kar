/*
 * Copyright IBM Corporation 2020,2023
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

// camel-k: dependency=github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626

package com.ibm.research.kar.camel;

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

        // Create a CloudEvent:
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

public class CloudEventProcessor extends RouteBuilder {
    @Override
    public void configure() throws Exception {
    }

    @BindToRegistry
    public TransformMessageToCloudEvent transformMessageToCloudEvent() {
        return new TransformMessageToCloudEvent();
    }

    @BindToRegistry
    public TransformCloudEventToMessage transformCloudEventToMessage() {
        return new TransformCloudEventToMessage();
    }
}
