package com.ibm.research.kar;

import io.quarkus.test.junit.QuarkusTest;
import io.smallrye.mutiny.Uni;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

import javax.inject.Inject;


@QuarkusTest
public class RESTInvokerTest {

    @Inject
    RESTInvoker invoker;

    @Test
    public void testInvoker() {
        if (invoker == null) {
            System.out.println("Invoker is null");
        } else {
            System.out.println("I have invoker");
        }
        Uni<String> uni = invoker.invokeKar("greeter/call/helloText", "I am Groot");
        String respStr = uni.subscribeAsCompletionStage().join();
        assertEquals("Hello I am Groot", respStr);
    }

}