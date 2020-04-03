package it.com.ibm.research.kar;

import org.microshed.testing.SharedContainerConfiguration;
import org.microshed.testing.testcontainers.ApplicationContainer;
import org.testcontainers.junit.jupiter.Container;

public class AppDeploymentConfig implements SharedContainerConfiguration {

    @Container
    public static ApplicationContainer app = new ApplicationContainer()
                    .withAppContextRoot("/Example")
                    .withExposedPorts(9090)
                    .withReadinessPath("/health/ready");

}
