package com.ibm.research.kar;

import javax.inject.Inject;

import io.quarkus.runtime.Quarkus;
import io.quarkus.runtime.QuarkusApplication;
import io.quarkus.runtime.annotations.QuarkusMain;
import io.smallrye.mutiny.Uni;

@QuarkusMain
public class Main {
    public static void main(String... args) {
        Quarkus.run(MyApp.class, args);
        Quarkus.waitForExit();
    }

    public static class MyApp implements QuarkusApplication {

        @Inject
        RESTInvoker invoker;

        @Override
        public int run(String... args) throws Exception {
            for (int i = 0; i < 100; i++) {
                Uni<String> uni = invoker.invokeKar("greeter/call/helloText", "Groot");
                String respStr = uni.subscribeAsCompletionStage().join();
                System.out.println("-------- Response from Service ---------\n");
                System.out.println(respStr);
                System.out.println("\n----------------------------------------");

            }
            return 0;

        }

    }
}