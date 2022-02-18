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

import java.time.Duration;
import java.time.Instant;

public final class Reminder {
  private final ActorRef actor;
  private final String ID;
  private final String path;
  private final Instant targetTime;
  private final Duration period;
  private final Object[] arguments;

  public Reminder(ActorRef actor, String ID, String path, Instant targetTime, Duration period, Object[] arguments) {
    this.actor = actor;
    this.ID = ID;
    this.path = path;
    this.targetTime = targetTime;
    this.period = period;
    this.arguments = arguments;
  }

  public ActorRef getActor() {
    return this.actor;
  }

  public String getID() {
    return this.ID;
  }

  public String getPath() {
    return this.path;
  }

  public Instant getTargetTime() {
    return this.targetTime;
  }

  public Duration getPeriod() {
    return this.period;
  }

  public Object[] getArguments() {
    return this.arguments;
  }

  public Object getArgument(int n) {
    return this.arguments[n];
  }

  public String toString() {
    return "{" + " ActorType: " + this.actor.getType() + ", ActorId: " + this.actor.getId() + ", ID: " + this.ID
        + ", targetTime: " + this.targetTime + ", period: " + this.period.toString() + ", args: " + this.arguments + "}";
  }
}
