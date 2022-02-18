/*
 * Copyright IBM Corporation 2020,2022
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
