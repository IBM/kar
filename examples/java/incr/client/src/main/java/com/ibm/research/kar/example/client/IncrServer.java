package com.ibm.research.kar.example.client;

import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.Produces;
import org.eclipse.microprofile.rest.client.annotation.RegisterProvider;
import org.eclipse.microprofile.rest.client.inject.RegisterRestClient;

import com.ibm.research.kar.KarRest;


@RegisterRestClient(configKey = "defaultClient", baseUri = "http://localhost:9080/")
@RegisterProvider(UnknownUriExceptionMapper.class)
@Path("number")
public interface IncrServer extends AutoCloseable{
	
	@POST
	@Path("incr")
	@Produces(KarRest.KAR_ACTOR_JSON) 
	Number incrNumber(Number num) throws UnknownUriException, ProcessingException;
	
	@GET
	@Produces(KarRest.KAR_ACTOR_JSON)
	Number getNumber()  throws UnknownUriException, ProcessingException;

}
