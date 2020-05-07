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
import javax.servlet.ServletContext;
import javax.ws.rs.core.Context;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Singleton
@ConcurrencyManagement(ConcurrencyManagementType.CONTAINER)
@ApplicationScoped
public class ActorManagerImpl implements ActorManager {

	@Context ServletContext ctx;

	private Map<String, ActorReference> actorMap;
	private Map<String, ActorReference> actorInstanceMap;

	@PostConstruct
	public void initialize() {

		System.out.println("ActorManagerImpl.initialize: Initializing the Actor Map");
		actorMap = new HashMap<String, ActorReference>();
		actorInstanceMap = new HashMap<String, ActorReference>();

		String actorClassStr = ActorRuntimeContextListener.actorClassStr;
		String actorTypeNameStr = ActorRuntimeContextListener.actorTypeNameStr;

		System.out.println("ActorManagerImpl.initialize: Got init params " + actorClassStr + ":"+ actorTypeNameStr);

		if ((actorClassStr != null) && (actorTypeNameStr != null)) {
			System.out.println("Parsing list");
			List<String> classList = Arrays.asList(actorClassStr.split("\\s*,\\s*"));
			List<String> nameList = Arrays.asList(actorTypeNameStr.split("\\s*,\\s*"));

			System.out.println("ActorManagerImpl.initialize: class size:" + classList.size() + " nameList.size:"+ nameList.size());
			if (classList.size() == nameList.size()) {
				for (String actorClassName : classList) {
					try {
						Class<?> cls = Class.forName(actorClassName);
						Annotation annotation = cls.getAnnotation(Actor.class);
						if (annotation != null) {

							Method[] methods = cls.getMethods();
							Map<String,Method> remoteMethods = new HashMap<String,Method>();
							Method activateMethod = null;
							Method deactivateMethod = null;

							for (Method method : methods) {
								if (method.isAnnotationPresent(Remote.class)) {
									remoteMethods.put(method.getName(),method);
								} else if (method.isAnnotationPresent(Activate.class)) {
									activateMethod = method;
								} else if (method.isAnnotationPresent(Deactivate.class)) {
									deactivateMethod = method;
								}
								
							}

							// only add actor to map if it can be activated and deactivated
							if ((activateMethod != null) && (deactivateMethod != null)) {

								ActorReference actorRef = new ActorReference();

								String karTypeName = nameList.get(classList.indexOf(actorClassName));
								actorRef.setType(karTypeName);
								actorRef.setCls(cls);
								actorRef.setClassName(actorClassName);

								actorRef.setActivateMethod(activateMethod);
								actorRef.setDeactivateMethod(deactivateMethod);

								actorRef.setRemoteMethods(remoteMethods);

								actorMap.put(karTypeName, actorRef);
							} else {
								System.out.println("ActorManagerImpl.initialize: Actor class " + actorClassName + " not added to runtime because it does not contain required initialization or de-initialization methods");
							}
						}
					} catch (ClassNotFoundException e) {
						e.printStackTrace();
					}
				}
			}
		}

		System.out.println("ActorManagerImpl.initialize: initialized with " + actorMap.size() + "actor references");
	}

	@Lock(LockType.WRITE)
	public ActorReference createActor(String type, String id) {
		ActorReference actorRef = actorMap.get(type);

		if (actorRef != null) {
			Class<?> cls = actorRef.getCls();
			try {
				Object actorObj = cls.getConstructor().newInstance();

				ActorReference inst = new ActorReference();
				inst.setClassName(actorRef.getClassName());
				inst.setType(type);
				inst.setId(id);
				inst.setActorInstance(actorObj);
				inst.setRemoteMethods(actorRef.getRemoteMethods());
				inst.setActivateMethod(actorRef.getActivateMethod());
				inst.setDeactivateMethod(actorRef.getDeactivateMethod());
				
				Method activate = actorRef.getActivateMethod();
				activate.invoke(actorObj);

				actorInstanceMap.put(type+":"+id, inst);

			} catch (InstantiationException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (IllegalAccessException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (IllegalArgumentException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (InvocationTargetException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (NoSuchMethodException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (SecurityException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			}
		}
		return actorRef;
	}

	@Lock(LockType.WRITE)
	public void deleteActor(String type, String id) {
		ActorReference actorRef = actorInstanceMap.get(type+":"+id);
		Object actorObj = actorRef.getActorInstance();
		Method deactivateMethod = actorRef.getDeactivateMethod();
		
		System.out.println("ActorManagerImpl.deleteActor: actor object is " + actorObj);

		System.out.println("ActorManagerImpl.deleteActor: deactivate method is " + deactivateMethod);
		
		try {
			deactivateMethod.invoke(actorObj);
		} catch (IllegalAccessException | IllegalArgumentException | InvocationTargetException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
		
		actorInstanceMap.remove(type+":"+id);
	}

	@Lock(LockType.READ) 
	public ActorReference getActor(String type, String id) {
		ActorReference actorRef = actorInstanceMap.get(type+":"+id);
		if (actorRef == null) {
			actorRef = createActor(type,id);
		}
		return actorRef;
	}

	@Lock(LockType.READ) 
	public int getNumActors() {
		System.out.println("ActorManagerImpl.getNumActors: checking actor map for size");

		if (actorMap != null) {
			return actorMap.size();
		} else {
			System.out.println("ActorManagerImpl.getNumActors: no map instance found");
			return 0;
		}
	}


}
