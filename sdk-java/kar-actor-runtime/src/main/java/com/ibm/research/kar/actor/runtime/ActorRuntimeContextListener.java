package com.ibm.research.kar.actor.runtime;

import java.util.logging.Logger;

import javax.servlet.ServletContext;
import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;
import javax.servlet.annotation.WebListener;

import com.ibm.research.kar.KarConfig;

/*
 * Reads configuration information from web.xml
 */
@WebListener
public class ActorRuntimeContextListener implements ServletContextListener {

	public static Logger logger = Logger.getLogger(ActorRuntimeContextListener.class.getName());

	public static final String KAR_ACTOR_CLASSES = "kar-actor-classes";
	public static final String KAR_ACTOR_TYPES = "kar-actor-types";
	public static final String KAR_CONNECTION_TIMEOUT = "kar-connection-timeout-millis";

	@Override
	public void contextInitialized(final ServletContextEvent servletContextEvent) {
		ServletContext ctx = servletContextEvent.getServletContext();

		KarConfig.ACTOR_CLASS_STR = ctx.getInitParameter(ActorRuntimeContextListener.KAR_ACTOR_CLASSES);
		KarConfig.ACTOR_TYPE_NAME_STR = ctx.getInitParameter(ActorRuntimeContextListener.KAR_ACTOR_TYPES);

		String timeOut = ctx.getInitParameter(ActorRuntimeContextListener.KAR_CONNECTION_TIMEOUT);
		if (timeOut != null) {
			try {
				logger.info("Setting default connection timeout to " + timeOut);
				KarConfig.DEFAULT_CONNECTION_TIMEOUT_MILLIS = Integer.parseInt(timeOut);
			} catch (NumberFormatException ex) {
				ex.printStackTrace();
			}
		}

		if (System.getenv("KAR_RUNTIME_PORT") == null) {
			logger.severe("KAR_RUNTIME_PORT is not set.  YOUR APPLICATION IS BADLY MISCONFIGURED AND WILL NOT WORK!");
			new Exception().printStackTrace();
		}
		if (System.getenv("KAR_APP_PORT") == null) {
			logger.severe("KAR_APP_PORT is not set.  YOUR APPLICATION IS BADLY MISCONFIGURED AND WILL NOT WORK!");
			new Exception().printStackTrace();
		}
	}

}
