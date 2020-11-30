package com.ibm.research.kar.example.timeout;

import static com.ibm.research.kar.Kar.actorCall;
import static com.ibm.research.kar.Kar.actorRef;

import javax.json.JsonString;

import com.ibm.research.kar.actor.ActorRef;
import com.ibm.research.kar.actor.ActorSkeleton;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class Test extends ActorSkeleton {

  @Remote public void A() {
    System.out.println("Entering method A");
    actorCall(this, this, "B"); // synchronous call to self within the same session -> OK
    System.out.println("Exiting method A");
  }

  @Remote public void B() {
    System.out.println("Entering method B");
    actorCall(this, "A"); // synchronous call to self in a new session -> deadlock
    System.out.println("Exiting method B");
  }

  @Remote public void externalA(JsonString target) {
    ActorRef other = actorRef("Test", target.getString());
    actorCall(other, "A");
  }

  @Remote public void externalB(JsonString target) {
    ActorRef other = actorRef("Test", target.getString());
    actorCall(other, "B");
  }
}
