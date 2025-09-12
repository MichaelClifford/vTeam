# Ambient Agentic Runner Project

## Project Overview
This is an automated agentic system that uses Ambient Code AI CLI with Playwright MCP server integration to perform comprehensive tasks including website analysis, automation, and data processing.

## Key Components
- **Python Runner**: Orchestrates the agentic session and handles status updates
- **AI CLI**: Executes tasks with integrated MCP server capabilities  
- **Playwright MCP Server**: Provides headless browser tools via MCP protocol for navigation, screenshots, content extraction
- **Backend Integration**: Reports progress and results to Kubernetes agentic session API

## Agentic Capabilities
Ambient AI has access to advanced browser automation tools through the integrated Playwright MCP server:
- Navigate to websites and handle loading/timeouts
- Take screenshots for visual analysis 
- Extract text content and metadata
- Interact with forms and page elements
- Handle dynamic content and SPAs
- Multi-step browsing and analysis

## Instructions for Agentic Sessions
When executing agentic tasks:
1. **Always start with navigation** - go to the target website first
2. **Take a screenshot** - capture visual state for reference
3. **Extract comprehensive content** - get all text, links, metadata
4. **Be methodical** - explore different sections if relevant
5. **Provide detailed analysis** - comprehensive findings aligned with task objectives
6. **Include actionable insights** - practical recommendations based on findings

## Technical Notes
- Container runs AI CLI with integrated Playwright MCP server
- Chrome runs headless with optimized flags via MCP server
- Direct prompt passing eliminates temporary file creation
- Timeout handling: 5-minute default per agentic session
- Vision capabilities enabled for screenshot analysis through MCP
