package com.ibm.research.kar.actor.runtime.quarkus;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.logging.Logger;

import javax.annotation.PostConstruct;

import javax.enterprise.context.ApplicationScoped;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.KarConfig;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

import com.ibm.research.kar.actor.runtime.ActorManager;
import com.ibm.research.kar.actor.runtime.ActorModel;
import com.ibm.research.kar.actor.runtime.ActorRuntimeContextListener;

import io.quarkus.arc.Lock;

@ApplicationScoped
public class ActorManagerImpl implements ActorManager {

	private final static String LOG_PREFIX = "ActorManagerImpl.";
	private final Logger logger = Logger.getLogger(ActorManagerImpl.class.getName());

	private Map<String, ActorModel> actorMap;

	@PostConstruct
	public void initialize() {
		logger.info(LOG_PREFIX + "initialize: Intializing Actor map");
		this.actorMap = new HashMap<String, ActorModel>();

		logger.info(
				LOG_PREFIX + "initialize: Got init params " + KarConfig.ACTOR_CLASS_STR + ":" + KarConfig.ACTOR_TYPE_NAME_STR);

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
										String key = method.getName()+":"+method.getParameterCount();
										if (remoteMethods.containsKey(key)) {
											logger.severe("Unsupported static overload of "+method.getName()+". Multiple overloads with "+method.getParameterCount()+" arguments");
											logger.severe("Method "+method.toString()+" failed to be registered as a @Remote method");
										} else {
											logger.info(LOG_PREFIX + "initialize: adding " + key + " to remote methods for "+ actorClassName);
											remoteMethods.put(method.getName()+":"+method.getParameterTypes().length, mh);
										}
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

	@Lock(value = Lock.Type.READ)
	public ActorInstance getActor(String type, String id) {
		ActorModel model = this.actorMap.get(type);
		return model != null ? model.getActorInstances().get(id) : null;
	}

	@Lock(value = Lock.Type.WRITE)
	public ActorInstance createActor(String type, String id) {
		ActorModel actorModel = actorMap.get(type);
		if (actorModel == null) {
			return null;
		}

		try {
			Class<ActorInstance> actorClass = actorModel.getActorClass();
			ActorInstance actorObj = actorClass.getConstructor().newInstance();
			actorObj.setType(type);
			actorObj.setId(id);
			actorModel.getActorInstances().put(id, actorObj);
			return actorObj;
		} catch (Throwable t) {
			logger.severe(LOG_PREFIX + "createActor: " + t.toString());
			return null;
		}
	}

	@Lock(value = Lock.Type.WRITE)
	public boolean deleteActor(String type, String id) {
		ActorModel actorModel = this.actorMap.get(type);
		if (actorModel != null) {
			return actorModel.getActorInstances().remove(id) != null;
		} else {
			return false;
		}
	}

	@Lock(value = Lock.Type.READ)
	public boolean hasActorType(String type) {
		return this.actorMap.containsKey(type);
	}

	@Override
	@Lock(value = Lock.Type.READ)
	public MethodHandle getActorMethod(String type, String name, int numParams) {
		ActorModel model = this.actorMap.get(type);
		return model != null ? model.getRemoteMethods().get(name+":"+numParams) : null;
	}

	@Override
	@Lock(value = Lock.Type.READ)
	public MethodHandle getActorActivateMethod(String type) {
		ActorModel model = this.actorMap.get(type);
		return model != null ? model.getActivateMethod() : null;
	}

	@Override
	@Lock(value = Lock.Type.READ)
	public MethodHandle getActorDeactivateMethod(String type) {
		ActorModel model = this.actorMap.get(type);
		return model != null ? model.getDeactivateMethod() : null;
	}

}
