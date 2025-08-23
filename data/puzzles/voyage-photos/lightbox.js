document.addEventListener('DOMContentLoaded', function () {
	const grid = document.querySelector('.photo-grid');
	const links = Array.from(document.querySelectorAll('.photo-link'));
	let enlargedIdx = null;
	let enlargedElem = null;

	function getColumns() {
		const style = window.getComputedStyle(grid);
		const cols = style.getPropertyValue('grid-template-columns').split(' ').length;
		return cols;
	}

	function createArrowBtn(direction) {
		const btn = document.createElement('button');
		btn.className = `enlarged-control-btn ${direction}`;
		btn.innerHTML = direction === 'left'
			? '<svg viewBox="0 0 32 32"><path d="M20 8l-8 8 8 8"/></svg>'
			: '<svg viewBox="0 0 32 32"><path d="M12 8l8 8-8 8"/></svg>';
		return btn;
	}

	function createEnlargedElem(imgSrc, imgAlt, idx) {
		const enlargedDiv = document.createElement('div');
		enlargedDiv.className = 'enlarged-grid-item';
		enlargedDiv.innerHTML = `
			<div class="enlarged-view" style="position:relative;">
				<img class="enlarged-img" src="${imgSrc}" alt="${imgAlt}">
				<div class="enlarged-controls"></div>
			</div>
		`;

		const controls = enlargedDiv.querySelector('.enlarged-controls');
		if (idx > 0) {
			const leftBtn = createArrowBtn('left');
			leftBtn.addEventListener('click', function (e) {
				e.stopPropagation();
				enlarge(idx - 1);
			});
			controls.appendChild(leftBtn);
		}
		if (idx < links.length - 1) {
			const rightBtn = createArrowBtn('right');
			rightBtn.addEventListener('click', function (e) {
				e.stopPropagation();
				enlarge(idx + 1);
			});
			controls.appendChild(rightBtn);
		}
		return enlargedDiv;
	}

	function insertEnlarged(idx) {
		const cols = getColumns();
		const row = Math.floor(idx / cols);
		const insertIdx = Math.min((row + 1) * cols, links.length);
		const refNode = links[insertIdx] ? links[insertIdx] : null;

		const link = links[idx];
		const imgSrc = link.querySelector('img').src;
		const imgAlt = link.querySelector('img').alt + ' enlarged';

		if (!enlargedElem) {
			enlargedElem = createEnlargedElem(imgSrc, imgAlt, idx);
			grid.insertBefore(enlargedElem, refNode);
		} else {
			// Update image src/alt if needed
			const img = enlargedElem.querySelector('.enlarged-img');
			img.src = imgSrc;
			img.alt = imgAlt;
			// Remove old controls and add new ones
			const controls = enlargedElem.querySelector('.enlarged-controls');
			controls.innerHTML = '';
			if (idx > 0) {
				const leftBtn = createArrowBtn('left');
				leftBtn.addEventListener('click', function (e) {
					e.stopPropagation();
					enlarge(idx - 1);
				});
				controls.appendChild(leftBtn);
			}
			if (idx < links.length - 1) {
				const rightBtn = createArrowBtn('right');
				rightBtn.addEventListener('click', function (e) {
					e.stopPropagation();
					enlarge(idx + 1);
				});
				controls.appendChild(rightBtn);
			}
			// Move to new position
			grid.insertBefore(enlargedElem, refNode);
		}

		// Scroll to the large image
		setTimeout(() => {
			enlargedElem.scrollIntoView({ behavior: 'smooth', block: 'center' });
		}, 10);
	}

	function removeEnlarged() {
		if (enlargedIdx !== null) {
			links[enlargedIdx].classList.remove('enlarged-preview');
		}
		if (enlargedElem) {
			grid.removeChild(enlargedElem);
			enlargedElem = null;
		}
		enlargedIdx = null;
	}

	function enlarge(idx) {
		if (enlargedIdx !== null) {
			links[enlargedIdx].classList.remove('enlarged-preview');
		}
		if (enlargedIdx === null || enlargedIdx !== idx) {
			links[idx].classList.add('enlarged-preview');
			insertEnlarged(idx);
		}
		enlargedIdx = idx;
	}

	links.forEach((link, idx) => {
		link.addEventListener('click', function (e) {
			e.preventDefault();
			if (enlargedIdx === idx) {
				removeEnlarged();
			} else {
				enlarge(idx);
			}
		});
	});

	document.addEventListener('keydown', function (e) {
		if (enlargedIdx === null) return;
		if (e.key === 'Escape') {
			removeEnlarged();
		} else if (e.key === 'ArrowLeft') {
			if (enlargedIdx > 0) {
				enlarge(enlargedIdx - 1);
			}
		} else if (e.key === 'ArrowRight') {
			if (enlargedIdx < links.length - 1) {
				enlarge(enlargedIdx + 1);
			}
		}
	});
});
