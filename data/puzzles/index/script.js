(() => {
	dtl = document.querySelector('#days .tens');
	dol = document.querySelector('#days .ones');
	htl = document.querySelector('#hours .tens');
	hol = document.querySelector('#hours .ones');
	mtl = document.querySelector('#minutes .tens');
	mol = document.querySelector('#minutes .ones');
	ttl = document.querySelector('#seconds .tens');
	tol = document.querySelector('#seconds .ones');

	const updateTime = () => {
		const now = new Date();
		const end = new Date('2025-09-20T12:00:00+02:00');
		let diff = end - now;

		if (diff <= 0) {
			diff = 0;
		}

		diff /= 1000;

		const seconds = Math.floor(diff % 60);
		diff /= 60;

		const minutes = Math.floor(diff % 60);
		diff /= 60;

		const hours = Math.floor(diff % 24);
		diff /= 24;

		const days = Math.floor(diff % 99);

		dtl.setAttribute("data-digit", Math.floor(days / 10));
		dol.setAttribute("data-digit", days % 10);
		htl.setAttribute("data-digit", Math.floor(hours / 10));
		hol.setAttribute("data-digit", hours % 10);
		mtl.setAttribute("data-digit", Math.floor(minutes / 10));
		mol.setAttribute("data-digit", minutes % 10);
		ttl.setAttribute("data-digit", Math.floor(seconds / 10));
		tol.setAttribute("data-digit", seconds % 10);

		document.title = `${days.toString().padStart(2, "0")} : ${hours.toString().padStart(2, "0")} : ${minutes.toString().padStart(2, "0")} : ${seconds.toString().padStart(2, "0")}`;
	};
	setInterval(updateTime, 1000);
	updateTime();
})();
