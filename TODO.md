## Services/Features
- [ ] Hide Dev Details behind .env
- [ ] Split the servers
- [ ] Integrate k8s into build/docker-compose 
- [x] Setup HTTPS
  - [x] Combine both services under one DNS (must be docker-friendly)
      * How to do this without coupling both services?
        * `/api` - jira, `/` - pages
        * Allow services to communicate with each other internally
- [ ] Verify and set server timeouts for endpoints
- [ ] Improve AuthGuard to actually check if valid, against JIRA API
- [ ] Add a generic grant handler for both refresh and auth
- [x] Allow CORs options to be overridden/merged if needed
  - ALLOWED_ORIGINs + ALLOWED_HEADERS
- [ ] Move Origin CORs args into .env/.yaml or somekind of config
- [ ] Finish Transform Handler
  - Needs to limit text response to 20MB
  - Add Comments as well
- [ ] Allow selection of multiple issues

## Pages
- [ ] Finish styling HTML Auth Page
- [ ] Finish covering states 
      i. Loading Auth
      ii.     Auth failed (with follow-up actions)
      iii.    Auth succeeded (w/ a redirect)
  
## Go Language
- [ ] Examine external Go Libs for error handling standards
- [ ] Double-check usage of defer / understand it
- [ ] Discover possible errors and check if recovery needed (to keep application alive)
- [ ] How to load test server endpoints?
    * See if I can improve
    * Can it be automated to set benchmarks
- [ ] Review why ScriptUrl is being double escaped when it doesn't needed to be

## Extra Research
* Look into the Serve Multiplexer
    - Path matching
    - How do urls get sanitised
* Look into the Template lib
    - How urls and data gets sanitised before injecting
    - A more intuitive, straightforward way of injecting CSS, JS into the HTML files
* Does it make sense to use channels to handle API-triggered calls? (i.e Gemini context)
    - Should I centralise log functionality this way as well?

## JIRA
- [ ] How can I grab issues from JIRA in a programmatic way?
  - [ ] Create / use API for fetching issues via JQL
  - [ ] (AC Specific) - How do I allow the user to grab issues when there are CORs protections set up?
    * Confirm exactly what the issue is - (there is a specific cookie that is set that I cannot get when fetching)
      * Confirm if this can be solved by using a domain name instead
- [ ] Oauth Scopes - can I type them stronger? (look into the Jira Go lib and see how they do it)

## Nice to Have's / Do after/during production release
- [ ] Review garbage collection and performance
- [ ] Add monitoring (to cover both FE and BE) - Sentry/Grafana/NewRelic?
  - Set some Dev logging on FE/BE
- [ ] Add offline mode
  * User can save previous entries (saved as localstorage of some kind)