package com.ibm.research.kar;

/**
 * An ActorRef represents an Actor instance by wrapping an ActorType and ActorId.
 */
public class ActorRef {
  public final String type;
  public final String id;

  /**
   * Construct a reference to an Actor instance
   * @param type The type of the referenced Actor instance
   * @param id The id of the referenced Actor instance
   */
  ActorRef(String type, String id) {
    this.type = type;
    this.id = id;
  }
}
