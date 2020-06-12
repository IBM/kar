package com.ibm.research.kar.example;

import java.math.BigDecimal;

public class NumberService {
	
	public BigDecimal incr(BigDecimal oldNum) {
		return  new BigDecimal(oldNum.intValue() + 1);
	}
	
}
