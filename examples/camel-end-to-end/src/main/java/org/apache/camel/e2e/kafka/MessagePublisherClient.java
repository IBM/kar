package org.apache.camel.e2e.kafka;

import java.net.URI;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.builder.component.ComponentsBuilderFactory;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.camel.component.gson.GsonDataFormat;
import org.apache.camel.Exchange;
import org.apache.camel.Processor;
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

public final class MessagePublisherClient {
    private static final Logger LOG = LoggerFactory.getLogger(MessagePublisherClient.class);
    private MessagePublisherClient() { }

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Kafka-Camel integration...");
        CamelContext camelContext = new DefaultCamelContext();

        // Add route to send messages to Kafka.
        camelContext.addRoutes(new RouteBuilder() {
            public void configure() {
                camelContext.getPropertiesComponent().setLocation("classpath:application.properties");

                GsonDataFormat gson = new GsonDataFormat(FinancialEnginesStock[].class);

                // Setup kafka component with the brokers.
                ComponentsBuilderFactory.kafka()
                    .brokers("{{kafka.host}}:{{kafka.port}}")
                    .register(camelContext, "kafka");

                // Fetch external information, package it as a Cloud Event and send it to KAR via Kafka.
                from("timer:clock")
                    .setBody().header(Exchange.TIMER_COUNTER)
                    .setHeader("CamelHttpMethod", constant("GET"))
                    .to("http://financialmodelingprep.com/api/v3/quote-short/AAPL?apikey=demo")
                    .unmarshal(gson)
                    .process(new TransformListToSingleElement())
                    .process(new TransformMessageToCloudEvent())
                    .to("kafka:StockEvent")
                    .log("${body}");
            }
        });

        camelContext.start();

        // Run for 10s.
        Thread.sleep(1 * 10 * 1000);

        camelContext.stop();
    }

}
