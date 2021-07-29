/*
 * Copyright IBM Corporation 2020,2021
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

package com.ibm.research.kar.actor.runtime;

import java.util.HashSet;
import java.util.Set;
import java.util.logging.Logger;

import javax.ws.rs.ApplicationPath;
import javax.ws.rs.core.Application;

import com.ibm.research.kar.JSONProvider;

@ApplicationPath("/kar/impl/v1/")
public class ActorRuntimeApplication extends Application {

  private static Logger logger = Logger.getLogger(ActorRuntimeApplication.class.getName());

  public Set<Class<?>> getClasses() {
    logger.info("Running ActorRuntimeApplication getClasses()");
    Set<Class<?>> classes = new HashSet<Class<?>>();
    classes.add(JSONProvider.class);
    classes.add(ActorRuntimeResource.class);
    classes.add(StatusReporter.class);
    return classes;
  }
}
