package com.ibm.research.kar.actor;

import javax.servlet.ServletContext;
import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;
import javax.servlet.annotation.WebListener;


/*
 * Will do initial load of supported actor classnames from web.xml `actor-list`
 */
@WebListener
public class ActorRuntimeContextListener implements ServletContextListener {
	
	public static final String KAR_ACTOR_CLASSES = "kar-actor-classes";
	public static final String KAR_ACTOR_TYPES = "kar-actor-types";	

	public static String actorClassStr;
	public static String actorTypeNameStr;
	
	@Override
	public void contextInitialized(final ServletContextEvent servletContextEvent) {
		ServletContext ctx = servletContextEvent.getServletContext();
		
		actorClassStr = ctx.getInitParameter(ActorRuntimeContextListener.KAR_ACTOR_CLASSES);
		actorTypeNameStr = ctx.getInitParameter(ActorRuntimeContextListener.KAR_ACTOR_TYPES);
	}

}
