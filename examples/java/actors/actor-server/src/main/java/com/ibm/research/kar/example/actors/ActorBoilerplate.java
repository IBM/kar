package com.ibm.research.kar.example.actors;

import com.ibm.research.kar.ActorInstance;

/**
 * Kar requires all Actor Classes to implement the ActorIntance interface.
 *
 * When writing an application component that contains multiple actor types,
 * one convenient pattern is to share the boilerplate code by defining a
 * common superclass extended by all your Actor classes.
 */
public abstract class ActorBoilerplate implements ActorInstance {
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
}
