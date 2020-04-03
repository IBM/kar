package it.com.ibm.research.kar;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.core.Response;

import org.junit.jupiter.api.Test;
import org.microshed.testing.SharedContainerConfig;
import org.microshed.testing.jaxrs.RESTClient;
import org.microshed.testing.jupiter.MicroShedTest;

import com.ibm.research.kar.KarParams;
import com.ibm.research.kar.example.client.ClientResource;

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
		Map<String,Object> numMap = new HashMap<String,Object>();
		numMap.put("number", 41);
		params.service = "number";
		params.path = incr;
		params.params = numMap;
		
		return params;
	}
    
}
