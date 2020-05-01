package com.ibm.research.kar.actor;

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class MyActor {

	@Activate
	public void init() {
		
	}
	
	@Remote
	public void canBeInvoked() {
		
	}
	
	public void cannotBeInvoked() {
		
	}
	
	@Deactivate
	public void kill() {
		
	}
}
