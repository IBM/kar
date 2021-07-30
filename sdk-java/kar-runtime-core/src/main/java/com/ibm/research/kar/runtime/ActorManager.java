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

package com.ibm.research.kar.runtime;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.reflect.Method;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.logging.Logger;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

public class ActorManager {

	private final static String LOG_PREFIX = "ActorManager.";
	private final static Logger logger = Logger.getLogger(ActorManager.class.getName());

	// This map is read only once it is initialized.
	private static final HashMap<String, ActorType> actorTypes = new HashMap<>();
	// This map is concurrently updated.
	private static final ConcurrentHashMap<String, ActorInstance> actorInstances = new ConcurrentHashMap<>();

	/**
	 * Build the ActorType meta-data that caches the Class and MethodHandle objects
	 * that are needed to instantiate Actor instances and invoke Actor methods.
	 *
	 * @param classList the list of Java classes that are actors
	 * @param nameList the list of KAR actor types used to refer to the classes
	 */
	public static void initialize(List<String> classList, List<String> nameList) {
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
									logger.info(LOG_PREFIX + "initialize: adding " + key + " to remote methods for " + actorClassName);
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
								logger.severe(LOG_PREFIX + "initialize: IllegalAccessException adding activate to " + actorClassName);
							}
						} else if (method.isAnnotationPresent(Deactivate.class)) {
							try {
								deactivateMethod = lookup.unreflect(method);
							} catch (IllegalAccessException e) {
								logger.severe(LOG_PREFIX + "initialize: IllegalAccessException adding deactivate to " + actorClassName);
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
						logger.severe(
								LOG_PREFIX + "initialize: " + actorClassName + " does not implement " + ActorInstance.class.getName());
					}
				}
			} catch (ClassNotFoundException e) {
				e.printStackTrace();
				System.out.print(LOG_PREFIX + "initialize: Cannot log class " + actorClassName);
			}
		}

		logger.info(LOG_PREFIX + "initialize: actor map initialized with " + actorTypes.size() + " entries");
	}

	public static ActorInstance getActor(String type, String id) {
		return actorInstances.get(actorInstanceKey(type, id));
	}

	public static ActorInstance createActor(String type, String id) {
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

	public static boolean deleteActor(String type, String id) {
		return actorInstances.remove(actorInstanceKey(type, id)) != null;
	}

	public static boolean hasActorType(String type) {
		return actorTypes.containsKey(type);
	}

	public static MethodHandle getActorMethod(String type, String name, int numParams) {
		ActorType actorType = actorTypes.get(type);
		return actorType != null ? actorType.getRemoteMethods().get(name + ":" + numParams) : null;
	}

	public static MethodHandle getActorActivateMethod(String type) {
		ActorType actorType = actorTypes.get(type);
		return actorType != null ? actorType.getActivateMethod() : null;
	}

	public static MethodHandle getActorDeactivateMethod(String type) {
		ActorType actorType = actorTypes.get(type);
		return actorType != null ? actorType.getDeactivateMethod() : null;
	}

	private static String actorInstanceKey(String type, String instance) {
		return type + ":" + instance;
	}
}
