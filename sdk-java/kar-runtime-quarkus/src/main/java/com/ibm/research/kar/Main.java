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
package com.ibm.research.kar;

import io.quarkus.runtime.Quarkus;
import io.quarkus.runtime.QuarkusApplication;
import io.quarkus.runtime.annotations.QuarkusMain;

import javax.json.JsonValue;

import java.util.concurrent.CompletionStage;

import javax.json.Json;
import javax.json.JsonArray;
import javax.ws.rs.core.Response;

@QuarkusMain
public class Main {
    public static void main(String... args) {
        Quarkus.run(MyApp.class, args);
        Quarkus.waitForExit();
    }

    public static class MyApp implements QuarkusApplication {

        KarRest client = new KarRest();

        @Override
        public int run(String... args) throws Exception {

            for (int i = 0; i < 0; i++) {
                syncCall(i);
            }

            for (int i = 0; i < 0; i++) {
                asyncCall(i);
            }

            for (int i = 0; i < 0; i++) {
                syncTell(i);
            }

            for (int i = 0; i < 0; i++) {
                syncActorCall(i);
            }

            for (int i = 0; i < 0; i++) {
                asyncActorCall(i);
            }

            for (int i = 0; i < 1; i++) {
                actorTell(i);
            }
            return 0;

        }

        private void syncCall(int num) {
            JsonValue json = Json.createObjectBuilder().add("name", "I am Groot!").build();

            System.out.println("Created JSON params " + json.toString());

            Response resp = client.callPost("greeter", "helloJson", json);

            Object value = resp.getEntity();
            System.out.println("-------- Sync Response from Service ---------\n");
            System.out.println(value);
            System.out.println("\n----------------------------------------");

            resp = client.callGet("greeter", "health");
            value = resp.getEntity();

            System.out.println("-------- Sync Response from Service ---------\n");
            System.out.println(value);
            System.out.println("\n----------------------------------------");

        }

        private void asyncCall(int num) {

            JsonValue json = Json.createObjectBuilder().add("name", "I am Async Groot!").build();

            System.out.println("Created JSON params " + json.toString());

            CompletionStage<Response> future = client.callAsyncPost("greeter", "helloJson", json);

            Response resp = future.toCompletableFuture().join();
            Object value = resp.getEntity();
            System.out.println("-------- Async Response from Service ---------\n");
            System.out.println(value);
            System.out.println("\n----------------------------------------");

            future = client.callAsyncGet("greeter", "health");
            resp = future.toCompletableFuture().join();

            value = resp.getEntity();

            System.out.println("-------- Sync Response from Service ---------\n");
            System.out.println(value);
            System.out.println("\n----------------------------------------");
        }

        private void syncTell(int num) {
            JsonValue json = Json.createObjectBuilder().add("name", "I am Tell Groot!").build();

            System.out.println("Created JSON params " + json.toString());

            Response resp = client.tellPost("greeter", "helloJson", json);

            Object value = resp.getEntity();
            System.out.println("-------- Sync Response from Service ---------\n");
            System.out.println(resp.getStatus());
            System.out.println("\n----------------------------------------");
        }

        private void syncActorCall(int num) {
            JsonArray jArr = Json.createArrayBuilder()
            .add(10)
            .add(20).build();

            System.out.println("Created JSON array " + jArr);

            Response resp = client.actorCall("Cafe", "Cafe de Flore", "seatTable", null, jArr);

            Object value = resp.getEntity();
            System.out.println("-------- Sync Response from Actor ---------\n");
            System.out.println(value);
            System.out.println("\n----------------------------------------");

        }

        private void asyncActorCall(int num) {
            JsonArray jArr = Json.createArrayBuilder()
            .add(10)
            .add(20).build();

            System.out.println("Created JSON array " + jArr);

            CompletionStage<Response> future = client.actorCallAsync("Cafe", "Cafe de Flore", "seatTable", null, jArr);

            Response resp = future.toCompletableFuture().join();
            Object value = resp.getEntity();
            System.out.println("-------- Async Response from Actor ---------\n");
            System.out.println(value);
            System.out.println("\n----------------------------------------");
        }


        private void actorTell(int num) {
            JsonArray jArr = Json.createArrayBuilder()
            .add(10)
            .add(20).build();

            System.out.println("Created JSON array " + jArr);

            Response resp = client.actorTell("Cafe", "Cafe de Flore", "seatTable", jArr);

            System.out.println("-------- Sync Response from Actor ---------\n");
            System.out.println(resp.getStatus());
            System.out.println("\n----------------------------------------");

        }
    }
}