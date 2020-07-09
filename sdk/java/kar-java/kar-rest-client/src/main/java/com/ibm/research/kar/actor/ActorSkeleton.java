package com.ibm.research.kar.actor;

/**
 * Kar requires all Actor Classes to implement the ActorInstance interface.
 *
 * This class provides a default implementation of ActorInstance that
 * may be used as the superclass of any application Actor class. It is not
 * required that Actor classes extend ActorSkeleton, but doing so eliminates
 * some boilerplate code from the application.
 */
public abstract class ActorSkeleton implements ActorInstance {
  protected String type;
  protected String id;
  protected String session;

  @Override
  public String getType() {
    return type;
  }

  @Override
  public String getId() {
    return id;
  }

  @Override
  public String getSession() {
    return session;
  }

  @Override
  public void setType(String type) {
    this.type = type;
  }

  @Override
  public void setId(String id) {
    this.id = id;
  }

  @Override
  public void setSession(String session) {
    this.session = session;
  }

  @Override
  public String toString() {
    return this.getType()+"["+this.getId()+"]";
  }
}
