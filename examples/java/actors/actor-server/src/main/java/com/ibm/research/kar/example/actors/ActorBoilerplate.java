package com.ibm.research.kar.example.actors;

import com.ibm.research.kar.ActorInstance;

public class ActorBoilerplate implements ActorInstance {
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
