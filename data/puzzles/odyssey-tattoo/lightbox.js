document.addEventListener('DOMContentLoaded', function () {
	const galleryLinks = document.querySelectorAll('.gallery-grid a');
	if (!galleryLinks.length) return;

	const images = Array.from(galleryLinks).map(link => ({
		full: link.getAttribute('href'),
		thumb: link.querySelector('img').getAttribute('src'),
		alt: link.querySelector('img').getAttribute('alt')
	}));

	galleryLinks.forEach((link, idx) => {
		link.addEventListener('click', function (e) {
			e.preventDefault();
			openLightbox(idx);
		});
	});

	function openLightbox(startIdx) {
		if (document.querySelector('.lightbox-overlay')) return; // Prevent multiple overlays
		let currentIdx = startIdx;
		const overlay = document.createElement('div');
		overlay.className = 'lightbox-overlay';
		overlay.innerHTML = `
			<button class="lightbox-close" title="Close">&times;</button>
			<div class="lightbox-image-container">
				<img class="lightbox-main-img" src="${images[startIdx].full}" alt="${images[startIdx].alt}">
			</div>
			<div class="lightbox-thumbnails">
				<button class="lightbox-arrow left" title="Previous" aria-label="Previous">&#8592;</button>
				${images.map((img, i) => `<img class="lightbox-thumb${i === startIdx ? ' selected' : ''}" src="${img.thumb}" data-idx="${i}" alt="${img.alt}">`).join('')}
				<button class="lightbox-arrow right" title="Next" aria-label="Next">&#8594;</button>
			</div>
		`;
		document.body.appendChild(overlay);

		const mainImg = overlay.querySelector('.lightbox-main-img');
		const thumbs = overlay.querySelectorAll('.lightbox-thumb');
		const closeBtn = overlay.querySelector('.lightbox-close');
		const leftArrow = overlay.querySelector('.lightbox-arrow.left');
		const rightArrow = overlay.querySelector('.lightbox-arrow.right');

		function showImage(idx) {
			currentIdx = idx;
			mainImg.src = images[idx].full;
			mainImg.alt = images[idx].alt;
			thumbs.forEach((t, i) => t.classList.toggle('selected', i === idx));
		}

		thumbs.forEach(thumb => {
			thumb.addEventListener('click', function () {
				const idx = parseInt(this.getAttribute('data-idx'));
				showImage(idx);
			});
		});

		leftArrow.addEventListener('click', function (e) {
			e.stopPropagation();
			showImage((currentIdx - 1 + images.length) % images.length);
		});
		rightArrow.addEventListener('click', function (e) {
			e.stopPropagation();
			showImage((currentIdx + 1) % images.length);
		});

		closeBtn.addEventListener('click', () => overlay.remove());
		overlay.addEventListener('click', e => {
			if (e.target === overlay) overlay.remove();
		});
		document.addEventListener('keydown', escListener);
		function escListener(e) {
			if (e.key === 'Escape') {
				overlay.remove();
				document.removeEventListener('keydown', escListener);
			} else if (e.key === 'ArrowLeft') {
				showImage((currentIdx - 1 + images.length) % images.length);
			} else if (e.key === 'ArrowRight') {
				showImage((currentIdx + 1) % images.length);
			}
		}
	}
});
