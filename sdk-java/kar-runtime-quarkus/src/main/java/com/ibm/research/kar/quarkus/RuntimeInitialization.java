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

package com.ibm.research.kar.quarkus;

import java.util.List;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.event.Observes;

import com.ibm.research.kar.runtime.ActorManager;

import org.eclipse.microprofile.config.inject.ConfigProperty;
import org.jboss.logging.Logger;

import io.quarkus.runtime.ShutdownEvent;
import io.quarkus.runtime.StartupEvent;

@ApplicationScoped
public class RuntimeInitialization {
  private static final Logger LOG = Logger.getLogger(RuntimeInitialization.class);

  @ConfigProperty(name = "kar.actors.classes", defaultValue = "")
  List<String> actorClasses;

  @ConfigProperty(name = "kar.actors.types", defaultValue = "")
  List<String> actorTypes;

  void onStart(@Observes StartupEvent ev) {
    LOG.info("Initializing KAR Actor Runtime");
    LOG.info("Actor Classes: "+actorClasses);
    LOG.info("Actor Types: "+actorTypes);
    if (actorClasses.size() != actorTypes.size()) {
      LOG.errorf("Incompatible actor configuration! %d types and %d clases", actorTypes.size(), actorClasses.size());
    } else {
      ActorManager.initialize(actorClasses, actorTypes);
    }
  }

  void onStop(@Observes ShutdownEvent ev) {

  }
}
