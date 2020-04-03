import java.util.Map;

import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.ProcessingException;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.eclipse.microprofile.rest.client.inject.RegisterRestClient;

import com.ibm.research.kar.example.client.Number;

@RegisterRestClient(configKey = "kar", baseUri = "http://localhost:3500")
@Path("kar")
public interface Kar extends AutoCloseable {

	@GET
	@Produces(MediaType.APPLICATION_JSON)
	Number getNumber()  throws ProcessingException;
	
	@POST
	@Path("send/{service}/{path}")
	public Response send(@PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;
	
	@POST
	@Path("call/{service}/{path}")
	public Response call(@PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

	/*	
    @POST
	@Path("session/{actor}/send/{service}/{path}")
	public Response actorSend(@PathParam("actor") String actor, @PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

    @POST
	@Path("session/{actor}/call/{service}/{path}")
	public Response actorCall(@PathParam("actor") String actor, @PathParam("service") String service, @PathParam("path") String path, Map<String,Object> params) throws ProcessingException;

    @POST
	@Path("broadcast/${path}")
	public Response broadcast(@PathParam("path") String path, Map<String,Object> params) throws ProcessingException;*/

}