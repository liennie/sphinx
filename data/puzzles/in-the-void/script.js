function lerp(a, b, t) {
	return a + (b - a) * t;
}

document.addEventListener('DOMContentLoaded', function () {
	// --- Floating symbols and keys ---
	const background = document.getElementById('background');
	const symbols = [];
	const symbolCount = 128;
	const keyCount = 2;
	const toCenterDuration = 3;
	const minSpeed = 1;
	const maxSpeed = 4;

	const keySVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48"><g stroke="white" stroke-width="4" stroke-linecap="round" fill="transparent"><circle cx="14" cy="24" r="7" fill="transparent"/><path d="M21 24 l18 0"/><path d="M35 24 l0 8"/><path d="M39 24 l0 4"/><path d="M25 24 l0 6"/></g><g stroke="white" stroke-width="3" fill="transparent"><path d="M21 24 Q24 20 27 24"/></g></svg>';

	// Place symbols with random directions and speeds
	let keysClicked = 0;
	let keyElements = [];
	for (let i = 0; i < symbolCount; ++i) {
		const el = document.createElement('div');
		el.className = 'symbol';
		el.classList.add(i < keyCount ? 'key' : 'x');
		// Random initial position
		el.dataset.baseX = Math.random() * 100; // vw
		el.dataset.baseY = Math.random() * 100; // vh
		// Random direction and speed
		el.dataset.angle = Math.random() * Math.PI * 2; // radians
		el.dataset.speed = minSpeed + Math.random() * (maxSpeed - minSpeed); // vw/vh per 10s
		el.dataset.drift = (Math.random() - 0.5) * 0.5; // slight drift
		el.dataset.glowing = "0";
		if (el.classList.contains('key')) {
			el.dataset.shaken = "0";
			el.innerHTML = keySVG;
			keyElements.push(el);
			el.addEventListener('click', function (e) {
				e.stopPropagation();
				if (el.dataset.glowing !== "1") {
					keysClicked++;
				}
				el.dataset.glowing = "1";
				if (el.dataset.shaken === "0") {
					el.classList.add('shake');
					el.classList.add('glowing');
					el.dataset.shaken = "1";
				}
				// If both keys are clicked, trigger symbols to center
				if (keysClicked === keyCount) {
					triggerSymbolsToCenter();
				}
			});
			el.addEventListener('animationend', function (e) {
				if (e.animationName === 'key-shake') {
					el.classList.remove('shake');
				}
			});
		}
		background.appendChild(el);
		symbols.push(el);
	}

	let toCenter = false;
	let toCenterStart = 0;
	let flashShown = false;
	let flashDiv = null;
	function triggerSymbolsToCenter() {
		toCenter = true;
		toCenterStart = performance.now();
		// Add flash overlay
		flashDiv = document.createElement('div');
		flashDiv.style.position = 'fixed';
		flashDiv.style.left = 0;
		flashDiv.style.top = 0;
		flashDiv.style.width = '100vw';
		flashDiv.style.height = '100vh';
		flashDiv.style.background = 'radial-gradient(circle at 50% 50%, #fff 0%, #fff8 50%, #fff0 200%)';
		flashDiv.style.opacity = 0;
		flashDiv.style.pointerEvents = 'none';
		flashDiv.style.zIndex = 1000;
		document.body.appendChild(flashDiv);
	}

	// --- Illumination and tilt ---
	const questionMark = document.getElementById('question-mark');
	const theEnd = document.getElementById('the-end');
	let mouse = { x: -1000, y: -1000 };

	function updateIllumination(e) {
		mouse.x = e.clientX;
		mouse.y = e.clientY;
	}
	window.addEventListener('mousemove', updateIllumination);

	// Animate floating symbols and illumination
	function animate() {
		const now = Date.now() / 1000;
		// Animate symbols
		symbols.forEach((el, i) => {
			let baseX = parseFloat(el.dataset.baseX);
			let baseY = parseFloat(el.dataset.baseY);
			let angle = parseFloat(el.dataset.angle);
			let speed = parseFloat(el.dataset.speed); // vw/vh per 10s
			let drift = parseFloat(el.dataset.drift);
			let x = ((baseX + Math.cos(angle) * speed * now + Math.sin(now * drift) * 2) % 100 + 100) % 100;
			let y = ((baseY + Math.sin(angle) * speed * now + Math.cos(now * drift) * 2) % 100 + 100) % 100;
			// Get center of symbol
			const rect = el.getBoundingClientRect();
			let gradX = ((mouse.x - rect.left) / rect.width) * 100;
			let gradY = ((mouse.y - rect.top) / rect.height) * 100;
			// Key glowing logic
			if (el.classList.contains('key') && el.dataset.glowing === "1") {
				gradX = 50;
				gradY = 50;
			}
			if (toCenter) {
				// Animate flying to center from current position
				const elapsed = (performance.now() - toCenterStart) / 1000;
				// Center in vw/vh
				const centerX = 50;
				const centerY = 50;
				// Exponential ease-in, longer duration
				const t = Math.min(1, elapsed / toCenterDuration * lerp(0.95, 1.5, (speed - minSpeed) / (maxSpeed - minSpeed)));
				// Normalized ease for lerp
				const w = Math.pow(t, 2.5);
				x = lerp(x, centerX, w);
				y = lerp(y, centerY, w);
				gradX = lerp(gradX, 50, Math.min(1, w * 5));
				gradY = lerp(gradY, 50, Math.min(1, w * 5));
				// Fade out at the end
				if (t > 0.95) {
					el.style.opacity = 1 - (t - 0.95) / 0.05;
				}
			}
			// Move in a straight (but slightly drifting) direction, wrap at edges
			el.style.left = x + 'vw';
			el.style.top = y + 'vh';
			el.style.setProperty('--grad-x', gradX + '%');
			el.style.setProperty('--grad-y', gradY + '%');
		});
		// Flash effect
		if (toCenter && flashDiv) {
			const elapsed = (performance.now() - toCenterStart) / 1000;
			if (elapsed > toCenterDuration - 1 && !flashShown) {
				flashDiv.style.transition = 'opacity 1s';
				flashDiv.style.opacity = 1;
				flashShown = true;
			}
			if (elapsed > toCenterDuration) {
				// Remove all symbols and flash
				symbols.forEach(el => el.remove());
				flashDiv.style.transition = 'opacity 2s';
				flashDiv.style.opacity = 0;
				setTimeout(() => { if (flashDiv) flashDiv.remove(); }, 2100);
				toCenter = false;
				startLetterCycle();
				// Cycle each letter of 'Is this The End' after the flash
				function startLetterCycle() {
					const allSpans = Array.from(document.querySelectorAll('#the-end span'));
					const cycles = [];
					allSpans.forEach(span => {
						const orig = span.textContent;
						let code = orig.charCodeAt(0);
						let isUpper = orig === orig.toUpperCase();
						let start = code;
						let min = isUpper ? 65 : 97;
						let max = isUpper ? 90 : 122;
						cycles.push({ span, code, start, min, max, isUpper });
					});
					let delay = 40;
					function tick() {
						cycles.forEach(c => {
							c.code = c.code + 1;
							if (c.code > c.max) c.code = c.min;
							c.span.textContent = String.fromCharCode(c.code);
						});
						delay = Math.min(delay * 1.05, 1000); // increase delay up to 1s
						setTimeout(tick, delay);
					}
					tick();
				}
			}
		}
		// Question mark gradual illumination only
		const qmRect = questionMark.getBoundingClientRect();
		const qmx = qmRect.left + qmRect.width / 2;
		const qmy = qmRect.top + qmRect.height / 2;
		const dist = Math.hypot(mouse.x - qmx, mouse.y - qmy);
		let t = 1 - Math.min(Math.max((dist - 30) / 100, 0), 1);
		let gradX = ((mouse.x - qmRect.left) / qmRect.width) * 100;
		let gradY = ((mouse.y - qmRect.top) / qmRect.height) * 100;
		questionMark.style.setProperty('--grad-x', gradX + '%');
		questionMark.style.setProperty('--grad-y', gradY + '%');
		// Set radial gradient center for #the-end
		const teRect = theEnd.getBoundingClientRect();
		let teGradX = ((mouse.x - teRect.left) / teRect.width) * 100;
		let teGradY = ((mouse.y - teRect.top) / teRect.height) * 100;
		theEnd.style.setProperty('--grad-x', teGradX + '%');
		theEnd.style.setProperty('--grad-y', teGradY + '%');
		requestAnimationFrame(animate);
	}
	animate();
});
