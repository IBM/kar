package com.ibm.research.kar.actor;

public final class Subscription {
  private final ActorRef actor;
  private final String ID;
  private final String path;
  private final String topic;

  public Subscription(ActorRef actor, String ID, String path, String topic) {
    this.actor = actor;
    this.ID = ID;
    this.path = path;
    this.topic = topic;
  }

  public final ActorRef getActor() { return this.actor; }

  public final String getID() { return this.ID; }

  public final String getPath() { return this.path; }

  public final String getTopic() { return this.topic; }

  public final String toString() {
    return "{" + " ActorType: " + this.actor.getType() + ", ActorId: " + this.actor.getId() + ", ID: " + this.ID
    + ", topic: " + this.topic + ", path: " + this.path + "}";
  }
}
