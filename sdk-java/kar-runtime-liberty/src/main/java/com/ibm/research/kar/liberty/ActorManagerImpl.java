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

package com.ibm.research.kar.liberty;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.logging.Logger;

import javax.annotation.PostConstruct;
import javax.ejb.Singleton;

import javax.enterprise.context.ApplicationScoped;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.KarConfig;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;
import com.ibm.research.runtime.ActorManager;
import com.ibm.research.runtime.ActorType;

@Singleton
@ApplicationScoped
public class ActorManagerImpl implements ActorManager {

	private final static String LOG_PREFIX = "ActorManagerImpl.";
	private final Logger logger = Logger.getLogger(ActorManagerImpl.class.getName());

	// This map is read only once it is initialized.
	private HashMap<String, ActorType> actorTypes;
	// This map is concurrently updated.
	private ConcurrentHashMap<String, ActorInstance> actorInstances;

	@PostConstruct
	public void initialize() {
		logger.info(LOG_PREFIX + "initialize: Intializing Actor map");
		this.actorInstances = new ConcurrentHashMap<String, ActorInstance>();
		actorTypes = new HashMap<String, ActorType>();

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
										String key = method.getName() + ":" + method.getParameterCount();
										if (remoteMethods.containsKey(key)) {
											logger.severe("Unsupported static overload of " + method.getName() + ". Multiple overloads with "
													+ method.getParameterCount() + " arguments");
											logger.severe("Method " + method.toString() + " failed to be registered as a @Remote method");
										} else {
											logger
													.info(LOG_PREFIX + "initialize: adding " + key + " to remote methods for " + actorClassName);
											remoteMethods.put(method.getName() + ":" + method.getParameterTypes().length, mh);
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

							String karTypeName = nameList.get(classList.indexOf(actorClassName));
							ActorType at = new ActorType(karTypeName, actorClass, remoteMethods, activateMethod, deactivateMethod);
							actorTypes.put(karTypeName, at);
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

			logger.info(LOG_PREFIX + "initialize: actor map initialized with " + actorTypes.size() + " entries");
		}
	}

	public ActorInstance getActor(String type, String id) {
		return actorInstances.get(actorInstanceKey(type, id));
	}

	public ActorInstance createActor(String type, String id) {
		ActorType actorType = actorTypes.get(type);
		if (actorType == null) {
			return null;
		}

		Class<ActorInstance> actorClass = actorType.getActorClass();
		try {
			ActorInstance actorObj = actorClass.getConstructor().newInstance();
			actorObj.setType(type);
			actorObj.setId(id);
			actorInstances.put(actorInstanceKey(type, id), actorObj);
			return actorObj;
		} catch (Throwable t) {
			logger.severe(LOG_PREFIX + "createActor: " + t.toString());
			return null;
		}
	}

	public boolean deleteActor(String type, String id) {
		return actorInstances.remove(actorInstanceKey(type, id)) != null;
	}

	public boolean hasActorType(String type) {
		return this.actorTypes.containsKey(type);
	}

	public MethodHandle getActorMethod(String type, String name, int numParams) {
		ActorType actorType = this.actorTypes.get(type);
		return actorType != null ? actorType.getRemoteMethods().get(name + ":" + numParams) : null;
	}

	public MethodHandle getActorActivateMethod(String type) {
		ActorType actorType = this.actorTypes.get(type);
		return actorType != null ? actorType.getActivateMethod() : null;
	}

	public MethodHandle getActorDeactivateMethod(String type) {
		ActorType actorType = this.actorTypes.get(type);
		return actorType != null ? actorType.getDeactivateMethod() : null;
	}

	private String actorInstanceKey(String type, String instance) {
		return type + ":" + instance;
	}

}
