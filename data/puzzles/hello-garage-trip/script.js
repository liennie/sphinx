function changeFont(font) {
	article = document.querySelector("article");

	for (ctrl of document.querySelectorAll(".ctrl")) {
		if (ctrl.id == font) {
			ctrl.classList.add("pressed");
		} else {
			ctrl.classList.remove("pressed");
			article.classList.remove(ctrl.id)
		}
	}

	article.classList.add(font);
}
