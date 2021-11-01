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

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.Kar.Actors;
import com.ibm.research.kar.Kar.Actors.ContinueResult;
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
      return Uni.createFrom().nullItem();
    });
  }

  private Uni<Void> checkpointState() {
    JsonObjectBuilder jb = Json.createObjectBuilder();
    jb.add("cafe", this.cafe);
    jb.add("n", this.n);
    jb.add("diners", this.diners);
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
  public Uni<ContinueResult> prepare(JsonString cafe, JsonNumber n, JsonNumber servings) {
    this.cafe = cafe;
    this.n = n;
    JsonArrayBuilder jba = Json.createArrayBuilder();
    for (int i = 0; i < n.intValue(); i++) {
      jba.add(Json.createValue(this.philosopher(i)));
    }
    this.diners = jba.build();
    return checkpointState().chain(() -> {
      System.out.println("Cafe "+this.cafe+" has seated table "+this.getId()+" with "+n+" hungry philosophers for "+servings+" servings");
      return Actors.continuation(this, "serve", servings);
    });
  }

  @Remote
  public Uni<Void> serve(JsonNumber servings) {
    Uni<Void> k = Uni.createFrom().nullItem();
    for (int i = 0; i < n.intValue() - 1; i++) {
      JsonString who = Json.createValue(philosopher(i));
      JsonString fork1 = Json.createValue(fork(i));
      JsonString fork2 = Json.createValue(fork(i + 1));
      k = k.chain(() -> Actors.tell(Actors.ref("Philosopher", who.getString()), "joinTable", Json.createValue(this.getId()), fork1, fork2, servings)).chain(() -> Uni.createFrom().nullItem());
    }
    return k.chain(() -> {
      JsonString who = Json.createValue(philosopher(n.intValue() - 1));
      JsonString fork1 = Json.createValue(fork(0));
      JsonString fork2 = Json.createValue(fork(n.intValue() - 1));
      return Actors.tell(Actors.ref("Philosopher", who.getString()), "joinTable", Json.createValue(this.getId()), fork1, fork2, servings);
    });
  }

  @Remote
  public Uni<Void> doneEating(JsonString philosopher) {
    JsonArrayBuilder jba = Json.createArrayBuilder();
    boolean stateChanged = false;
    for (JsonValue diner : this.diners) {
      if (philosopher.equals(diner)) {
        stateChanged = true;
      } else {
        jba.add(diner);
      }
    }
    if (stateChanged) {
      this.diners = jba.build();
      return this.checkpointState().chain(() -> {
        System.out.println("Philosopher " + philosopher.getString() + " is done eating; there are now " + this.diners.size() + " present at the table");
        if (this.diners.size() == 0) {
          System.out.println("Table " + this.getId() + " is now empty!");
          return Actors.tell(this, "busTable");
        } else {
          return Uni.createFrom().nullItem();
        }
      });
    } else {
      return Uni.createFrom().nullItem();
    }
  }

  @Remote
  public Uni<Void> busTable() {
    Uni<Void> k = Uni.createFrom().nullItem();
    for (int i = 0; i<n.intValue(); i++) {
      final int captureI = i;
      k = k.chain(() -> Actors.remove(Actors.ref("Philosopher", philosopher(captureI))));
      k = k.chain(() -> Actors.remove(Actors.ref("Fork", fork(captureI))));
    }
    return k.chain(() -> Actors.remove(this));
  }
}
