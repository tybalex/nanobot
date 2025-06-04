# server.py
from mcp.server.fastmcp import FastMCP
from agent.graph import graph

# Create an MCP server
mcp = FastMCP("Demo")


# Add an addition tool
@mcp.tool()
def add(a: int, b: int) -> int:
    """Add two numbers"""
    return a + b


@mcp.tool()
def web_research(query: str) -> str:
    """Search the web for information"""
    state = graph.invoke({"messages": [{"role": "user", "content": query}], "max_research_loops": 3, "initial_search_query_count": 3})
    # content = markdownify(state["messages"][-1].content)
    return state['messages'][-1].content


if __name__ == "__main__":
    mcp.run()
