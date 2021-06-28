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
import io.smallrye.mutiny.Uni;
import io.vertx.core.json.JsonObject;
import io.vertx.core.json.JsonArray;

import static com.ibm.research.kar.Kar.Services.*;

@QuarkusMain
public class Main {
    public static void main(String... args) {
        Quarkus.run(MyApp.class, args);
        Quarkus.waitForExit();
    }

    public static class MyApp implements QuarkusApplication {

  
        @Override
        public int run(String... args) throws Exception {

            for (int i = 0; i < 1; i++) {
                syncCall(i);
            }

            for (int i = 0; i < 0; i++) {
                asyncCall(i);
            }
            return 0;

        }

        private void syncCall(int num) {
            JsonObject json = new JsonObject().put("name", "I am Groot! " + num);
            System.out.println("Created JSON params " + json.toString());

            Object value = post("greeter","helloJson", json);
            //Object value2 = get("greeter","helloJson");
            //Object value3 = put("greeter", "helloJson", json);
            //Object value4 = head("greeter", "helloJson");

            String respType = "JsonObject";
            if (value instanceof JsonArray) {
                respType = "JsonArray";
            }

            System.out.println("-------- Sync Response from Service ---------\n");
            System.out.println(value.toString());
            System.out.println("Type:" + respType);
            System.out.println("\n----------------------------------------");
        }

        private void asyncCall(int num) {
            JsonObject json = new JsonObject().put("name", "I am Rocket! " + num);
            System.out.println("Created JSON params " + json.toString());

            Uni<Object> uni = postAsync("greeter","helloJson", json);

            Object value = uni.subscribeAsCompletionStage().join();

            String respType = "JsonObject";
            if (value instanceof JsonArray) {
                respType = "JsonArray";
            }

            System.out.println("-------- Async Response from Service ---------\n");
            System.out.println(value.toString());
            System.out.println("Type:" + respType);
            System.out.println("\n----------------------------------------");
        }

    }
}