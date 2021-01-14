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

package org.apache.camel.kamel;

import org.apache.camel.Processor;
import java.net.URI;

import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.gson.GsonDataFormat;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.cloudevents.CloudEvent;
import io.cloudevents.core.builder.CloudEventBuilder;
import io.cloudevents.core.format.EventFormat;
import io.cloudevents.core.provider.EventFormatProvider;

class FinancialEnginesStock {
	private String symbol;
    private float price;
    private int volume;

	public String getSymbol() {
		return symbol;
	}

	public void setSymbol(String symbol) {
		this.symbol = symbol;
	}

	public float getPrice() {
		return price;
	}

	public void setPrice(float price) {
		this.price = price;
    }

    public int getVolume() {
		return volume;
	}

	public void setVolume(int volume) {
		this.volume = volume;
    }
}

class TransformMessageToCloudEvent implements Processor {
    private static final Logger LOG = LoggerFactory.getLogger(TransformMessageToCloudEvent.class);

    public void process(Exchange exchange) throws Exception {
        FinancialEnginesStock exchangeBody = exchange.getIn().getBody(FinancialEnginesStock.class);
        LOG.info("Received message from console with body: {}", exchangeBody.getSymbol());

        String price = new Float(exchangeBody.getPrice()).toString();

        // Create a Cloud Event:
        //  - The event type of the event is the stock name.
        //  - The event data is the stock price.
        CloudEvent event = CloudEventBuilder.v1()
            .withId("stock.price")
            .withType(exchangeBody.getSymbol())
            .withSource(URI.create("http://localhost"))
            .withData("text/plain", price.getBytes())
            .build();
        LOG.info("User generated message packaged as CloudEvent: {}", event.toString());

        // Serialize event.
        EventFormat format = EventFormatProvider.getInstance().resolveFormat("application/cloudevents+json");
        String eventAsString = new String(format.serialize(event));

        // Set Exchange body to CloudEvent and send it along.
        exchange.getIn().setBody(eventAsString);
    }
}

class TransformListToSingleElement implements Processor {
    private static final Logger LOG = LoggerFactory.getLogger(TransformListToSingleElement.class);

    public void process(Exchange exchange) throws Exception {
        FinancialEnginesStock[] stock = exchange.getIn().getBody(FinancialEnginesStock[].class);
        exchange.getIn().setBody(stock[0]);
        LOG.info("Process Response: {} {}", stock[0].getSymbol(), stock[0].getPrice());
    }
}

public class KafkaProducer extends RouteBuilder {
    @Override
    public void configure() throws Exception {
        GsonDataFormat gson = new GsonDataFormat(FinancialEnginesStock[].class);

        from("timer:clock")
            .setBody().header(Exchange.TIMER_COUNTER)
            .setHeader("CamelHttpMethod", constant("GET"))
            .to("http://financialmodelingprep.com/api/v3/quote-short/AAPL?apikey=demo")
            .unmarshal(gson)
            .process(new TransformListToSingleElement())
            .process(new TransformMessageToCloudEvent())
            .to("kafka:CamelKEvent?brokers="+System.getenv("KAR_KAFKA_CLUSTER_IP")+":9092")
            .log("${body}");
    }
}