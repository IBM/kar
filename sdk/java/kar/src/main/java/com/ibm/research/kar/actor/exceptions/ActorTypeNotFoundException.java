package com.ibm.research.kar.actor.exceptions;

public class ActorTypeNotFoundException extends BaseActorException { 

	public ActorTypeNotFoundException() {
		super();
	}
	public ActorTypeNotFoundException(String errorMessage) {
        super(errorMessage);
    }
}