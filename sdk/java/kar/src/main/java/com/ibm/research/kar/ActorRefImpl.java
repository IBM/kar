package com.ibm.research.kar;

final class ActorRefImpl implements ActorRef {
  final String type;
  final String id;

  ActorRefImpl(String type, String id) {
    this.type = type;
    this.id = id;
  }

  @Override
  public String getType() {
    return type;
  }

  @Override
  public String getId() {
    return id;
  }
}
