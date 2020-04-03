package com.ibm.research.kar.example;

public class NumberService {
	
	public int incr(int num) {
		num++;
		return num;
	}
	
	public Number getNum() {
		Number num = new Number();
		num.setNumber(7);
		
		return num;
	}

}
