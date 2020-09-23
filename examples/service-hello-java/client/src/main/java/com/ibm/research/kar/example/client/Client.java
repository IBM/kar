package com.ibm.research.kar.example.client;

import java.io.IOException;

import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.client.ClientProtocolException;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.client.DefaultHttpClient;
import org.apache.http.util.EntityUtils;

public class Client {

  public static void main(String[] args) throws ClientProtocolException, IOException {
    DefaultHttpClient httpClient = new DefaultHttpClient();
    String KAR_RUNTIME_PORT = System.getenv("KAR_RUNTIME_PORT");
    try {

      // Hit the helloText route of the greeter service
      HttpPost postRequest = new HttpPost("http://127.0.0.1:"+KAR_RUNTIME_PORT+"/kar/v1/service/greeter/call/helloText");
      postRequest.addHeader("content-type", "text/plain");

      StringEntity userEntity = new StringEntity("Gandalf the Grey", "UTF-8");
      postRequest.setEntity(userEntity);

      HttpResponse response = httpClient.execute(postRequest);

      int statusCode = response.getStatusLine().getStatusCode();
      if (statusCode != 200) {
        throw new RuntimeException("Unexpected HTTP status code : " + statusCode);
      }

      HttpEntity entity = response.getEntity();
      String msg = EntityUtils.toString(entity);
      System.out.println(msg);

      String expected = "Hello Gandalf the Grey";
      if (!msg.equals(expected)) {
        throw new RuntimeException("Test FAILED: expected `"+expected+"` but got `"+msg+"`");
      }

      System.out.println("SUCCESS!");

    } finally {
      httpClient.getConnectionManager().shutdown();
    }
  }
}
