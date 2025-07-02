const JIRA_URI = "//activecampaign.atlassian.net";
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
        console.log(parsedDescription);
        try {
            if (btn) {
                btn.classList.add('loading');
                btn.innerText = 'Loading';
                btn.setAttribute('disabled', true);
            }

            const response = await fetch(`//localhost:5000/transform`, {
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
        let params = `status=no,location=no,toolbar=no,menubar=no,width=600,height=800,popup=yes`;
        window.open(url, '_blank', params);
    },
    fetchIssues: async () => {
        try {
            //
            const data = getSavedUser();
            const COOKIES = getCookies();
            const auth = `${data.email}:${COOKIES.oauth_token}`;
            const jql = encodeURI(`assignee = currentUser() AND statusCategoryChangedDate >= \"2025-05-01\" AND statusCategoryChangedDate <= \"2025-05-30\" ORDER BY statusCategoryChangedDate DESC`);

            const response = await fetch(`//localhost:5000/temp`, {
                method: 'GET',
                credentials: 'include'
            })

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
            const response = await fetch(`//localhost:5000/refresh`, {
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
            monitorAuthTime();
            await setAuth();
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
    document.getElementById("authed").style.display = "block";
    document.getElementById("login").style.display = "none";

    JiraAPI.loadIssues()
}

function deleteCookie(name) {
    console.log("Deleting cookie: ", name);
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
}

function logout() {
    COOKIE_LIST.forEach(deleteCookie);
    localStorage.removeItem(USER_KEY);
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

function monitorAuthTime() {
    const COOKIES = getCookies();
    const expires = new Date(COOKIES.expiry);
    const refreshCount = localStorage.getItem('');

    document.getElementById('refresh-token').innerHTML = `${refreshCount}`;

    const checkingAuthTime = setInterval(async () => {
        const now = new Date();
        const diffMs = expires - now;
        let countDown = Math.floor(diffMs / 1000);
        const hrs = Math.floor(countDown / 3600);
        const mins = Math.floor((countDown % 3600) / 60);
        const secs = countDown % 60;
        document.getElementById("time-left").innerHTML = [
            hrs.toString().padStart(2, '0'),
            mins.toString().padStart(2, '0'),
            secs.toString().padStart(2, '0')
        ].join(':');

        if (document.getElementById("reauth-required").style.display === 'none') {
            document.getElementById("reauth-required").style.display = "block";
        }

        if (countDown <= 10) {
            clearInterval(checkingAuthTime);
            await JiraAPI.refreshSession();
        }
    }, 1000)

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

// Might be replaced by Datepicker functionality
function togglePicker() {
    const pickerElem = document.getElementById('date-picker-faux');
    if (pickerElem.style.visibility !== 'hidden') {
        pickerElem.style.visibility = 'hidden';
        return;
    }

    pickerElem.style.visibility = 'visible';
}