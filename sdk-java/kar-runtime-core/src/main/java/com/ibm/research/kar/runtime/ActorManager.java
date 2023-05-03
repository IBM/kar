/*
 * Copyright IBM Corporation 2020,2023
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

import java.io.PrintWriter;
import java.io.StringWriter;
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

/**
 * The ActorManager is responsible for creating and removing the in-memory instances
 * of KAR Actors and for managing the reflective meta-data stored in ActorTypes.
 */
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
	 * @param nameList  the list of KAR actor types used to refer to the classes
	 */
	public static void initialize(List<String> classList, List<String> nameList) {
		ClassLoader cl = Thread.currentThread().getContextClassLoader();
		MethodHandles.Lookup lookup = MethodHandles.lookup();
		for (String actorClassName : classList) {
			try {
				Class<?> cls = Class.forName(actorClassName, true, cl);
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


	/**
	 * Is the given type a known ActorType?
	 * @param type the type to look for
	 * @return <code>true</code> if type is a known actor type and <code>false</code> otherwise.
	 */
	public static boolean hasActorType(String type) {
		return actorTypes.containsKey(type);
	}

	/**
	 * Get the ActorType for a type
	 * @param type
	 * @return the requested actor type or null
	 */
	public static ActorType getActorType(String type) {
		return actorTypes.get(type);
	}

	/**
	 * Get the requested instance if it is already in memory
	 * @param type
	 * @param id
	 * @return The Actor instance or null
	 */
	public static ActorInstance getInstanceIfPresent(String type, String id) {
		return actorInstances.get(actorInstanceKey(type, id));
	}

	/**
	 * Remove the requested instance if it is in memory
	 * @param type
	 * @param id
	 * @return true if remove, false otherwise
	 */
	public static boolean removeInstanceIfPresent(String type, String id) {
		return actorInstances.remove(actorInstanceKey(type, id)) != null;
	}

	/**
	 * Allocate a fresh actor instance of the given type and id; the activate method is not invoked.
	 * @param type
	 * @param id
	 * @return
	 */
	public static ActorInstance allocateFreshInstance(ActorType type, String id) {
		Class<ActorInstance> actorClass = type.getActorClass();
		ActorInstance actorObj;
		try {
			actorObj = actorClass.getConstructor().newInstance();
			actorObj.setType(type.getType());
			actorObj.setId(id);
			actorInstances.put(actorInstanceKey(type.getType(), id), actorObj);
		} catch (Throwable t) {
			logger.severe(LOG_PREFIX + "allocateFreshInstance: " + t.toString());
			return null;
		}
		return actorObj;
	}

	/**
	 * Filter/truncate a Throwable's stacktrace to elide implementation details of actor method invocation
	 */
	public static String stacktraceToString(Throwable t, String filterClass, String filterMethod) {
		if (KarConfig.SHORTEN_ACTOR_STACKTRACES) {
			// Elide all of the implementation details above us in the backtrace
			StackTraceElement [] fullBackTrace = t.getStackTrace();
			for (int i=0; i<fullBackTrace.length; i++) {
				if (fullBackTrace[i].getClassName().equals(filterClass) && fullBackTrace[i].getMethodName().equals(filterMethod)) {
					StackTraceElement[] reducedBackTrace = new StackTraceElement[i+1];
					System.arraycopy(fullBackTrace, 0, reducedBackTrace, 0, i+1);
					t.setStackTrace(reducedBackTrace);
					break;
				}
			}
		}

		StringWriter sw = new StringWriter();
		PrintWriter pw = new PrintWriter(sw);
		t.printStackTrace(pw);
		String backtrace = sw.toString();
		if (backtrace.length() > KarConfig.MAX_STACKTRACE_SIZE) {
			backtrace = backtrace.substring(0, KarConfig.MAX_STACKTRACE_SIZE) + "\n...Backtrace truncated due to message length restrictions\n";
		}
		return backtrace;
	}

	private static String actorInstanceKey(String type, String instance) {
		return type + ":" + instance;
	}
}
