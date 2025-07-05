const JIRA_URI = "activecampaign.atlassian.net";
const IFRAME_PARAMS = `status=no,location=no,toolbar=no,menubar=no,width=600,height=800,popup=yes`;
const shortMonths = ['Jan', 'Feb', 'Mar', 'Apr', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const longMonths = [
    "January", "February", "March", "April",
    "May", "June", "July", "August",
    "September", "October", "November", "December"
];
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
    formatDate: (date) => {
        const yyyy = date.getFullYear();
        const mm = String(date.getMonth() + 1).padStart(2, '0');
        const dd = String(date.getDate()).padStart(2, '0');
        return `${yyyy}-${mm}-${dd}`;
    },
    loadIssues: async (startDate, endDate) => {
        try {
            let start = startDate;
            let end = endDate;

            if (!start || !end) {
                const now = new Date();
                let year = now.getFullYear();
                let monthIndex = now.getMonth();
                const defaultStart = new Date(year, monthIndex, 1);
                const lastDay = new Date(year, monthIndex + 1, 0).getDate();
                const defaultEnd = new Date(year, monthIndex, lastDay);
                start = JiraAPI.formatDate(defaultStart);
                end = JiraAPI.formatDate(defaultEnd);
            }

            const data = await JiraAPI.fetchIssues(start, end);

            // Switch statement
            if (data.issues && data.issues.length > 0) {
                JiraAPI.setIssueList(data.issues);
            } else if (data.issues.length === 0) {
                // TODO: Create a set list function
                const list = document.createElement('ul');
                list.id = 'issues-list';
                list.setAttribute('class', 'issues-list');
                document.getElementById('issue-container').textContent = 'No issues found - try a different month';
            } else {
                const list = document.createElement('ul');
                list.id = 'issues-list';
                list.setAttribute('class', 'issues-list');
                document.getElementById('issue-container').textContent = 'No issues found for this month - try a different one';
            }
        } catch (e) {
            console.error("Fetch err: ", e);
            const list = document.createElement('ul');
            list.id = 'issues-list';
            list.setAttribute('class', 'issues-list');
            document.getElementById('issue-container').textContent = 'Unable to fetch issues';
            return false;
        }
    },
    setIssueList: (issues) => {
        const list = document.createElement('ul');
        list.id = 'issues-list';
        list.setAttribute('class', 'issues-list');
        for (const issue of issues) {
            const {key, renderedFields: {description}, fields: {summary, updated, issuetype}} = issue;
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
                                <svg xmlns="http://www.w3.org/2000/svg" width="16px" height="16px" viewBox="0 0 16 16" data-ember-extension="1">
                                    <title>bug</title>
                                    <desc>Created with Sketch.</desc>
                                    <defs/>
                                    <g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
                                        <g id="bug">
                                            <g id="Bug" transform="translate(1.000000, 1.000000)">
                                                <rect id="Rectangle-36" fill="#E5493A" x="0" y="0" width="14" height="14" rx="2"/>
                                                <path d="M10,7 C10,8.657 8.657,10 7,10 C5.343,10 4,8.657 4,7 C4,5.343 5.343,4 7,4 C8.657,4 10,5.343 10,7" id="Fill-2" fill="#FFFFFF" />
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

        document.getElementById('issue-container').innerHTML = '';
        document.getElementById('issue-container').append(list);
    },
    triggerPopup: (url) => {
        window.open(url, '_blank', IFRAME_PARAMS);
    },
    fetchIssues: async (start, end) => {
        try {
            const COOKIES = getCookies();
            const user = getSavedUser();
            const apiKey = sessionStorage.getItem('apiKey');
            let auth = `Bearer ${COOKIES.oauth_token}`;
            let token = '';
            if (user.email && apiKey) {
                token = btoa(`${user.email}:${apiKey}`);
                auth = "Basic " + token;
            }

            const jql = encodeURI(`assignee = currentUser() AND statusCategoryChangedDate >= \"${start}\" AND statusCategoryChangedDate <= \"${end}\" ORDER BY statusCategoryChangedDate DESC`);
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
    document.getElementById("auth-container").style.display = "block";
    document.getElementById("authed").style.display = "block";
    document.getElementById("login").style.display = "none";
    document.getElementById('issues').style.display = 'block';

    const avatar = document.createElement('div');
    avatar.setAttribute('class', 'avatar');
    avatar.innerHTML = `<img height="40" width="40" src="${picture}" alt="Avatar of ${name}" title="Welcome! ${name}"/>`;
    document.getElementById("logout").style.display = "block";
    document.getElementById("user").appendChild(avatar);
    document.getElementById('user-details').style.display = 'flex';
    document.getElementById('api-token-panel').style.display = 'block';


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
    document.getElementById('api-panel').addEventListener('submit', async event => {
        event.preventDefault();
        const apiKey = event.target.querySelector('input[name="add-api-token"]').value;
        sessionStorage.setItem('apiKey', apiKey);
        await JiraAPI.loadIssues();
        document.getElementById('api-panel').style.display = 'none';
        document.getElementById('remove-token').style.display = 'block';
    });

    await JiraAPI.loadIssues();
    loadMonthPicker();
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

function loadMonthPicker() {
    const container = document.getElementById('month-picker');

    if (!container) {
        console.warn("Missing #month-picker element");
        return;
    }

    container.innerHTML = '';
    container.style.position = 'relative';

    const months = [
        "January", "February", "March", "April",
        "May", "June", "July", "August",
        "September", "October", "November", "December"
    ];

    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonthIndex = now.getMonth();
    let selectedYear = currentYear;

    // Create CTA
    const labelDiv = document.createElement('div');
    labelDiv.className = 'cta-inverse';
    labelDiv.textContent = `${months[currentMonthIndex]} ${currentYear} (press to select a prev. month)`;
    container.appendChild(labelDiv);

    // Create wrapper for ul + year navigation
    const navWrapper = document.createElement('div');
    navWrapper.className = 'month-nav-wrapper';
    navWrapper.style.position = 'absolute';
    navWrapper.style.top = '100%';
    navWrapper.style.left = '0';
    navWrapper.style.width = '100%';
    navWrapper.style.background = 'white';
    navWrapper.style.border = '1px solid #ccc';
    navWrapper.style.boxShadow = '0 4px 8px rgba(0,0,0,0.1)';
    navWrapper.style.display = 'none';
    navWrapper.style.zIndex = '1000';

    // Create header row with arrows and year
    const header = document.createElement('div');
    header.className = 'month-nav-header';
    header.style.display = 'flex';
    header.style.justifyContent = 'space-between';
    header.style.alignItems = 'center';
    header.style.padding = '8px 12px';
    header.style.borderBottom = '1px solid #eee';

    const leftArrow = document.createElement('span');
    leftArrow.textContent = '←';
    leftArrow.style.cursor = 'pointer';

    const rightArrow = document.createElement('span');
    rightArrow.textContent = '→';
    rightArrow.style.cursor = 'pointer';

    const yearLabel = document.createElement('span');
    yearLabel.textContent = selectedYear;
    yearLabel.style.fontWeight = 'bold';

    header.appendChild(leftArrow);
    header.appendChild(yearLabel);
    header.appendChild(rightArrow);
    navWrapper.appendChild(header);

    // Create UL
    const ul = document.createElement('ul');
    ul.className = 'month-list';
    ul.style.listStyle = 'none';
    ul.style.margin = '0';
    ul.style.padding = '0';

    navWrapper.appendChild(ul);
    container.appendChild(navWrapper);

    // Populate the months
    function renderMonths() {
        ul.innerHTML = ''; // Clear
        yearLabel.textContent = `${selectedYear}`;

        months.forEach((month, index) => {
            const li = document.createElement('li');
            li.textContent = `${month} ${selectedYear}`;
            li.style.padding = '8px 12px';
            li.style.cursor = 'pointer';

            if (selectedYear >= currentYear) {
                rightArrow.style.opacity = '0.3';
                rightArrow.style.pointerEvents = 'none';
            } else {
                rightArrow.style.opacity = '1';
                rightArrow.style.pointerEvents = 'auto';
            }

            // Disable future months if in current year
            if (selectedYear === currentYear && index > currentMonthIndex) {
                li.classList.add('disabled');
                li.style.color = '#999';
                li.style.pointerEvents = 'none';
                li.style.background = '#f9f9f9';
            }

            li.addEventListener('click', () => {
                labelDiv.textContent = `${month} ${selectedYear} (press to select a prev. month)`;
                navWrapper.style.display = 'none';
            });

            ul.appendChild(li);
        });
    }

    // Initial render
    renderMonths();

    // Toggle open/close
    labelDiv.addEventListener('click', (e) => {
        e.stopPropagation();
        navWrapper.style.display = (navWrapper.style.display === 'none') ? 'block' : 'none';
    });

    // Outside click hides
    document.addEventListener('click', (e) => {
        if (!container.contains(e.target)) {
            navWrapper.style.display = 'none';
        }
    });

    ul.addEventListener('click', async (e) => {
        const clicked = e.target;

        // Check if clicked element is an li and not disabled
        if (clicked.tagName === 'LI' && !clicked.classList.contains('disabled')) {
            // Extract month and year from li text
            // li.textContent looks like: "January 2024"
            const [monthName, yearStr] = clicked.textContent.split(' ');
            const monthIndex = months.indexOf(monthName);
            const year = parseInt(yearStr, 10);

            // Build start/end dates
            const startDate = new Date(year, monthIndex, 1);
            const lastDay = new Date(year, monthIndex + 1, 0).getDate();
            const endDate = new Date(year, monthIndex, lastDay);

            const startStr = JiraAPI.formatDate(startDate);
            const endStr = JiraAPI.formatDate(endDate);

            // Update CTA label
            labelDiv.textContent = `${monthName} ${yearStr} (press to select a prev. month)`;

            // Hide the month list
            navWrapper.style.display = 'none';
        try {
            container.classList.add('disabled');
            await JiraAPI.loadIssues(startStr, endStr)
        } finally {
            container.classList.remove('disabled');
        }

            // Update label
            labelDiv.textContent = `${monthName} ${year} (press to select a prev. month)`;
            navWrapper.style.display = 'none';
        }
    });

    // Arrow navigation
    leftArrow.addEventListener('click', () => {
        selectedYear--;
        renderMonths();
    });

    rightArrow.addEventListener('click', () => {
        selectedYear++;
        renderMonths();
    });
}
