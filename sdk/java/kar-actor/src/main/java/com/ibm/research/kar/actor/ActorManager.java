package com.ibm.research.kar.actor;

import com.ibm.research.kar.ActorInstance;

public interface ActorManager {
	// create actor instance
	public ActorInstance createActor(String type, String id);

	// delete actor instance
	public void deleteActor(String type, String id);

	// get existing or create new actor instance
	public ActorInstance getActor(String type, String id);

	public RemoteMethod getActorMethod(String type, String name);
}
