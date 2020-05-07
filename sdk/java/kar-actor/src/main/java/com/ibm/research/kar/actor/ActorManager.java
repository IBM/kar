package com.ibm.research.kar.actor;

public interface ActorManager {
	public ActorReference createActor(String type, String id);
	public void deleteActor(String type, String id);
	public ActorReference getActor(String type, String id);
	public int getNumActors();
}
