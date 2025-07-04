const JIRA_URI = "activecampaign.atlassian.net";
const IFRAME_PARAMS = `status=no,location=no,toolbar=no,menubar=no,width=600,height=800,popup=yes`;
const shortMonths = ['Jan', 'Feb', 'Mar', 'Apr', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const REFRESH_COUNT_KEY = 'refresh_token';
const transformAPI = {
    generateEntry: async (event, taskName, heading, description) => {
        const btn = event.target;
        const parsedDescription = [...description.childNodes.values().filter(node => {
            if (node.nodeName.toLowerCase() === "ul") {
                return true;
            }
        }).map(node => node.innerText)];
        try {
            if (btn) {
                btn.classList.add('loading');
                btn.innerText = 'Loading';
                btn.setAttribute('disabled', true);
            }

            const response = await fetch(`/api/transform`, {
                method: "POST",
                credentials: 'include',
                body: JSON.stringify({
                    taskName,
                    heading,
                    description: parsedDescription
                })
            });
            if (!response.ok) {
                throw new Error("Fetch failed");
            }
            if (btn) {
                btn.classList.remove('loading');
                btn.innerText = 'Regenerate Tax Entry';
                btn.removeAttribute('disabled');
            }

            const result = await response.json();
            document.getElementById(`${taskName}-result`).innerHTML = `<hr /><div>${result.heading}</div><div>${result.description}</div><ul><li>${result.links}</li></ul>`
        } catch (e) {
            if (btn) {
                btn.classList.remove('loading');
                btn.classList.add('failed');
                btn.innerText = 'Try Again? (Generation Failed)';
                btn.removeAttribute('disabled');
            }
            console.error(e);
        }
    }
}

const JiraAPI = {
    loadIssues: async () => {
        try {
            const data = await JiraAPI.fetchIssues();
            document.getElementById('issues').style.display = 'block';
            JiraAPI.setIssueList(data.issues);
        } catch (e) {
            console.error("Fetch err: ", e)
        }
    },
    setIssueList: (issues) => {
        const list = document.createElement('ul');
        list.id = 'issues-list';
        list.setAttribute('class', 'issues-list');
        for (const issue of issues) {
            const {key, renderedFields: { description }, fields: {summary, created, updated, issuetype }} = issue;
            localStorage.setItem(`issue-${key}`, description);
            const listItem = document.createElement('li');
            listItem.setAttribute('class', 'issue-type');

            const button = document.createElement('button');
            button.setAttribute('class', 'generate-issue cta small');
            button.innerText = 'Generate Tax Entry';

            listItem.innerHTML = `
                    <section class="issue-wrapper">
                        <aside class="issue-icon">
<!--                            <img src="${issuetype.iconUrl}" title="${issuetype.name} icon" alt="${issuetype.name} icon" />-->
<!-- Temporary -->
                                <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns" width="16px" height="16px" viewBox="0 0 16 16" version="1.1" data-ember-extension="1">
                                    <!-- Generator: Sketch 3.5.2 (25235) - http://www.bohemiancoding.com/sketch -->
                                    <title>bug</title>
                                    <desc>Created with Sketch.</desc>
                                    <defs/>
                                    <g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd" sketch:type="MSPage">
                                        <g id="bug" sketch:type="MSArtboardGroup">
                                            <g id="Bug" sketch:type="MSLayerGroup" transform="translate(1.000000, 1.000000)">
                                                <rect id="Rectangle-36" fill="#E5493A" sketch:type="MSShapeGroup" x="0" y="0" width="14" height="14" rx="2"/>
                                                <path d="M10,7 C10,8.657 8.657,10 7,10 C5.343,10 4,8.657 4,7 C4,5.343 5.343,4 7,4 C8.657,4 10,5.343 10,7" id="Fill-2" fill="#FFFFFF" sketch:type="MSShapeGroup"/>
                                            </g>
                                        </g>
                                    </g>
                                </svg>
                        </aside>
                        <article class="issue-details" id="${key}-details">
                            <h4 class="title">${key} - ${summary}</h4>
                            <div id="${key}-description" class="task-description"></div>
                            <aside class="sub-issue">Last Updated on ${addFormattedTime(updated)}</aside>
                            <aside class="button-group"></aside>
                            <div id="${key}-result"></div>
                        </article>
                    </section>
                `;
            listItem.querySelector(`#${key}-description`).innerHTML = description;
            button.addEventListener('click', event => transformAPI.generateEntry(event, key, summary, listItem.querySelector(`#${key}-description`)));

            listItem.querySelector(`#${key}-details .button-group`).appendChild(button);
            list.appendChild(listItem);
        }

        document.getElementById('issue-container').append(list);
    },
    triggerPopup: (url) => {
        window.open(url, '_blank', IFRAME_PARAMS);
    },
    fetchIssues: async () => {
        try {
            const COOKIES = getCookies();
            const user = getSavedUser();
            const apiKey =  sessionStorage.getItem('apiKey');
            let auth = `Bearer ${COOKIES.oauth_token}`;
            let token = '';
            if (user.email && apiKey) {
                token = btoa(`${user.email}:${apiKey}`);
                auth = "Basic " + token;
            }

            const jql = encodeURI(`assignee = currentUser() AND statusCategoryChangedDate >= \"2025-05-01\" AND statusCategoryChangedDate <= \"2025-05-30\" ORDER BY statusCategoryChangedDate DESC`);
            const response = await fetch(`/cors/${JIRA_URI}/rest/api/3/search/jql?expand=renderedFields&fields=issuetype,summary,description,created,updated&jql=${jql}`, {
                method: 'GET',
                headers: {
                    Accept: 'application/json',
                    Authorization: auth,
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            if (!response.ok) {
                throw new Error("Fetch failed");
            }

            return await response.json();
        } catch (e) {
            console.error(e);
        }
    },
    fetchUser: async () => {
        try {
            const COOKIES = getCookies();
            const response = await fetch(`https://api.atlassian.com/me`, {
                headers: {
                    Authorization: `Bearer ${COOKIES.oauth_token}`
                }
            });

            return await response.json();
        } catch (e) {
            // ERROR LOG
            console.error('Error fetching user, attempting to fetch previously cached user ', e);
            return getSavedUser();
        }
    },
    refreshSession: async () => {
        try {
            const COOKIES = getCookies();
            const response = await fetch(`/api/refresh`, {
                method: 'POST',
                headers: {'x-refresh': COOKIES['refresh_token']},
                credentials: 'include'
            });
            if (!response.ok) {
                throw new Error('Request failed');
            }
            const refreshCount = Number(localStorage.getItem('refresh_token')) ?? 0;
            localStorage.setItem(REFRESH_COUNT_KEY, refreshCount + 1);

            window.location.reload();
        } catch (e) {
            // ERROR LOG
            console.error('Error fetching refresh: ', e);
            window.location.reload();
        }
    },
    startAuthFlow: async () => {
        const COOKIES = getCookies();
        if (!!COOKIES?.oauth_token) {
            await setAuth();
            document.dispatchEvent(new CustomEvent('auth-loaded'));
        }
    }
}

const COOKIE_LIST = ['oauth_token', 'scopes', 'expiry', 'refresh_token'];
const USER_KEY = 'user';

async function setAuth() {
    const {name, email, picture} = await JiraAPI.fetchUser();
    window.document.title = `Hello ${name}` + window.document.title.replace('Log in', ' ');
    localStorage.setItem(USER_KEY, JSON.stringify({name, email, picture}));
    const avatar = document.createElement('div');
    avatar.setAttribute('class', 'avatar');
    avatar.innerHTML = `<img height="40" width="40" src="${picture}" alt="Avatar of ${name}"/>`;
    document.getElementById("logout").style.display = "block";
    document.getElementById("user").appendChild(avatar);
    document.getElementById("user").append(` Welcome! ${name}`);
    document.getElementById('user-details').style.display = 'flex';
    document.getElementById('api-token-panel').style.display = 'block';
    document.getElementById("authed").style.display = "block";
    document.getElementById("login").style.display = "none";

    if (sessionStorage.getItem('apiKey')) {
        document.getElementById('api-panel').style.display = 'none';
        document.getElementById('remove-token').style.display = 'block';
    } else {
        document.getElementById("api-panel").style.display = 'block';
        document.getElementById('remove-token').style.display = 'none';
    }
    document.getElementById('remove-api-token').addEventListener('click', event => {
        event.preventDefault();
        sessionStorage.removeItem('apiKey');
        document.getElementById('api-panel').style.display = 'block';
        document.getElementById('remove-token').style.display = 'none';
    });
    document.getElementById('api-panel').addEventListener('submit', event => {
        event.preventDefault();
        const apiKey = event.target.querySelector('input[name="add-api-token"]').value;
        sessionStorage.setItem('apiKey', apiKey);
        document.getElementById('api-panel').style.display = 'none';
        document.getElementById('remove-token').style.display = 'block';
    });
    await JiraAPI.loadIssues()
}

function deleteCookie(name) {
    console.log("Deleting cookie: ", name);
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
}

function logout() {
    COOKIE_LIST.forEach(deleteCookie);
    localStorage.removeItem(USER_KEY);

    sessionStorage.removeItem('apiKey');
    document.getElementById("api-panel").style.display = 'block';
    document.getElementById('remove-token').style.display = 'none';

    localStorage.removeItem(REFRESH_COUNT_KEY);
    window.location.reload();
}


function getCookies() {
    const cookies = document.cookie.split(';');
    return cookies.reduce((acc, cookieStr) => {
        const [key, val] = cookieStr.split('=');
        const findKey = key.toLowerCase().trim();
        if (COOKIE_LIST.includes(findKey)) {
            return {
                ...acc,
                [findKey]: val
            }
        }
        return acc;
    }, {});
}


function getSavedUser() {

    const data = localStorage.getItem(USER_KEY);

    if (!!data) {
        return JSON.parse(localStorage.getItem(USER_KEY));
    }

    return {};
}

function addFormattedTime(timestamp) {
    const date = new Date(timestamp);
    return `${date.getDay()} ${shortMonths[date.getMonth() - 1]} ${date.getFullYear()}`;
}