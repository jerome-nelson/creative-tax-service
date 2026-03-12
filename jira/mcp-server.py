#!/usr/bin/env python3
"""
MCP Server for Jira Issue Transformation
Provides a tool to transform Jira issues into tax statement format
"""

import json
import sys
from typing import Any

def read_style_guide():
    """Read the style guide template"""
    try:
        with open('/usr/src/app/jira/templates/style-guide.md', 'r') as f:
            return f.read()
    except FileNotFoundError:
        return "Transform the following Jira issue into a professional tax statement format."

def transform_issue(heading: str, description: list[str], task_name: str) -> dict[str, Any]:
    """
    Transform a Jira issue into tax statement format.
    This will be processed by Kiro's agent system.
    """
    style_guide = read_style_guide()
    
    # Format the description
    desc_text = "\n".join(description)
    
    # Create the prompt for Kiro
    prompt = f"""{style_guide}

Use the above style guide to transform the following input:

Heading: {heading}
Description: {desc_text}
Task Name: {task_name}

Please provide the output in JSON format with the following structure:
{{
    "heading": "transformed heading",
    "description": "transformed description",
    "links": ["relevant links"]
}}
"""
    
    return {
        "prompt": prompt,
        "heading": heading,
        "task_name": task_name
    }

def handle_list_tools():
    """List available MCP tools"""
    return {
        "tools": [
            {
                "name": "transform_jira_issue",
                "description": "Transform a Jira issue into a professional tax statement format using the style guide",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "heading": {
                            "type": "string",
                            "description": "The issue heading/summary"
                        },
                        "description": {
                            "type": "array",
                            "items": {"type": "string"},
                            "description": "The issue description as an array of strings"
                        },
                        "taskName": {
                            "type": "string",
                            "description": "The Jira task name/key"
                        }
                    },
                    "required": ["heading", "description", "taskName"]
                }
            }
        ]
    }

def handle_call_tool(tool_name: str, arguments: dict[str, Any]):
    """Handle tool execution"""
    if tool_name == "transform_jira_issue":
        result = transform_issue(
            arguments["heading"],
            arguments["description"],
            arguments["taskName"]
        )
        return {
            "content": [
                {
                    "type": "text",
                    "text": json.dumps(result, indent=2)
                }
            ]
        }
    else:
        return {
            "error": {
                "code": -32601,
                "message": f"Unknown tool: {tool_name}"
            }
        }

def main():
    """Main MCP server loop"""
    for line in sys.stdin:
        try:
            request = json.loads(line)
            method = request.get("method")
            params = request.get("params", {})
            request_id = request.get("id")
            
            if method == "tools/list":
                response = handle_list_tools()
            elif method == "tools/call":
                tool_name = params.get("name")
                arguments = params.get("arguments", {})
                response = handle_call_tool(tool_name, arguments)
            else:
                response = {
                    "error": {
                        "code": -32601,
                        "message": f"Unknown method: {method}"
                    }
                }
            
            # Send response
            result = {
                "jsonrpc": "2.0",
                "id": request_id,
                "result": response
            }
            print(json.dumps(result), flush=True)
            
        except Exception as e:
            error_response = {
                "jsonrpc": "2.0",
                "id": request.get("id") if 'request' in locals() else None,
                "error": {
                    "code": -32603,
                    "message": str(e)
                }
            }
            print(json.dumps(error_response), flush=True)

if __name__ == "__main__":
    main()
