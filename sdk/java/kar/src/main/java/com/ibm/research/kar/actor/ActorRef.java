package com.ibm.research.kar.actor;

/**
 * An ActorRef supports getting the Type and Id of the referenced Actor.
 */
public interface ActorRef {
  public String getType();
  public String getId();
}
