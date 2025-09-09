
// Confetti animation
let confettiAnimationId = null;
let confettiFallId = null;
const confettiColors = [
	'#43c6ac', '#f8ffae', '#ffb6b9', '#f6d365', '#96fbc4', '#f9f586', '#a1c4fd', '#c2e9fb', '#fcb69f', '#ffdde1'
];

function randomBetween(a, b) {
	return a + Math.random() * (b - a);
}

function throwConfetti() {
	// Cancel any previous animation
	if (confettiAnimationId) {
		cancelAnimationFrame(confettiAnimationId);
		confettiAnimationId = null;
	}
	if (confettiFallId) {
		cancelAnimationFrame(confettiFallId);
		confettiFallId = null;
	}
	const canvas = document.getElementById('confetti-canvas');
	const ctx = canvas.getContext('2d');
	const W = window.innerWidth;
	const H = window.innerHeight;
	canvas.width = W;
	canvas.height = H;

	let confetti = [];
	for (let i = 0; i < 120; i++) {
		confetti.push({
			x: randomBetween(0, W),
			y: randomBetween(-H, 0),
			r: randomBetween(6, 14),
			d: randomBetween(2, 8),
			color: confettiColors[Math.floor(Math.random() * confettiColors.length)],
			tilt: randomBetween(-10, 10),
			tiltAngle: 0,
			tiltAngleIncremental: randomBetween(0.05, 0.12)
		});
	}

	function draw() {
		ctx.clearRect(0, 0, W, H);
		confetti.forEach(c => {
			ctx.beginPath();
			ctx.lineWidth = c.r;
			ctx.strokeStyle = c.color;
			ctx.moveTo(c.x + c.tilt + c.r / 3, c.y);
			ctx.lineTo(c.x + c.tilt, c.y + c.d * 2);
			ctx.stroke();
		});
		update();
	}

	function update() {
		for (let i = 0; i < confetti.length; i++) {
			let c = confetti[i];
			c.y += (Math.cos(c.d) + 2 + c.d / 2);
			c.x += Math.sin(0.5) * 2;
			c.tiltAngle += c.tiltAngleIncremental;
			c.tilt = Math.sin(c.tiltAngle) * 15;
			// Only respawn during the burst phase
			if (c.y > H && frame < 120) {
				confetti[i] = {
					x: randomBetween(0, W),
					y: randomBetween(-20, 0),
					r: c.r,
					d: c.d,
					color: c.color,
					tilt: randomBetween(-10, 10),
					tiltAngle: 0,
					tiltAngleIncremental: c.tiltAngleIncremental
				};
			}
			// After burst, let them fall off screen (do not respawn)
		}
	}

	let frame = 0;
	function animate() {
		draw();
		frame++;
		if (frame < 120) { // Shorter burst, then let them fall
			confettiAnimationId = requestAnimationFrame(animate);
		} else {
			// Let confetti keep falling until all are off screen
			function fallOffScreen() {
				draw();
				if (confetti.every(c => c.y > H + 30)) {
					ctx.clearRect(0, 0, W, H);
					confettiFallId = null;
				} else {
					confettiFallId = requestAnimationFrame(fallOffScreen);
				}
			}
			fallOffScreen();
			confettiAnimationId = null;
		}
	}
	animate();
}

// Auto confetti on load
window.onload = throwConfetti;
