package it.com.ibm.research.kar.example;

import static org.junit.jupiter.api.Assertions.assertEquals;

import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.core.Response;

import org.junit.jupiter.api.Test;
import org.microshed.testing.SharedContainerConfig;
import org.microshed.testing.jaxrs.RESTClient;
import org.microshed.testing.jupiter.MicroShedTest;

import com.ibm.research.kar.example.KarResource;

@MicroShedTest
@SharedContainerConfig(AppDeploymentConfig.class)
public class KarServerIT {
	
    @RESTClient
    public static KarResource karResource;
    
    @Test
    public void testCall() {
    	
    	Map<String,Object> params = new HashMap<String,Object>();
    	params.put("number",23);
    	
    	Response resp = karResource.call(params, "number", "incr");
    
    	assertEquals(resp.getStatus(), 200);
    	
    }
    
    
    @Test
    public void testTell() {
    	
    	Map<String,Object> params = new HashMap<String,Object>();
    	params.put("number",23);
    	
    	Response resp = karResource.tell(params, "number", "incr");
    
    	assertEquals(resp.getStatus(), 200);
    	
    }
    
}
