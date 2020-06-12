package com.ibm.research.kar.actor.runtime;

import java.util.HashSet;
import java.util.Set;
import java.util.logging.Logger;

import javax.ws.rs.ApplicationPath;
import javax.ws.rs.core.Application;

@ApplicationPath("/kar/impl/v1/")
public class ActorRuntimeApplication extends Application {

  private static Logger logger = Logger.getLogger(ActorRuntimeApplication.class.getName());

  public Set<Class<?>> getClasses() {
    logger.info("Running ActorRuntimeApplication getClasses()");
    Set<Class<?>> classes = new HashSet<Class<?>>();
    classes.add(JSONProvider.class);
    classes.add(ActorRuntimeResource.class);
    return classes;
  }
}
