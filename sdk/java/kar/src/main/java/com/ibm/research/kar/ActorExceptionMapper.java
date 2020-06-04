package com.ibm.research.kar;

import java.util.logging.Logger;

import javax.ws.rs.core.Response;
import javax.ws.rs.ext.Provider;

import org.eclipse.microprofile.rest.client.ext.ResponseExceptionMapper;

@Provider
public class ActorExceptionMapper implements ResponseExceptionMapper<ActorMethodNotFoundException>{
	
	private static Logger logger = Logger.getLogger(ActorExceptionMapper.class.getName());

	@Override
	public ActorMethodNotFoundException toThrowable(Response response) {
		logger.info("handles(): got status code " + response.getStatus());
        return new ActorMethodNotFoundException("Cannot find requested method");
	}

}
