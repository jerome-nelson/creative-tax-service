## Creative Tax Repo
A Go-Based Service that generates IP-Box/Creative Tax compliant entries, using Oauth and Google Gemini. 
Why? Because I was tired of manually doing this every month

### Requirements
* A JIRA Developer Application to be set up 
* These Oauth scopes:
  - `"offline_access", "read:me", "read:project.avatar:jira", "read:filter:jira", "read:group:jira", "read:issue:jira", "read:attachment:jira", "read:comment:jira", "read:comment.property:jira", "read:field:jira", "read:issue-details:jira", "read:field.default-value:jira", "read:field.option:jira", "read:field:jira", "read:group:jira"`
    (Note: `offline_access` is required for the `refresh_token` flow to be triggered)


### All Environment Variables Needed
These environment variables can be set as command line args or on a OS-level

```
SERVICE_NAME=lowercased-name-of-service

# Jira App Details
CLIENT_SECRET=<taken-from-developer-app>
CLIENT_ID=<taken-from-developer-app>
OAUTH_URL=https://auth.atlassian.com/oauth/token
REDIRECT_URL=<<taken-from-developer-app> 

# Server Details
PORT=<port to launch server on>
HOST=<hostname to use>

## LLM 
LLM_API_KEY=<developer-api-key>
```

### Approach
* Each service must be:
  i. Scale-able/non-blocking when operating
  ii. Fault-tolerant
  iii. Error hardened