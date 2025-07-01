## Creative Tax Repo


### Env Setup

```
SERVICE_NAME=lowercased-name-of-service

# Jira App Details
CLIENT_SECRET=
CLIENT_ID=
OAUTH_URL=https://auth.atlassian.com/oauth/token
REDIRECT_URL=

# Server Details
PORT=
HOST=

```
* How to handle errors consistently, gracefully
* Look into the Serve Multiplexer
    - Path matching
    - How do urls get sanitised
* Look into the Template lib
  - How urls and data gets sanitised before injecting
  - A more intuitive, straightforward way of injecting CSS,JS into the HTML files
* Finish the splitting of the servers
* Needs Okta integration for AC usage of JIRA api specifically :-/ 
  * How to enable this but also to allow general use?
  * For now use CSV?
* Add Grafana to it? (Or NewRelic) (Maybe)
* Add offline mode (nice to have)