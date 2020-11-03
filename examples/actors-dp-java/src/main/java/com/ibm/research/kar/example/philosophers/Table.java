package com.ibm.research.kar.example.philosophers;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorGetAllState;
import static com.ibm.research.kar.Kar.actorRef;
import static com.ibm.research.kar.Kar.actorRemove;
import static com.ibm.research.kar.Kar.actorSetMultipleState;
import static com.ibm.research.kar.Kar.actorSetState;
import static com.ibm.research.kar.Kar.actorTell;

import java.util.Map;
import java.util.UUID;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonArrayBuilder;
import javax.json.JsonNumber;
import javax.json.JsonObject;
import javax.json.JsonObjectBuilder;
import javax.json.JsonString;
import javax.json.JsonValue;

import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Table extends ActorSkeleton {
  JsonString cafe;
  JsonNumber n;
  JsonArray diners;
  JsonString step;

  @Activate
  public void activate() {
    Map<String, JsonValue> state = actorGetAllState(this);
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
  }

  private void checkpointState() {
    JsonObjectBuilder jb = Json.createObjectBuilder();
    jb.add("cafe", this.cafe);
    jb.add("n", this.n);
    jb.add("diners", this.diners);
    jb.add("step", this.step);
    JsonObject state = jb.build();
    actorSetMultipleState(this, state);
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
  public void prepare(JsonString cafe, JsonNumber n, JsonNumber servings, JsonString step) {
    if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
    step = Json.createValue(UUID.randomUUID().toString());
    this.cafe = cafe;
    this.n = n;
    JsonArrayBuilder jba = Json.createArrayBuilder();
    for (int i = 0; i < n.intValue(); i++) {
      jba.add(Json.createValue(this.philosopher(i)));
    }
    this.diners = jba.build();
    System.out.println("Cafe "+this.cafe+" is seating table "+this.getId()+" with "+n+" hungry philosophers for "+servings+" servings");
    actorTell(this, "serve", servings, step);
    this.step = step;
    this.checkpointState();
  }

  @Remote
  public void serve(JsonNumber servings, JsonString step) {
    if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
    step = Json.createValue(UUID.randomUUID().toString());
    for (int i = 0; i < n.intValue() - 1; i++) {
      JsonString who = Json.createValue(philosopher(i));
      JsonString fork1 = Json.createValue(fork(i));
      JsonString fork2 = Json.createValue(fork(i + 1));
      actorCall(actorRef("Philosopher", who.getString()), "joinTable", Json.createValue(this.getId()), fork1, fork2, servings, who);
    }
    JsonString who = Json.createValue(philosopher(n.intValue() - 1));
    JsonString fork1 = Json.createValue(fork(0));
    JsonString fork2 = Json.createValue(fork(n.intValue() - 1));
    actorCall(actorRef("Philosopher", who.getString()), "joinTable", Json.createValue(this.getId()), fork1, fork2, servings, who);
    this.step = step;
    actorSetState(this, "step", step);
  }

  @Remote
  public void doneEating(JsonString philosopher) {
    JsonArrayBuilder jba = Json.createArrayBuilder();
    for (JsonValue diner : this.diners) {
      if (!philosopher.equals(diner)) {
        jba.add(diner);
      }
    }
    this.diners = jba.build();
    this.checkpointState();
    System.out.println("Philosopher "+philosopher.getString()+" is done eating; there are now "+this.diners.size()+" present at the table");
    if (this.diners.size() == 0) {
      System.out.println("Table " + this.getId() + " is now empty!");
      JsonString step = Json.createValue(UUID.randomUUID().toString());
      actorTell(this, "busTable", step);
      this.step = step;
      actorSetState(this, "step", step);
    }
  }

  @Remote
  public void busTable(JsonString step) {
    if (!this.step.equals(step)) throw new RuntimeException("unexpected step");
    step = Json.createValue(UUID.randomUUID().toString());
    for (int i = 0; i<n.intValue(); i++) {
      actorRemove(actorRef("Philosopher", philosopher(i)));
      actorRemove(actorRef("Fork", fork(i)));
    }
    actorRemove(this);
    this.step = step;
    actorSetState(this, "step", step);
  }
}
