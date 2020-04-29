package kar;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import java.io.StringReader;

import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonReader;
import javax.json.JsonValue;
import javax.json.JsonValue.ValueType;
import javax.json.JsonNumber;
import javax.ws.rs.core.Response;
import org.junit.jupiter.api.Test;

import com.ibm.research.kar.Kar;

public class KarTests {


	@Test
	void testCall() {
		Kar kar = new Kar();
		kar.buildRestClient();

		JsonObject params = Json.createObjectBuilder()
				.add("number", 5)
				.build();

		Response resp = kar.call("number", "incr", params);

		String replyString = resp.readEntity(String.class);
		JsonReader jsonReader = Json.createReader(new StringReader(replyString));
		JsonObject reply = jsonReader.readObject();

		assertNotNull(resp);

		JsonValue value = reply.get("number");

		assertNotNull(value);

		ValueType type = value.getValueType();

		assertEquals(type, ValueType.NUMBER);
		int num = ((JsonNumber)value).intValue();
		assertEquals(6,num);
	}

}
