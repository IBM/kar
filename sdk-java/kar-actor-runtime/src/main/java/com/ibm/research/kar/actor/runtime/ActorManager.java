package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;

import com.ibm.research.kar.actor.ActorInstance;

public interface ActorManager {
	// get an existing actor instance
	public ActorInstance getActor(String type, String id);

	// allocate an actor instance -- does not invoke activate
	public ActorInstance createActor(String type, String id);

	// delete an actor instance -- deos not invoke deactivate
	public boolean deleteActor(String type, String id);

	public boolean hasActorType(String type);

	public MethodHandle getActorMethod(String type, String name, int numParams);

	public MethodHandle getActorActivateMethod(String type);

	public MethodHandle getActorDeactivateMethod(String type);
}
