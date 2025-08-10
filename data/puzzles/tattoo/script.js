document.addEventListener('DOMContentLoaded', function () {
	const links = document.querySelectorAll('a');
	links.forEach(function (link) {
		link.addEventListener('click', function (e) {
			e.preventDefault();
		});
	});
});
