import {McpServer} from "@modelcontextprotocol/sdk/server/mcp.js";
import {z} from "zod";
import {sampleText, text} from "./utils.ts";
import {StdioServerTransport} from "@modelcontextprotocol/sdk/server/stdio.js";

const server = new McpServer({
    name: "Research Bot",
    description: "A bot that helps you find research papers.",
    version: "1.0.0",
});

type Queries = {
    searches: {
        reason: string;
        query: string;
    }[];
}

type ReflectionResults = {
    is_sufficient: boolean;
    knowledge_gap: string;
    follow_up_query: string;
}

const currentDateTime = new Date().toISOString();

server.tool("research", {prompt: z.string()}, async ({prompt}, ctx) => {
    const queries = await sampleText(ctx, "planner", `Query: ${prompt}`);
    const parsedQueries = JSON.parse(queries) as Queries;

    const max_loop = 5;
    const searches = [] as string[];

    for (const search of parsedQueries.searches) { // parallelize the web search & reflection cycle
        let loop = 0;
        let is_sufficient = false;
        let this_search_results = [] as string[];
        let this_search_query = search.query;

        while (loop < max_loop && !is_sufficient) { // the web search & reflection cycle
            const searchResult = await sampleText(ctx, "searchAgent", `Search term: ${this_search_query}\nReason for searching: ${search.reason}`);
            this_search_results.push(searchResult);
            const reflection_results = await sampleText(ctx, "reflection", `Original user query: ${prompt}\nSearch results: ${JSON.stringify(this_search_results)}\n`);
            const parsedReflectionResults = JSON.parse(reflection_results) as ReflectionResults;
            is_sufficient = parsedReflectionResults.is_sufficient;
            if (is_sufficient) { // condition for either move forward to final report or continue the web search & reflection cycle
                break;
            } else {
                this_search_query = parsedReflectionResults.follow_up_query;
            }
            loop++;
        }
        searches.push(...this_search_results);
    }
    

    const paper = await sampleText(ctx, "writer",
        `Current date and time: ${currentDateTime}\nOriginal query: ${prompt}\nSummarized search results: ${JSON.stringify(searches)}\n\n`)

    return text(paper);
})

const transport = new StdioServerTransport()
await server.connect(transport)
