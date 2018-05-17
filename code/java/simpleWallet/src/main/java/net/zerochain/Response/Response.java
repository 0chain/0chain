package net.zerochain.Response;

import java.io.Serializable;

public class Response implements Serializable
{
	private static final long serialVersionUID = 1L;

	private String name;
	private String message;

	public Response()
	{
		this.name = "";
		this.message = "";
	}

	public Response(String name, String message)
	{
		this.name = name;
		this.message = message;
	}

	public void setName(String name)
	{
		this.name = name;
	}

	public String getName()
	{
		return name;
	}

	public void setMessage(String message)
	{
		this.message = message;
	}

	public String getMessage()
	{
		return message;
	}

	@Override
	public String toString()
	{
		return "Response{" +
		" name='" + name + '\'' +
		", message='" + message + '\'' + 
		'}';
	}
}