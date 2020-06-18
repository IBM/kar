package com.ibm.research.kar.actor.exceptions;

public class ActorMethodNotFoundException extends BaseActorException { 

	public ActorMethodNotFoundException() {
		super();
	}
	public ActorMethodNotFoundException(String errorMessage) {
        super(errorMessage);
    }
}