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
 * of KAR Actors and for invoking methods on them as requested by the applicaiton.
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
	 * Activate an actor instance if it is not already in memory. If the specified
	 * instance is in memory already, simply return success. If the specified
	 * instance it not in memory already, allocate a language-level instance for it
	 * and, if provided by the ActorType, invoke the optional activate method.
	 *
	 * @param type The type of the actor instance to be activated
	 * @param id   The id of the actor instance to be activated
	 * @return A Response indicating success (200, 201) or an error condition (400,
	 *         404)
	 */
	public static KarResponse activateInstanceIfNotPresent(String type, String id) {
		if (actorInstances.get(actorInstanceKey(type, id)) != null) {
			// Already exists; nothing to do.
			return new KarResponse(KarResponse.OK);
		}

		// Find the ActorType
		ActorType actorType = actorTypes.get(type);
		if (actorType == null) {
			return new KarResponse(KarResponse.NOT_FOUND, KarResponse.TEXT_PLAIN, "Not found: " + type + " actor " + id);
		}

		// Allocate an instance
		Class<ActorInstance> actorClass = actorType.getActorClass();
		ActorInstance actorObj;
		try {
			actorObj = actorClass.getConstructor().newInstance();
			actorObj.setType(type);
			actorObj.setId(id);
			actorInstances.put(actorInstanceKey(type, id), actorObj);
		} catch (Throwable t) {
			logger.severe(LOG_PREFIX + "activateInstanceIfNotPresent: " + t.toString());
			return new KarResponse(KarResponse.BAD_REQUEST, KarResponse.TEXT_PLAIN, t.toString());
		}

		// Call the optional activate method
		try {
			MethodHandle activate = actorType.getActivateMethod();
			if (activate != null) {
				activate.invoke(actorObj);
			}
			return new KarResponse(KarResponse.CREATED, KarResponse.TEXT_PLAIN, "Created " + type + " actor " + id);
		} catch (Throwable t) {
			return new KarResponse(KarResponse.BAD_REQUEST, KarResponse.TEXT_PLAIN, t.toString());
		}
	}


	/**
	 * Deactivate an actor instance. If the ActorType has a deactivate method, it
	 * will be invoked on the instance. The actor instance will be removed from the
	 * in-memory state of the runtime.
	 *
	 * @param type The type of the actor instance to be deactivated
	 * @param id   The id of the actor instance to be deactivated
	 * @return A Response indicating success (200) or an error condition (400, 404)
	 */
	public static KarResponse deactivateInstanceIfPresent(String type, String id) {
		ActorInstance actorObj = actorInstances.get(actorInstanceKey(type, id));
		if (actorObj == null) {
			return new KarResponse(KarResponse.NOT_FOUND, KarResponse.TEXT_PLAIN, "Not found: " + type + " actor " + id);
		}

		// Call the optional deactivate method
		ActorType actorType = actorTypes.get(type);
		if (actorType != null && actorType.getDeactivateMethod() != null) {
			try {
				actorType.getDeactivateMethod().invoke(actorObj);
			} catch (Throwable t) {
				return new KarResponse(KarResponse.BAD_REQUEST, KarResponse.TEXT_PLAIN, t.toString());
			}
		}

		// Actually remove the instance
		actorInstances.remove(actorInstanceKey(type, id));

		return new KarResponse(KarResponse.OK);
	}


	/**
	 * Invoke an actor method
	 *
	 * @param type The type of the actor
	 * @param id The id of the target instancr
	 * @param sessionid The session in which the method is being invoked
	 * @param path The method to invoke
	 * @param args The arguments to the method
	 * @return a Response containing the result of the method invocation
	 */
	public static KarResponse invokeActorMethod(String type, String id, String sessionid, String path, Object[] actuals) {
		ActorInstance actorObj = actorInstances.get(actorInstanceKey(type, id));
		if (actorObj == null) {
			return new KarResponse(KarResponse.NOT_FOUND, KarResponse.TEXT_PLAIN, "Actor instance not found: " + type + "[" + id +"]");
		}

		int nParams = actuals.length-1; // Java convention "this" doesn't count as an argument
		ActorType actorType = actorTypes.get(type);
		MethodHandle actorMethod = actorType != null ? actorType.getRemoteMethods().get(path + ":" + nParams) : null;
		if (actorMethod == null) {
			return new KarResponse(KarResponse.NOT_FOUND, KarResponse.TEXT_PLAIN, "Method not found: " + type + "." + path + " with " + nParams + " arguments");
		}

		// set the session
		actorObj.setSession(sessionid);

		// inject the actorObject into the argument array
		actuals[0] = actorObj;

		try {
			Object result = actorMethod.invokeWithArguments(actuals);
			if (result == null && actorMethod.type().returnType().equals(Void.TYPE)) {
				return new KarResponse(KarResponse.NO_CONTENT);
			} else {
				return new KarResponse(KarResponse.OK, KarResponse.KAR_ACTOR_JSON, new ActorInvokeResult(result));
			}
		} catch (Throwable t) {
			if (KarConfig.SHORTEN_ACTOR_STACKTRACES) {
				// Elide all of the implementation details above us in the backtrace
				StackTraceElement [] fullBackTrace = t.getStackTrace();
				for (int i=0; i<fullBackTrace.length; i++) {
					if (fullBackTrace[i].getClassName().equals(ActorManager.class.getName()) && fullBackTrace[i].getMethodName().equals("invokeActorMethod")) {
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
			return new KarResponse(KarResponse.OK, KarResponse.KAR_ACTOR_JSON, new ActorInvokeResult(t.getMessage(), sw.toString()));
		}
	}

	private static String actorInstanceKey(String type, String instance) {
		return type + ":" + instance;
	}
}
