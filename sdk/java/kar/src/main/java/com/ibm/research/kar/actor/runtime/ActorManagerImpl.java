package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.logging.Logger;

import javax.annotation.PostConstruct;
import javax.ejb.ConcurrencyManagement;
import javax.ejb.ConcurrencyManagementType;
import javax.ejb.Lock;
import javax.ejb.LockType;
import javax.ejb.Singleton;

import javax.enterprise.context.ApplicationScoped;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.KarConfig;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Singleton
@ApplicationScoped
@ConcurrencyManagement(ConcurrencyManagementType.CONTAINER)
public class ActorManagerImpl implements ActorManager {

	private final static String LOG_PREFIX = "ActorManagerImpl.";
	private final Logger logger = Logger.getLogger(ActorManagerImpl.class.getName());

	private Map<String, ActorModel> actorMap;

	@PostConstruct
	public void initialize() {
		logger.info(LOG_PREFIX + "initialize: Intializing Actor map");
		this.actorMap = new HashMap<String, ActorModel>();

		logger.info(LOG_PREFIX + "initialize: Got init params " + KarConfig.ACTOR_CLASS_STR + ":" + KarConfig.ACTOR_TYPE_NAME_STR);

		// ensure that we have non-null class and kar type strings from web.xml
		if ((KarConfig.ACTOR_CLASS_STR != null) && (KarConfig.ACTOR_TYPE_NAME_STR != null)) {
			List<String> classList = Arrays.asList(KarConfig.ACTOR_CLASS_STR.split("\\s*,\\s*"));
			List<String> nameList = Arrays.asList(KarConfig.ACTOR_TYPE_NAME_STR.split("\\s*,\\s*"));

			if (classList.size() != nameList.size()) {
				logger.severe("Incompatible actor configuration! " + ActorRuntimeContextListener.KAR_ACTOR_CLASSES + "="
						+ KarConfig.ACTOR_CLASS_STR + " and " + ActorRuntimeContextListener.KAR_ACTOR_TYPES + "="
						+ KarConfig.ACTOR_TYPE_NAME_STR);
			} else {
				// Create ActorModel for each class
				MethodHandles.Lookup lookup = MethodHandles.lookup();
				for (String actorClassName : classList) {
					try {
						Class<?> cls = Class.forName(actorClassName);
						boolean isAnnotated = cls.getAnnotation(Actor.class) != null;
						boolean isActorInstance = ActorInstance.class.isAssignableFrom(cls);

						// class must be annotated with @Actor and implement ActorInstance to be
						// processed as a valid Actor
						if (isAnnotated && isActorInstance) {
							@SuppressWarnings("unchecked") // can never fail because isActorInstance is true
							Class<ActorInstance> actorClass = ((Class<ActorInstance>) cls);

							Method[] methods = cls.getMethods();
							Map<String, MethodHandle> remoteMethods = new HashMap<String, MethodHandle>();
							MethodHandle activateMethod = null;
							MethodHandle deactivateMethod = null;

							for (Method method : methods) {
								if (method.isAnnotationPresent(Remote.class)) {
									try {
										MethodHandle mh = lookup.unreflect(method);
										logger.info(LOG_PREFIX + "initialize: adding method " + method.getName() + " to remote methods for "
												+ actorClassName);
										remoteMethods.put(method.getName(), mh);
									} catch (IllegalAccessException e) {
										logger.severe(LOG_PREFIX + "initialize: IllegalAccessException when adding" + method.getName()
												+ " to remote methods for " + actorClassName);
									}
								} else if (method.isAnnotationPresent(Activate.class)) {
									try {
										activateMethod = lookup.unreflect(method);
									} catch (IllegalAccessException e) {
										logger
												.severe(LOG_PREFIX + "initialize: IllegalAccessException adding activate to " + actorClassName);
									}
								} else if (method.isAnnotationPresent(Deactivate.class)) {
									try {
										deactivateMethod = lookup.unreflect(method);
									} catch (IllegalAccessException e) {
										logger.severe(
												LOG_PREFIX + "initialize: IllegalAccessException adding deactivate to " + actorClassName);
									}
								}

							}
							// create new ActorModel
							ActorModel actorModel = new ActorModel();

							String karTypeName = nameList.get(classList.indexOf(actorClassName));

							// add kar type and class for future (?) bookeeping
							actorModel.setType(karTypeName);
							actorModel.setActorClass(actorClass);

							// add methods so we don't have to look them up later
							actorModel.setActivateMethod(activateMethod); // ok to be null
							actorModel.setDeactivateMethod(deactivateMethod); // ok to be null
							actorModel.setRemoteMethods(remoteMethods); // ok to be empty

							// put new ActorModel in ActorMap with KAR type as key
							actorMap.put(karTypeName, actorModel);
						} else {
							if (!isAnnotated) {
								logger.severe(LOG_PREFIX + "initialize: " + actorClassName + " is not annotated with @Actor");
							}
							if (!isActorInstance) {
								logger.severe(LOG_PREFIX + "initialize: " + actorClassName + " does not implement "
										+ ActorInstance.class.getName());
							}
						}
					} catch (ClassNotFoundException e) {
						e.printStackTrace();
						System.out.print(LOG_PREFIX + "initialize: Cannot log class " + actorClassName);
					}
				}

			}

			logger.info(LOG_PREFIX + "initialize: actor map initialized with " + actorMap.size() + " entries");
		}
	}

	@Lock(LockType.WRITE)
	public ActorInstance createActor(String type, String id) {
		logger.info(LOG_PREFIX + "createActor ActorManager");

		ActorModel actorRef = actorMap.get(type);
		ActorInstance actorObj = null;

		if (actorRef != null) {
			try {
				Class<ActorInstance> actorClass = actorRef.getActorClass();
				actorObj = actorClass.getConstructor().newInstance();
				actorObj.setType(type);
				actorObj.setId(id);

				// initialize actor instance
				MethodHandle activate = actorRef.getActivateMethod();
				if (activate != null) {
					activate.invoke(actorObj);
				}

				// put reference to actorObj in the ActorModel
				actorRef.getActorInstances().put(id, actorObj);
			} catch (Throwable t) {
				logger.severe(LOG_PREFIX + "createActor: " + t.getMessage());
			}
		}

		return actorObj;
	}

	@Lock(LockType.WRITE)
	public void deleteActor(String type, String id) {
		logger.info(LOG_PREFIX + "deleteActor: deleting " + type + " actor " + id);
		ActorModel actorRef = this.actorMap.get(type);

		if (actorRef != null) {
			Object actorObj = actorRef.getActorInstances().get(id);

			if (actorObj != null) {
				MethodHandle deactivateMethod = actorRef.getDeactivateMethod();
				if (deactivateMethod != null) {
					try {
						deactivateMethod.invoke(actorObj);
					} catch (Throwable t) {
						logger.severe(LOG_PREFIX + "deleteActor: error executing actor deactivate method "+t.getMessage());
					}
				}
				actorRef.getActorInstances().remove(actorObj);
			} else {
				logger.info(LOG_PREFIX + "deleteActor: warning, no instance found for actor " + id);
			}
		} else {
			logger.info(LOG_PREFIX + "deleteActor: warning, no model found for " + type + " actor");
		}
	}

	@Lock(LockType.READ)
	public ActorInstance getActor(String type, String id) {
		logger.info(LOG_PREFIX + "getActor: Retrieving actor instance");

		ActorModel model = this.actorMap.get(type);
		if (model != null) {
			return model.getActorInstances().get(id);
		} else {
			return null;
		}
	}

	@Lock(LockType.READ)
	public int getNumActors() {
		logger.info("ActorManagerImpl.getNumActors: checking actor map for size");

		if (actorMap != null) {
			return actorMap.size();
		} else {
			logger.info(LOG_PREFIX + "getNumActors: no map instance found");
			return 0;
		}
	}

	@Override
	@Lock(LockType.READ)
	public MethodHandle getActorMethod(String type, String name) {
		logger.info(LOG_PREFIX + "getactorMethod: getting method " + name + " for " + type + " actor");
		ActorModel model = this.actorMap.get(type);

		logger.info(LOG_PREFIX + "getactorMethod: found actor model " + model);

		MethodHandle method = null;

		if (model != null) {
			method = model.getRemoteMethods().get(name);
		} else {
			logger.info(LOG_PREFIX + "getActorMethod: Warning, no model of type " + type + " found");
		}

		return method;
	}

}
