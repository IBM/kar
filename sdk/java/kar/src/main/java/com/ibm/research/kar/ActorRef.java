package com.ibm.research.kar;

/**
 * An ActorRef supports getting the Type and Id of the referenced Actor.
 */
public interface ActorRef {
  public String getType();
  public String getId();
}
