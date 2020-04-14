package it.com.ibm.research.kar.example;

import com.ibm.research.kar.example.Number;
import com.ibm.research.kar.example.NumberResource;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import org.junit.jupiter.api.Test;
import org.microshed.testing.SharedContainerConfig;
import org.microshed.testing.jaxrs.RESTClient;
import org.microshed.testing.jupiter.MicroShedTest;

@MicroShedTest
@SharedContainerConfig(AppDeploymentConfig.class)
public class IncrServerIT {
	
    @RESTClient
    public static NumberResource numResource;
    
    @Test
    public void testIncrNumber() {
    	
    	Number oldNum = new Number();
    	oldNum.setNumber(54);
    	
    	Number newNum = numResource.incrNumber(oldNum);
    	
    	assertNotNull(newNum, "Number resource returns null");
    	assertEquals(newNum.getNumber(), oldNum.getNumber() + 1, "Number resource doesn't know math");
    }
    
    @Test
    public void testFake() {
    	assertEquals(1,1);
    }

}
