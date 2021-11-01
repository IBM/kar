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

package com.ibm.research.kar.example.philosophers;

import java.util.UUID;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar.Actors;
import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

import io.smallrye.mutiny.Uni;

@Actor
public class Table extends ActorSkeleton {
  JsonString cafe;
  JsonNumber n;
  JsonArray diners;
  JsonString step;

  @Activate
  public Uni<Void> activate() {
    return Actors.State.getAll(this).chain(state -> {
      if (state.containsKey("cafe")) {
        this.cafe = ((JsonString) state.get("cafe"));
      }
      if (state.containsKey("n")) {
        this.n = ((JsonNumber) state.get("n"));
      }
      if (state.containsKey("diners")) {
        this.diners = ((JsonArray) state.get("diners"));
      }
      if (state.containsKey("step")) {
        this.step = ((JsonString) state.get("step"));
      } else {
        // Initial step for an uninitialized Table is its id
        this.step = Json.createValue(this.getId());
      }
      return Uni.createFrom().nullItem();
    });
  }

  private Uni<Void> checkpointState() {
    JsonObjectBuilder jb = Json.createObjectBuilder();
    jb.add("cafe", this.cafe);
    jb.add("n", this.n);
    jb.add("diners", this.diners);
    jb.add("step", this.step);
    JsonObject state = jb.build();
    return Actors.State.setV(this, state);
  }

  @Remote
  public JsonNumber occupancy() {
    return Json.createValue(this.diners != null ? this.diners.size() : 0);
  }

  private String philosopher(int p) {
    return this.cafe.getString() + "-" + this.getId() + "-philosopher-" + p;
  }

  private String fork(int f) {
    return this.cafe.getString() + "-" + this.getId() + "-fork-" + f;
  }

  @Remote
  public Uni<Void> prepare(JsonString cafe, JsonNumber n, JsonNumber servings, JsonString currentStep) {
    if (!this.step.equals(step)) {
      return Uni.createFrom().failure(new RuntimeException("unexpected step"));
    }
    final JsonString step = Json.createValue(UUID.randomUUID().toString());
    this.cafe = cafe;
    this.n = n;
    JsonArrayBuilder jba = Json.createArrayBuilder();
    for (int i = 0; i < n.intValue(); i++) {
      jba.add(Json.createValue(this.philosopher(i)));
    }
    this.diners = jba.build();
    System.out.println("Cafe "+this.cafe+" is seating table "+this.getId()+" with "+n+" hungry philosophers for "+servings+" servings");
    return Actors.tell(this, "serve", servings, step)
      .chain(() -> {
        this.step = step;
        return this.checkpointState();
      });
  }

  @Remote
  public Uni<Void> serve(JsonNumber servings, JsonString currentStep) {
    if (!this.step.equals(step)) {
      return Uni.createFrom().failure(new RuntimeException("unexpected step"));
    }
    final JsonString step = Json.createValue(UUID.randomUUID().toString());
    Uni<Void> k = Uni.createFrom().nullItem();
    for (int i = 0; i < n.intValue() - 1; i++) {
      JsonString who = Json.createValue(philosopher(i));
      JsonString fork1 = Json.createValue(fork(i));
      JsonString fork2 = Json.createValue(fork(i + 1));
      k = k.chain(() -> Actors.call(Actors.ref("Philosopher", who.getString()), "joinTable", Json.createValue(this.getId()), fork1, fork2, servings)).chain(() -> Uni.createFrom().nullItem());
    }
    return k.chain(() -> {
      JsonString who = Json.createValue(philosopher(n.intValue() - 1));
      JsonString fork1 = Json.createValue(fork(0));
      JsonString fork2 = Json.createValue(fork(n.intValue() - 1));
      return Actors.call(Actors.ref("Philosopher", who.getString()), "joinTable", Json.createValue(this.getId()), fork1, fork2, servings)
        .chain(() -> {
          this.step = step;
          return Actors.State.setV(this, "step", step);
        });
    });

  }

  @Remote
  public Uni<Void> doneEating(JsonString philosopher) {
    JsonArrayBuilder jba = Json.createArrayBuilder();
    for (JsonValue diner : this.diners) {
      if (!philosopher.equals(diner)) {
        jba.add(diner);
      }
    }
    this.diners = jba.build();
    return this.checkpointState().chain(() -> {
      System.out.println("Philosopher "+philosopher.getString()+" is done eating; there are now "+this.diners.size()+" present at the table");
      if (this.diners.size() == 0) {
        System.out.println("Table " + this.getId() + " is now empty!");
        JsonString step = Json.createValue(UUID.randomUUID().toString());
        return Actors.tell(this, "busTable", step)
          .chain(() -> {
            this.step = step;
            return Actors.State.setV(this, "step", step);
          });
      } else {
        return Uni.createFrom().nullItem();
      }
    });
  }

  @Remote
  public Uni<Void> busTable(JsonString currentStep) {
    if (!this.step.equals(step)) {
      return Uni.createFrom().failure(new RuntimeException("unexpected step"));
    }
    Uni<Void> k = Uni.createFrom().nullItem();
    final JsonString step = Json.createValue(UUID.randomUUID().toString());
    for (int i = 0; i<n.intValue(); i++) {
      final int captureI = i;
      k = k.chain(() -> Actors.remove(Actors.ref("Philosopher", philosopher(captureI))));
      k = k.chain(() -> Actors.remove(Actors.ref("Fork", fork(captureI))));
    }
    return k.chain(() -> {
      Actors.remove(this);
      this.step = step;
      return Actors.State.setV(this, "step", step);
    });
  }
}
