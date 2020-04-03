package com.ibm.research.kar.example;


/**
 * POJO Class to parse number objects
 *
 */
public class Number {
	
	int number;
	
	public Number() {
	}
	
	public int getNumber() {
		return number;
	}
	
	public void setNumber(int number) {
		this.number = number;
	}
	

    @Override
    public String toString(){
        return "Number is " + this.number;
    }
}
