![Nanobot Logo](./docs/header.svg)

# Nanobot: Build Agents with MCP

## Status: I just wrote this last week, might work, might not ¯\\_(ツ)_/¯

Nanobot is a framework that combines MCP clients and servers to create full AI agents using
only MCP. It is designed to be small, simple, non-intrusive, and leverage the pure awesomeness
of MCP.

```yaml
# Example nanobot.yaml

publish:
  entrypoint: fs-agent

agents:
  fs-agent:
    model: gpt-4o
    instructions: You're awesome at file systems things, help the user with files... and things.
    tools: ["filesystem"]

mcpServers:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "${ROOT_CWD}"]
```

```shell
nanobot run nanobot.yaml

> List my current files
```

## Installation

Install with brew (macOS/Linux):

```shell
brew install nanobot-ai/tap/nanobot
```

Install with curl (macOS/Linux):
```shell
curl -sSfL https://get.nanobot.ai | sh -
```

For Windows there are binaries in the [releases](https://github.com/nanobot-ai/nanobot/releases) but they aren't really tested.


## Running

export openai key by
```
 export OPENAI_API_KEY=xxx
```

### Chat

Doing `nanobot run [FILE|DIRECTORY]` will start an interactive chat session with the agent pointed to
by the `publish.entrypoint` in the `nanobot.yaml`.

### Scripting

You can run `nanobot run [FILE|DIRECTORY] [PROMPT]` to run a single prompt against the agent. The output can be
saved to a file using `-o [OUTPUT_FILE]`

### MCP Server

You agent can be ran as a MCP Server by using the `--mcp` flag. This will start a MCP server.
```shell
# Start a HTTP Streaming server
nanobot run nanobot.yaml --mcp --listen-address localhost:8099
```

## Examples

Refer to the [examples](./examples) directory for more examples of using Nanobot. The examples are
1. [Hello World](./examples/hello-world)
1. [Research Agent](examples/research-bot)
1. [OpenAI Computer Use Agent](examples/openai-computer-using-agent)

## Configuration Reference

```yaml
publish:
  # The main agent to interactive with when using `nanobot run`. This can also be a tool and
  # should be in the format [MCP_SERVER]/[TOOL_NAME], for example `filesystem/ls`. If your
  # are using a tool, `nanobot run` will assume it accepts a string argument named "prompt"
  entrypoint: fs-agent
  
  # Tools to publish when ran as a MCP server. This can refer to tools or agents.
  # If you refer to a MCP Server it will publish all tools from that server.
  tools: [server1/tool2, agent2, server3]
  
  # Prompt to publish when ran as a MCP server. This can be all prompts
  # from a server or a specific prompt.
  prompts: [server2, server1/prompt1]
  
  # Resources to publish when ran as a MCP server. This can be all
  # resources from a server or a specific resource.
  resources: [server1/resource1, server2]
  
  # Resource templates to publish when ran as a MCP server. This can be all
  # resource templates from a server or a specific resource template.
  resourceTemplates: ["server1/resource-template1", server2]
  
  # The name of the MCP Server when running as a MCP server.
  name: My Agent
  
  # The description of the MCP Server when running as a MCP server.
  description: This is a description of my agent
  
  # The version of the MCP Server when running as a MCP server.
  version: 0.0.1
  
  # The instructions for the MCP Server when running as a MCP server.
  instructions: Use wisely and be nice
  
  # The introduction message show in the console run running `nanobot run`
  # This field can also refer to a prompt to create a dynamic messages.
  #   introduction:
  #     prompt: prompt1
  #     mcpServer: [arg1, arg2]
  #     args:
  #       field1: value1
  introduction: I'm a fancy agent that can do things. Nice to mean you
  
agents:
  myAgent:
    # The model name. Currently supported is OpenAI Responses API and
    # Anthropic API. The model string is passed as is to the API.
    model: gpt-4o
    
    # The instructions for the agent. This can be a static string or a dynamic
    # prompt.
    #   instructions:
    #     prompt: prompt1
    #     mcpServer: server1
    #     args:
    #       field1: value1
    instructions: You're awesome at things, help the user with things.
    
    # The tools to use for the agent. This can be a reference to a MCP Server,
    # a specific tool, or an agent.
    tools: [filesystem, server1/tool2, agent2]
    
    # Inform the model on how to choose a tool. This can be "auto", "none", or a specific
    # tool name. If set to "auto", the model will choose the best tool for the task.
    toolChoice: "auto"
    
    toolExtensions:
      # The tool name is the key and the value is a map of options to pass as in the tool API
      # field. Refer to the openai-computer-using-agent example for an example of using this.
      # This is a mechanism to support custom server side tools that are offered in the OpenAI
      # and Anthropic APIs.
      tool1:
        option1: value1
        option2: value2
        # This is a special option that will remove fields that are normal sent in the tool object
        # in the API.
        remove: [description]

    # Whether chat history is saved for this agent in the MCP session. If set to true,
    # which is the default, each invocation of the agent will be added to the same chat
    # thread. If set to false, each invocation will be a new chat thread.
    chatHistory: true
    
    # The temperature to use for the agent. This is a number between 0 and 1. The default
    # is dependent on the model API.
    temperature: 0.7
    
    # The top_p to use for the agent. This is a number between 0 and 1. The default
    # is dependent on the model API. It is always recommend to use top_p or temperature
    # but not both.
    topP: 0.9
    
    # The output schema to use for the agent. This is a map of field names to types.
    output:
      # Whether or not the output must strictly conform. This is only supported in OpenAI API
      strict: true
      schema: # Standard JSON Schema
        type: object
        properties: {}
    
    # Whether the chat history should automatically be truncated to match the context size.
    # Can be set to "auto" or "disabled". "disabled" is the default
    truncation: "auto"
    
    # The maximum number of output tokens allowed by default. This can be overridden by the
    # `maxTokens` field in the CreateMessage sampling request.
    maxToken: 1024
    
    # Aliases used to match model hint name in CreateMessage ModelPreferences. The name of the
    # agent is always used as the first alias.
    alias: ["model1", "model2"]
    
    # The cost of the agent used for the priority evaluation in the CreateMessage ModelPreferences.
    cost: 0.5
    
    # The speed of the agent used for the priority evaluation in the CreateMessage ModelPreferences.
    speed: 0.5
    
    # The intelligence of the agent used for the priority evaluation in the CreateMessage ModelPreferences.
    intelligence: 0.5
```

# LICENSE

Apache 2.0 License
