# server.py
from datetime import datetime
from mcp.server.fastmcp import FastMCP
from agent.graph import graph

# Create an MCP server
mcp = FastMCP("MyResearch")

# https://github.com/google-gemini/gemini-fullstack-langgraph-quickstart/tree/main
@mcp.tool()
def web_research(query: str) -> str:
    """Search the web for information, do some research """
    state = graph.invoke({"messages": [{"role": "user", "content": f"Today's date and time: {get_date_time()}\nUser question: {query}"}], "max_research_loops": 3, "initial_search_query_count": 3})
    return state['messages'][-1].content


@mcp.tool()
def get_date_time() -> str:
    """Get the current date and time"""
    return f"The current date and time is {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"

if __name__ == "__main__":
    mcp.run()
