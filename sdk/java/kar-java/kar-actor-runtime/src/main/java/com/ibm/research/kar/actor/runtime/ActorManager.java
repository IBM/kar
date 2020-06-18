package com.ibm.research.kar.actor.runtime;

import java.lang.invoke.MethodHandle;

import com.ibm.research.kar.actor.ActorInstance;
import com.ibm.research.kar.actor.exceptions.ActorCreateException;

public interface ActorManager {
	// create actor instance
	public ActorInstance createActor(String type, String id) throws ActorCreateException;

	// delete actor instance
	public void deleteActor(String type, String id);

	// get existing or create new actor instance
	public ActorInstance getActor(String type, String id);

	public MethodHandle getActorMethod(String type, String name);
}
