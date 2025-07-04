function monitorAuthTime() {
    const COOKIES = getCookies();
    const expires = new Date(COOKIES.expiry);
    const refreshCount = localStorage.getItem('');

    document.getElementById('refresh-token').innerHTML = `${refreshCount ?? 0}`;

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

document.addEventListener('auth-loaded', async () => {
    console.log("Dev Panel loaded");
    monitorAuthTime();
});