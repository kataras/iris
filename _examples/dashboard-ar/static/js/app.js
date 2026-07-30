(function () {
    function updateStats() {
        fetch("/api/stats")
            .then(function (res) { return res.json(); })
            .then(function (data) {
                var el;

                el = document.getElementById("stat-requests");
                if (el) el.textContent = data.requests;

                el = document.getElementById("stat-uptime");
                if (el) el.textContent = data.uptime;

                el = document.getElementById("stat-memory");
                if (el) el.textContent = data.memoryMB + " MB";

                el = document.getElementById("stat-goroutines");
                if (el) el.textContent = data.numGoroutine;
            })
            .catch(function () {});
    }

    // Update every 5 seconds.
    setInterval(updateStats, 5000);
})();
