package com.ibm.research.kar.actor;

import java.lang.annotation.Annotation;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import javax.annotation.PostConstruct;
import javax.ejb.ConcurrencyManagement;
import javax.ejb.ConcurrencyManagementType;
import javax.ejb.Lock;
import javax.ejb.LockType;
import javax.ejb.Singleton;

import javax.enterprise.context.ApplicationScoped;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Singleton
@ApplicationScoped
@ConcurrencyManagement(ConcurrencyManagementType.CONTAINER)
public class ActorManagerImpl implements ActorManager {

	private final static String LOG_PREFIX = "ActorManagerImpl.";

	private Map<String, ActorModel> actorMap;

	@PostConstruct
	public void initialize() {
		System.out.println(LOG_PREFIX+"initialize: Intializing Actor map");
		this.actorMap = new HashMap<String, ActorModel>();

		System.out.println(LOG_PREFIX+"initialize: Got init params " + ActorRuntimeContextListener.actorClassStr + ":"+ ActorRuntimeContextListener.actorTypeNameStr);

		// ensure that we have non-null class and kar type strings from web.xml
		if ((ActorRuntimeContextListener.actorClassStr != null) && (ActorRuntimeContextListener.actorTypeNameStr != null)) {
			List<String> classList = Arrays.asList(ActorRuntimeContextListener.actorClassStr.split("\\s*,\\s*"));
			List<String> nameList = Arrays.asList(ActorRuntimeContextListener.actorTypeNameStr.split("\\s*,\\s*"));


			// lists should be same size
			if (classList.size() == nameList.size()) {

				// Create ActorModel for each class
				for (String actorClassName : classList) {

					// class should be annotated with @Actor, otherwise reject
					try {
						Class<?> cls = Class.forName(actorClassName);
						Annotation annotation = cls.getAnnotation(Actor.class);

						// if annotation is present, get annotated methods for
						// 1. remote 
						// 2. activate
						// 3. deactivate
						if (annotation != null) {

							Method[] methods = cls.getMethods();
							Map<String,Method> remoteMethods = new HashMap<String,Method>();
							Method activateMethod = null;
							Method deactivateMethod = null;

							for (Method method : methods) {
								if (method.isAnnotationPresent(Remote.class)) {
									System.out.print(LOG_PREFIX+"initialize: adding method " + method.getName() + " to remote methods");
									remoteMethods.put(method.getName(),method);
								} else if (method.isAnnotationPresent(Activate.class)) {
									activateMethod = method;
								} else if (method.isAnnotationPresent(Deactivate.class)) {
									deactivateMethod = method;
								}

							}
							// create new ActorModel
							ActorModel actorRef = new ActorModel();

							String karTypeName = nameList.get(classList.indexOf(actorClassName));

							// add kar type and class name for future (?) bookeeping
							actorRef.setType(karTypeName);
							actorRef.setClassName(actorClassName);

							// add methods so we don't have to look them up later
							actorRef.setActivateMethod(activateMethod); // ok to be null
							actorRef.setDeactivateMethod(deactivateMethod); // ok to be null
							actorRef.setRemoteMethods(remoteMethods); // ok to be empty

							// put new ActorModel in ActorMap with KAR type as key
							actorMap.put(karTypeName, actorRef);		
						}
					} catch (ClassNotFoundException e) {
						e.printStackTrace();
						System.out.print(LOG_PREFIX+"initialize: Cannot log class " + actorClassName);
					}
				}

			}

			System.out.println(LOG_PREFIX + "initialize: actor map initialized with " + actorMap.size() + " entries");
		}
	}


	@Lock(LockType.WRITE)
	public Object createActor(String type, String id) {
		System.out.println(LOG_PREFIX + "createActor ActorManager");


		ActorModel actorRef = actorMap.get(type);
		Object actorObj = null;

		if (actorRef != null) {
			Class<?> actorClass = actorRef.getActorClass();

			try {
				actorObj = actorClass.getConstructor().newInstance();

				// initialize actor instance
				Method activate = actorRef.getActivateMethod();

				if (activate != null) {
					activate.invoke(actorObj);
				}

				// put reference to actorObj in the ActorModel
				actorRef.getActorInstances().put(id, actorObj);


			} catch (InstantiationException | IllegalAccessException | IllegalArgumentException
					| InvocationTargetException | NoSuchMethodException | SecurityException e) {
				e.printStackTrace();
			}
		} 

		return actorObj;
	}

	@Lock(LockType.WRITE)
	public void deleteActor(String type, String id) {
		System.out.println(LOG_PREFIX + "deleteActor: deleting " + type + " actor " + id);
		ActorModel actorRef = this.actorMap.get(type);

		if (actorRef != null) {
			Object actorObj = actorRef.getActorInstances().get(id);

			if (actorObj != null) {
				Method deactivateMethod = actorRef.getDeactivateMethod();
				if (deactivateMethod != null) {
					try {
						deactivateMethod.invoke(actorObj);
					} catch (IllegalAccessException | IllegalArgumentException | InvocationTargetException e) {
						// TODO Auto-generated catch block
						e.printStackTrace();
						System.out.println(LOG_PREFIX + "deleteActor: error executing actor deactivate method");
					}
				}
				actorRef.getActorInstances().remove(actorObj);
			} else {
				System.out.println(LOG_PREFIX + "deleteActor: warning, no instance found for actor " + id);
			}
		} else {
			System.out.println(LOG_PREFIX + "deleteActor: warning, no model found for " + type + " actor");
		}
	}

	@Lock(LockType.READ) 
	public Object getActor(String type, String id) {

		System.out.println(LOG_PREFIX+"getActor: Retrieving actor instance");  

		ActorModel model = this.actorMap.get(type);
		Object actorObj = null;

		if (model != null) {
			actorObj = model.getActorInstances().get(id);
		}

		return actorObj;
	}

	@Lock(LockType.READ) 
	public int getNumActors() {
		System.out.println("ActorManagerImpl.getNumActors: checking actor map for size");

		if (actorMap != null) {
			return actorMap.size();
		} else {
			System.out.println(LOG_PREFIX + "getNumActors: no map instance found");
			return 0;
		}
	}


	@Override
	public Method getActorMethod(String type, String name) {
		System.out.println(LOG_PREFIX + "getactorMethod: getting method " + name + " for " + type + " actor");
		ActorModel model = this.actorMap.get(type);

		System.out.println(LOG_PREFIX + "getactorMethod: found actor model " + model);

		Method method = null;


		if (model != null) {
			method = model.getRemoteMethods().get(name);
		} else {
			System.out.println(LOG_PREFIX+"getActorMethod: Warning, no model of type " + type + " found");
		}

		return method; 
	}




}
