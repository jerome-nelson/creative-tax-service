
const JiraAPI = {
    triggerPopup: (url) => {
        let params = `status=no,location=no,toolbar=no,menubar=no,
    width=600,height=800,popup=yes`;
        window.open(url, '_blank', params);
    },
    fetchIssues: async () => {
        try {
            const { email } = getSavedUser();
            const data = await fetch(`//localhost:5000/search?user=${email}`, { credentials: "include", method: 'POST' });
            const response = await data.json();

            if (!response.ok) {
                throw new Error("Fetch failed");
            }
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
            console.error('Error fetching user: ', e);
        }
    },
    refreshSession: async () => {
        try {
            const COOKIES = getCookies();
            const response = await fetch(`//localhost:5000/refresh`, { method: 'POST', headers: {'x-refresh': COOKIES['refresh_token']}});
            if (!response.ok) {
                throw new Error('Request failed');
            }
            const refreshCount = Number(localStorage.getItem('refresh_token')) ?? 0;
            localStorage.setItem('refresh_token', refreshCount + 1);

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
    const { name, email, picture } = await JiraAPI.fetchUser();
    window.document.title = `Hello ${name} | ` + window.document.title;
    localStorage.setItem(USER_KEY, JSON.stringify({ name, email, picture }));

    document.getElementById("logout").style.display = "block";
    document.getElementById("authed").innerHTML = `<img height="40" width="40" src="${picture}" alt="Avatar of ${name}"/> Welcome! ${name}`;
    document.getElementById("authed").style.display = "block";
    document.getElementById("login").style.display = "none";
}

function deleteCookie(name) {
    // DEV
    console.log("Deleting cookie: ", name);
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
}

function logout() {
    COOKIE_LIST.forEach(deleteCookie);
    localStorage.removeItem(USER_KEY);
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
    const refreshCount = localStorage.getItem('refresh_token');

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