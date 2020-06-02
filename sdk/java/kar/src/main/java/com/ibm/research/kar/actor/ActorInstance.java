package com.ibm.research.kar.actor;

/**
 * ActorInstace must be implemented by every Class that
 * provides an Actor type to Kar.
 */
public interface ActorInstance extends ActorRef {
  public String getSession();

  public void setType(String type);
  public void setId(String id);
  public void setSession(String session);
}
