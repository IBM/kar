package it.com.ibm.research.kar;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;


import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import org.junit.jupiter.api.Test;
import org.microshed.testing.SharedContainerConfig;
import org.microshed.testing.jaxrs.RESTClient;
import org.microshed.testing.jupiter.MicroShedTest;

import com.ibm.research.kar.example.client.ClientResource;
import com.ibm.research.kar.example.client.KarParams;

@MicroShedTest
@SharedContainerConfig(AppDeploymentConfig.class)
public class KarClienttIT {
	
	@RESTClient
	public static ClientResource client;

	@Test
	public void testCall() {
		
		KarParams params = getKarParams("incrSync");
		Response resp = client.call(params);
		
		System.out.println("Got Response " + resp.getStatus());
		
		assertNotNull(resp);
		assertEquals(200, resp.getStatus(), "Call should return with status OK");
		
		resp.close();
	}
	
	@Test
	public void testTell() {
		
		KarParams params = getKarParams("incrAsync");
		Response resp = client.tell(params);
		
		assertNotNull(resp);
		assertEquals(200, resp.getStatus(), "Tell should return with status OK");
		
		resp.close();
	}

	
	private KarParams getKarParams(String incr) {
		KarParams params = new KarParams();
		
		JsonObject numMap = Json.createObjectBuilder()
				.add("number", 41)
				.build();
		
		params.params = numMap;
		
		return params;
	}
    
}
